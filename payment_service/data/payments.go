package data

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Payment status constants
const (
	PaymentStatusPending    = "pending"
	PaymentStatusProcessing = "processing"
	PaymentStatusCompleted  = "completed"
	PaymentStatusFailed     = "failed"
)

// Error types
var (
	ErrPaymentNotFound         = fmt.Errorf("payment not found")
	ErrPaymentAlreadyProcessed = fmt.Errorf("payment already processed")
	ErrInvalidAmount           = fmt.Errorf("invalid payment amount")
	ErrDuplicateIdempotencyKey = fmt.Errorf("duplicate idempotency key")
)

// Payment represents a payment transaction
type Payment struct {
	ID             string     `json:"id"`
	OrderID        string     `json:"order_id" validate:"required"`
	Amount         float64    `json:"amount" validate:"required,gt=0"`
	Currency       string     `json:"currency" validate:"required,currency"`
	Status         string     `json:"status"`
	Method         string     `json:"method" validate:"required,payment_method"`
	CustomerID     string     `json:"customer_id" validate:"required"`
	IdempotencyKey string     `json:"idempotency_key" validate:"required"`
	TransactionID  string     `json:"transaction_id,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	FailedAt       *time.Time `json:"failed_at,omitempty"`
}

// FromJSON deserializes a payment from JSON
func (p *Payment) FromJSON(r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(p)
}

// Validate checks if the payment data is valid
func (p *Payment) Validate() error {
	validate := validator.New()
	validate.RegisterValidation("currency", validateCurrency)
	validate.RegisterValidation("payment_method", validatePaymentMethod)
	return validate.Struct(p)
}

// validateCurrency checks if currency code is valid
func validateCurrency(fl validator.FieldLevel) bool {
	validCurrencies := map[string]bool{
		"USD": true,
		"EUR": true,
		"GBP": true,
		"AED": true,
		"SAR": true,
		"EGP": true,
	}
	return validCurrencies[fl.Field().String()]
}

// validatePaymentMethod checks if payment method is supported
func validatePaymentMethod(fl validator.FieldLevel) bool {
	validMethods := map[string]bool{
		"credit_card":   true,
		"debit_card":    true,
		"paypal":        true,
		"bank_transfer": true,
		"apple_pay":     true,
	}
	return validMethods[fl.Field().String()]
}

// Payments is a collection of Payment pointers
type Payments []*Payment

// ToJSON serializes payments to JSON
func (p *Payments) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p)
}

// GeneratePaymentID creates a unique payment ID using UUID
func GeneratePaymentID() string {
	return "pay_" + uuid.New().String()
}

// Now returns current UTC time
func Now() time.Time {
	return time.Now().UTC()
}

// ProcessPaymentGateway simulates payment gateway processing
func ProcessPaymentGateway(payment *Payment) Payment {
	payment.Status = PaymentStatusProcessing

	// Simulate gateway call
	success := simulatePaymentGateway(payment)

	now := Now()
	if success {
		payment.Status = PaymentStatusCompleted
		payment.ProcessedAt = &now
		payment.TransactionID = fmt.Sprintf("txn_%d", time.Now().Unix())
	} else {
		payment.Status = PaymentStatusFailed
		payment.FailedAt = &now
		payment.ErrorMessage = "Payment declined by gateway"
	}

	return *payment
}

// simulatePaymentGateway simulates calling a payment gateway
func simulatePaymentGateway(payment *Payment) bool {
	time.Sleep(100 * time.Millisecond)

	// Test scenarios
	if payment.Amount == 666 || payment.Amount > 10000 {
		return false
	}

	return true
}
