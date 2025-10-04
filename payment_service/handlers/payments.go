package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
	"github.com/maisarasherif/order-processing-microservices/payment_service/repository"
)

// Payments is the handler for all payment-related operations
type Payments struct {
	l    *log.Logger
	repo repository.PaymentRepository
}

// NewPaymentsWithRepository creates a new payment handler with repository
func NewPaymentsWithRepository(l *log.Logger, repo repository.PaymentRepository) *Payments {
	return &Payments{
		l:    l,
		repo: repo,
	}
}

// PaymentResponse is the standard response structure
type PaymentResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Payment *data.Payment `json:"payment,omitempty"`
}

// PaymentsListResponse for listing multiple payments
type PaymentsListResponse struct {
	Success  bool            `json:"success"`
	Message  string          `json:"message"`
	Payments []*data.Payment `json:"payments"`
	Count    int             `json:"count"`
}

// GetPayments handles GET /payments - returns all payments
func (p *Payments) GetPayments(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payments")
	w.Header().Set("Content-Type", "application/json")

	// Get all payments from database
	payments, err := p.repo.GetAll()
	if err != nil {
		p.l.Println("[ERROR] retrieving payments:", err)
		response := PaymentsListResponse{
			Success: false,
			Message: "Failed to retrieve payments",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentsListResponse{
		Success:  true,
		Message:  "Payments retrieved successfully",
		Payments: payments,
		Count:    len(payments),
	}

	json.NewEncoder(w).Encode(response)
}

// GetPayment handles GET /payments/{id} - returns a specific payment
func (p *Payments) GetPayment(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payment by ID")
	w.Header().Set("Content-Type", "application/json")

	// Extract payment ID from URL
	vars := mux.Vars(r)
	paymentID := vars["id"]

	// Retrieve payment from database
	payment, err := p.repo.GetByID(paymentID)

	if err == data.ErrPaymentNotFound {
		response := PaymentResponse{
			Success: false,
			Message: "Payment not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		p.l.Println("[ERROR] retrieving payment:", err)
		response := PaymentResponse{
			Success: false,
			Message: "Error retrieving payment",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := PaymentResponse{
		Success: true,
		Message: "Payment retrieved successfully",
		Payment: payment,
	}

	json.NewEncoder(w).Encode(response)
}

// ProcessPayment handles POST /payments - creates and processes a new payment
func (p *Payments) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle POST Payment - Process Payment")
	w.Header().Set("Content-Type", "application/json")

	// Get validated payment from context
	payment := r.Context().Value(KeyPayment{}).(data.Payment)

	// Check for duplicate idempotency key
	existingPayment, err := p.repo.GetByIdempotencyKey(payment.IdempotencyKey)
	if err == nil {
		// Payment with this idempotency key already exists
		p.l.Printf("Duplicate idempotency key: %s, returning existing payment\n", payment.IdempotencyKey)
		response := PaymentResponse{
			Success: true,
			Message: "Payment already processed (idempotent request)",
			Payment: existingPayment,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate payment ID and set initial values
	payment.ID = data.GeneratePaymentID()
	payment.Status = data.PaymentStatusPending
	payment.CreatedAt = data.Now()

	// Create payment in database
	err = p.repo.Create(&payment)
	if err != nil {
		p.l.Println("[ERROR] creating payment:", err)
		response := PaymentResponse{
			Success: false,
			Message: "Failed to create payment",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Process the payment (simulate payment gateway)
	processedPayment := data.ProcessPaymentGateway(&payment)

	// Update payment status in database
	err = p.repo.UpdateStatus(&processedPayment)
	if err != nil {
		p.l.Println("[ERROR] updating payment status:", err)
		response := PaymentResponse{
			Success: false,
			Message: "Payment created but failed to process",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Send response based on payment status
	if processedPayment.Status == data.PaymentStatusCompleted {
		response := PaymentResponse{
			Success: true,
			Message: "Payment processed successfully",
			Payment: &processedPayment,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		response := PaymentResponse{
			Success: false,
			Message: fmt.Sprintf("Payment failed: %s", processedPayment.ErrorMessage),
			Payment: &processedPayment,
		}
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(response)
	}

	p.l.Printf("Payment %s processed with status: %s\n", processedPayment.ID, processedPayment.Status)
}

// GetStatistics handles GET /payments/stats - returns payment statistics
func (p *Payments) GetStatistics(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payment Statistics")
	w.Header().Set("Content-Type", "application/json")

	stats, err := p.repo.GetStatistics()
	if err != nil {
		p.l.Println("[ERROR] retrieving statistics:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve statistics",
		})
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"message":    "Statistics retrieved successfully",
		"statistics": stats,
	}

	json.NewEncoder(w).Encode(response)
}

// KeyPayment is the context key for validated payment data
type KeyPayment struct{}

// MiddlewarePaymentValidation validates incoming payment data
func (p *Payments) MiddlewarePaymentValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payment := data.Payment{}

		// Decode JSON from request body
		err := payment.FromJSON(r.Body)
		if err != nil {
			p.l.Println("[ERROR] deserializing payment", err)
			http.Error(w, "Error reading payment data", http.StatusBadRequest)
			return
		}

		// Validate the payment data
		err = payment.Validate()
		if err != nil {
			p.l.Println("[ERROR] validating payment", err)
			http.Error(
				w,
				fmt.Sprintf("Error validating payment: %s", err),
				http.StatusBadRequest,
			)
			return
		}

		// Add validated payment to context
		ctx := context.WithValue(r.Context(), KeyPayment{}, payment)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
