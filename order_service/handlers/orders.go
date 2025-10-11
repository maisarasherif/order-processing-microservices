package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/maisarasherif/order-processing-microservices/order_service/data"
	"github.com/maisarasherif/order-processing-microservices/order_service/payment"
	"github.com/maisarasherif/order-processing-microservices/order_service/repository"
)

type Orders struct {
	l             *log.Logger
	repo          repository.OrderRepository
	paymentClient *payment.Client
}

func NewOrdersHandler(l *log.Logger, repo repository.OrderRepository, paymentClient *payment.Client) *Orders {
	return &Orders{
		l:             l,
		repo:          repo,
		paymentClient: paymentClient,
	}
}

// ============ RESPONSE STRUCTURES ============

type SuccessResponse struct {
	Data interface{} `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(SuccessResponse{Data: data})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// ============ PRODUCT ENDPOINTS ============

func (o *Orders) GetProducts(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Products")

	products, err := o.repo.GetAvailableProducts()
	if err != nil {
		o.l.Println("[ERROR] retrieving products:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve products")
		return
	}

	respondJSON(w, http.StatusOK, products)
}

func (o *Orders) GetProduct(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Product by ID")

	vars := mux.Vars(r)
	productID := vars["id"]

	product, err := o.repo.GetProduct(productID)
	if err == data.ErrProductNotFound {
		respondError(w, http.StatusNotFound, "Product not found")
		return
	}
	if err != nil {
		o.l.Println("[ERROR] retrieving product:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve product")
		return
	}

	respondJSON(w, http.StatusOK, product)
}

// ============ ORDER ENDPOINTS ============

func (o *Orders) GetOrders(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Orders")

	orders, err := o.repo.GetAll()
	if err != nil {
		o.l.Println("[ERROR] retrieving orders:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve orders")
		return
	}

	respondJSON(w, http.StatusOK, orders)
}

func (o *Orders) GetOrder(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Order by ID")

	vars := mux.Vars(r)
	orderID := vars["id"]

	order, err := o.repo.GetByID(orderID)
	if err == data.ErrOrderNotFound {
		respondError(w, http.StatusNotFound, "Order not found")
		return
	}
	if err != nil {
		o.l.Println("[ERROR] retrieving order:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve order")
		return
	}

	respondJSON(w, http.StatusOK, order)
}

func (o *Orders) CreateOrder(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle POST Order - Create Order with Payment")

	// Get validated request from context
	orderReq := r.Context().Value(KeyOrderRequest{}).(data.CreateOrderRequest)

	// Build order with server-validated prices
	order, err := o.buildOrderFromRequest(orderReq)
	if err != nil {
		o.l.Println("[ERROR] building order:", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Generate order ID and set initial state
	order.ID = data.GenerateOrderID()
	order.Status = data.OrderStatusPending
	order.CreatedAt = data.Now()
	order.UpdatedAt = data.Now()

	// Validate total amount
	if order.TotalAmount <= 0 {
		respondError(w, http.StatusBadRequest, "Invalid order total")
		return
	}

	// Create order in database
	if err := o.repo.Create(order); err != nil {
		o.l.Println("[ERROR] creating order:", err)
		respondError(w, http.StatusInternalServerError, "Failed to create order")
		return
	}

	o.l.Printf("Order %s created, initiating payment...\n", order.ID)

	// Process payment
	paymentReq := payment.PaymentRequest{
		OrderID:        order.ID,
		Amount:         order.TotalAmount,
		Currency:       order.Currency,
		Method:         order.PaymentMethod,
		CustomerID:     order.CustomerID,
		IdempotencyKey: data.GenerateIdempotencyKey(order.ID),
	}

	paymentResp, err := o.paymentClient.ProcessPayment(paymentReq)
	if err != nil {
		o.l.Println("[ERROR] processing payment:", err)
		o.repo.UpdateStatus(order.ID, data.OrderStatusFailed)
		order.Status = data.OrderStatusFailed
		respondError(w, http.StatusServiceUnavailable, "Payment service unavailable")
		return
	}

	// Update order with payment info
	order.PaymentID = paymentResp.Payment.ID
	if err := o.repo.UpdatePaymentID(order.ID, paymentResp.Payment.ID); err != nil {
		o.l.Printf("[ERROR] updating payment ID: %v\n", err)
	}

	// Handle payment result
	if paymentResp.Payment.Status == "completed" {
		if err := o.repo.UpdateStatus(order.ID, data.OrderStatusPaid); err != nil {
			o.l.Printf("[CRITICAL] Order %s paid but status update failed: %v\n", order.ID, err)
		}
		order.Status = data.OrderStatusPaid

		o.l.Printf("Order %s paid successfully with payment %s\n", order.ID, paymentResp.Payment.ID)
		respondJSON(w, http.StatusCreated, order)
	} else {
		if err := o.repo.UpdateStatus(order.ID, data.OrderStatusFailed); err != nil {
			o.l.Printf("[ERROR] Failed to update failed order status: %v\n", err)
		}
		order.Status = data.OrderStatusFailed

		o.l.Printf("Order %s payment failed: %s\n", order.ID, paymentResp.Payment.ErrorMessage)
		respondError(w, http.StatusPaymentRequired, "Payment failed: "+paymentResp.Payment.ErrorMessage)
	}
}

func (o *Orders) GetCustomerOrders(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Customer Orders")

	vars := mux.Vars(r)
	customerID := vars["id"]

	orders, err := o.repo.GetByCustomerID(customerID)
	if err != nil {
		o.l.Println("[ERROR] retrieving customer orders:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve customer orders")
		return
	}

	respondJSON(w, http.StatusOK, orders)
}

// ============ HELPER FUNCTIONS ============

// buildOrderFromRequest validates products and builds order with server prices
func (o *Orders) buildOrderFromRequest(req data.CreateOrderRequest) (*data.Order, error) {
	order := &data.Order{
		CustomerID:      req.CustomerID,
		CustomerEmail:   req.CustomerEmail,
		Currency:        req.Currency,
		PaymentMethod:   req.PaymentMethod,
		ShippingAddress: req.ShippingAddress,
		Items:           make([]data.OrderItem, 0, len(req.Items)),
	}

	// Validate each item and fetch server prices
	for _, itemReq := range req.Items {
		// Fetch product from database
		product, err := o.repo.GetProduct(itemReq.ProductID)
		if err == data.ErrProductNotFound {
			return nil, fmt.Errorf("product not found: %s", itemReq.ProductID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to fetch product %s: %w", itemReq.ProductID, err)
		}

		// Check if product is available
		if !product.Available {
			return nil, fmt.Errorf("product unavailable: %s", product.Name)
		}

		// Build order item with SERVER price (ignore any client price)
		orderItem := data.OrderItem{
			ProductID:   product.ID,
			ProductName: product.Name,
			Quantity:    itemReq.Quantity,
			UnitPrice:   product.Price, // ← SERVER decides price!
			Subtotal:    product.Price * float64(itemReq.Quantity),
		}

		order.Items = append(order.Items, orderItem)
	}

	// Calculate total
	order.CalculateTotal()

	return order, nil
}

// ============ MIDDLEWARE ============

type KeyOrderRequest struct{}

func (o *Orders) MiddlewareOrderValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orderReq := data.CreateOrderRequest{}

		// Decode JSON from request body
		err := orderReq.FromJSON(r.Body)
		if err != nil {
			o.l.Println("[ERROR] deserializing order request:", err)
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate the request
		err = orderReq.Validate()
		if err != nil {
			o.l.Println("[ERROR] validating order request:", err)
			respondError(w, http.StatusBadRequest, "Validation error: "+err.Error())
			return
		}

		// Add validated request to context
		ctx := context.WithValue(r.Context(), KeyOrderRequest{}, orderReq)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
