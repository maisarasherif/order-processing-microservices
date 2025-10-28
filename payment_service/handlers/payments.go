package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
	"github.com/maisarasherif/order-processing-microservices/payment_service/logger"
	"github.com/maisarasherif/order-processing-microservices/payment_service/repository"
)

type Payments struct {
	l    *logger.StructuredLogger
	repo repository.PaymentRepository
}

func NewPaymentsWithRepository(l *logger.StructuredLogger, repo repository.PaymentRepository) *Payments {
	return &Payments{
		l:    l,
		repo: repo,
	}
}

// Standard response structures
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

// GetPayments handles GET /payments - returns all payments
func (p *Payments) GetPayments(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payments")

	payments, err := p.repo.GetAll()
	if err != nil {
		p.l.Println("[ERROR] retrieving payments:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve payments")
		return
	}

	respondJSON(w, http.StatusOK, payments)
}

// GetPayment handles GET /payments/{id} - returns a specific payment
func (p *Payments) GetPayment(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payment by ID")

	vars := mux.Vars(r)
	paymentID := vars["id"]

	payment, err := p.repo.GetByID(paymentID)

	if err == data.ErrPaymentNotFound {
		respondError(w, http.StatusNotFound, "Payment not found")
		return
	}

	if err != nil {
		p.l.Println("[ERROR] retrieving payment:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve payment")
		return
	}

	respondJSON(w, http.StatusOK, payment)
}

// ProcessPayment handles POST /payments - creates and processes a new payment
func (p *Payments) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle POST Payment - Process Payment")

	// Get validated payment from context
	payment := r.Context().Value(KeyPayment{}).(data.Payment)

	// Check for duplicate idempotency key
	existingPayment, err := p.repo.GetByIdempotencyKey(payment.IdempotencyKey)
	if err == nil {
		// Payment with this idempotency key already exists
		p.l.Printf("Duplicate idempotency key: %s, returning existing payment\n", payment.IdempotencyKey)
		respondJSON(w, http.StatusOK, existingPayment)
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
		respondError(w, http.StatusInternalServerError, "Failed to create payment")
		return
	}

	// Process the payment (simulate payment gateway)
	processedPayment := data.ProcessPaymentGateway(&payment)

	// Update payment status in database
	err = p.repo.UpdateStatus(&processedPayment)
	if err != nil {
		p.l.Println("[ERROR] updating payment status:", err)
		respondError(w, http.StatusInternalServerError, "Payment created but failed to process")
		return
	}

	// Send response based on payment status
	if processedPayment.Status == data.PaymentStatusCompleted {
		p.l.Printf("Payment %s processed successfully\n", processedPayment.ID)
		respondJSON(w, http.StatusOK, &processedPayment)
	} else {
		p.l.Printf("Payment %s failed: %s\n", processedPayment.ID, processedPayment.ErrorMessage)
		respondError(w, http.StatusPaymentRequired, "Payment declined: "+processedPayment.ErrorMessage)
	}
}

// GetStatistics handles GET /payments/stats - returns payment statistics
func (p *Payments) GetStatistics(w http.ResponseWriter, r *http.Request) {
	p.l.Println("Handle GET Payment Statistics")

	stats, err := p.repo.GetStatistics()
	if err != nil {
		p.l.Println("[ERROR] retrieving statistics:", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve statistics")
		return
	}

	respondJSON(w, http.StatusOK, stats)
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
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate the payment data
		err = payment.Validate()
		if err != nil {
			p.l.Println("[ERROR] validating payment", err)
			respondError(w, http.StatusBadRequest, "Validation error: "+err.Error())
			return
		}

		// Add validated payment to context
		ctx := context.WithValue(r.Context(), KeyPayment{}, payment)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
