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

type OrderResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Order   *data.Order `json:"order,omitempty"`
}

type OrdersListResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Orders  []*data.Order `json:"orders"`
	Count   int           `json:"count"`
}

func (o *Orders) GetOrders(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Orders")
	w.Header().Set("Content-Type", "application/json")

	orders, err := o.repo.GetAll()
	if err != nil {
		o.l.Println("[ERROR] retrieving orders:", err)
		response := OrdersListResponse{
			Success: false,
			Message: "Failed to retrieve orders",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := OrdersListResponse{
		Success: true,
		Message: "Orders retrieved successfully",
		Orders:  orders,
		Count:   len(orders),
	}

	json.NewEncoder(w).Encode(response)
}

func (o *Orders) GetOrder(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Order by ID")
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	orderID := vars["id"]

	order, err := o.repo.GetByID(orderID)

	if err == data.ErrOrderNotFound {
		response := OrderResponse{
			Success: false,
			Message: "Order not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		o.l.Println("[ERROR] retrieving order:", err)
		response := OrderResponse{
			Success: false,
			Message: "Error retrieving order",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := OrderResponse{
		Success: true,
		Message: "Order retrieved successfully",
		Order:   order,
	}

	json.NewEncoder(w).Encode(response)
}

func (o *Orders) CreateOrder(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle POST Order - Create Order")
	w.Header().Set("Content-Type", "application/json")

	order := r.Context().Value(KeyOrder{}).(data.Order)

	order.ID = data.GenerateOrderID()
	order.Status = data.OrderStatusPending
	order.CreatedAt = data.Now()
	order.UpdatedAt = data.Now()

	order.CalculateTotal()

	if order.TotalAmount <= 0 {
		response := OrderResponse{
			Success: false,
			Message: "Invalid order total",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := o.repo.Create(&order)
	if err != nil {
		o.l.Println("[ERROR] creating order:", err)
		response := OrderResponse{
			Success: false,
			Message: "Failed to create order",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	o.l.Printf("Order %s created, initiating payment...\n", order.ID)

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

		response := OrderResponse{
			Success: false,
			Message: "Payment service unavailable",
			Order:   &order,
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	order.PaymentID = paymentResp.Payment.ID
	o.repo.UpdatePaymentID(order.ID, paymentResp.Payment.ID)

	if paymentResp.Payment.Status == "completed" {

		o.repo.UpdateStatus(order.ID, data.OrderStatusPaid)
		order.Status = data.OrderStatusPaid

		o.l.Printf("Order %s paid successfully with payment %s\n",
			order.ID, paymentResp.Payment.ID)

		response := OrderResponse{
			Success: true,
			Message: "Order created and paid successfully",
			Order:   &order,
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	} else {

		o.repo.UpdateStatus(order.ID, data.OrderStatusFailed)
		order.Status = data.OrderStatusFailed

		o.l.Printf("Order %s payment failed: %s\n",
			order.ID, paymentResp.Payment.ErrorMessage)

		response := OrderResponse{
			Success: false,
			Message: fmt.Sprintf("Order created but payment failed: %s",
				paymentResp.Payment.ErrorMessage),
			Order: &order,
		}
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(response)
	}
}

func (o *Orders) GetCustomerOrders(w http.ResponseWriter, r *http.Request) {
	o.l.Println("Handle GET Customer Orders")
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	customerID := vars["id"]

	orders, err := o.repo.GetByCustomerID(customerID)
	if err != nil {
		o.l.Println("[ERROR] retrieving customer orders:", err)
		response := OrdersListResponse{
			Success: false,
			Message: "Failed to retrieve customer orders",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := OrdersListResponse{
		Success: true,
		Message: "Customer orders retrieved successfully",
		Orders:  orders,
		Count:   len(orders),
	}

	json.NewEncoder(w).Encode(response)
}

type KeyOrder struct{}

func (o *Orders) MiddlewareOrderValidation(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order := data.Order{}

		err := order.FromJSON(r.Body)
		if err != nil {
			o.l.Println("[ERROR] deserializing order", err)
			http.Error(w, "Error reading order data", http.StatusBadRequest)
			return
		}

		err = order.Validate()
		if err != nil {
			o.l.Println("[ERROR] validating order", err)
			http.Error(w,
				fmt.Sprintf("Error validating order: %s", err),
				http.StatusBadRequest,
			)
			return
		}

		ctx := context.WithValue(r.Context(), KeyOrder{}, order)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
