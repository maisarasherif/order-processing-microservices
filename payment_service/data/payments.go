package data

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	PaymentStatusPending    = "pending"
	PaymentStatusProcessing = "processing"
	PaymentStatusCompleted  = "completed"
	PaymentStatusFailed     = "failed"
)

var (
	ErrPaymentNotFound         = fmt.Errorf("payment not found")
	ErrPaymentAlreadyProcessed = fmt.Errorf("payment already processed")
	ErrInvalidAmount           = fmt.Errorf("invalid payment amount")
	ErrDuplicateIdempotencyKey = fmt.Errorf("duplicate idempotency key")
)

// core domain model
type Payment struct {
	ID string `json:"id"`

	OrderID string `json:"order_id" validate:"required"`

	Amount   float64 `json:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"required,currency"`

	Status string `json:"status"`

	Method string `json:"method" validate:"required,payment_method"`

	CustomerID string `json:"customer_id" validate:"required"`

	// Idempotency key to prevent duplicate payments
	IdempotencyKey string `json:"idempotency_key" validate:"required"`

	TransactionID string `json:"transaction_id,omitempty"`

	ErrorMessage string `json:"error_message,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
}

func (p *Payment) FromJSON(r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(p)
}

func (p *Payment) Validate() error {
	validate := validator.New()

	validate.RegisterValidation("currency", validateCurrency)
	validate.RegisterValidation("payment_method", validatePaymentMethod)

	return validate.Struct(p)
}

func validateCurrency(fl validator.FieldLevel) bool {
	validCurrencies := map[string]bool{
		"EGP": true,
		"USD": true,
		"EUR": true,
		"GBP": true,
		"AED": true,
		"SAR": true,
	}

	currency := fl.Field().String()
	return validCurrencies[currency]
}

func validatePaymentMethod(fl validator.FieldLevel) bool {
	validMethods := map[string]bool{
		"credit_card":   true,
		"debit_card":    true,
		"apple_pay":     true,
		"bank_transfer": true,
	}

	method := fl.Field().String()
	return validMethods[method]
}

type Payments []*Payment

func (p *Payments) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p)
}

var (
	paymentList = []*Payment{}
	nextID      = 1
	mu          sync.RWMutex

	// Map to track idempotency keys -> payment ID
	idempotencyMap = make(map[string]string)
)

func generateID() string {
	id := fmt.Sprintf("pay_%d", nextID)
	nextID++
	return id
}

func GetPayments() Payments {
	mu.RLock()
	defer mu.RUnlock()

	result := make(Payments, len(paymentList))
	copy(result, paymentList)
	return result
}

func CreatePayment(p *Payment) error {
	mu.Lock()
	defer mu.Unlock()

	// Check for duplicate idempotency key
	if existingPaymentID, exists := idempotencyMap[p.IdempotencyKey]; exists {
		// Return the existing payment instead of creating a duplicate
		existingPayment, _ := getPaymentByID(existingPaymentID)
		*p = *existingPayment
		return ErrDuplicateIdempotencyKey
	}

	p.ID = generateID()
	p.Status = PaymentStatusPending
	p.CreatedAt = time.Now().UTC()

	paymentList = append(paymentList, p)

	// Track idempotency key
	idempotencyMap[p.IdempotencyKey] = p.ID

	return nil
}

func ProcessPayment(paymentID string) error {
	mu.Lock()
	defer mu.Unlock()

	payment, index, err := findPayment(paymentID)
	if err != nil {
		return err
	}

	if payment.Status == PaymentStatusCompleted {
		return ErrPaymentAlreadyProcessed
	}

	payment.Status = PaymentStatusProcessing

	success := simulatePaymentGateway(payment)

	now := time.Now().UTC()
	if success {
		payment.Status = PaymentStatusCompleted
		payment.ProcessedAt = &now
		payment.TransactionID = fmt.Sprintf("txn_%d", time.Now().Unix())
	} else {
		payment.Status = PaymentStatusFailed
		payment.FailedAt = &now
		payment.ErrorMessage = "Payment declined by gateway"
	}

	paymentList[index] = payment

	return nil
}

func simulatePaymentGateway(payment *Payment) bool {
	// Simulate processing delay
	time.Sleep(100 * time.Millisecond)

	// Test scenarios:
	// - Amount 666: Always fail (for testing)
	// - Amount > 10000: Fail (suspicious transaction)
	// - Otherwise: Success
	if payment.Amount == 666 || payment.Amount > 10000 {
		return false
	}

	return true
}

func GetPaymentByID(id string) (*Payment, error) {
	mu.RLock()
	defer mu.RUnlock()
	return getPaymentByID(id)
}

func getPaymentByID(id string) (*Payment, error) {
	for _, p := range paymentList {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, ErrPaymentNotFound
}

func findPayment(id string) (*Payment, int, error) {
	for i, p := range paymentList {
		if p.ID == id {
			return p, i, nil
		}
	}
	return nil, -1, ErrPaymentNotFound
}

func GetPaymentsByOrderID(orderID string) Payments {
	mu.RLock()
	defer mu.RUnlock()

	var result Payments
	for _, p := range paymentList {
		if p.OrderID == orderID {
			result = append(result, p)
		}
	}
	return result
}

func GetPaymentByIdempotencyKey(key string) (*Payment, error) {
	mu.RLock()
	defer mu.RUnlock()

	if paymentID, exists := idempotencyMap[key]; exists {
		return getPaymentByID(paymentID)
	}
	return nil, ErrPaymentNotFound
}

// GeneratePaymentID is exported for handlers
func GeneratePaymentID() string {
	mu.Lock()
	defer mu.Unlock()
	return generateID()
}

// Now returns current UTC time
func Now() time.Time {
	return time.Now().UTC()
}

// ProcessPaymentGateway processes payment and returns updated payment
func ProcessPaymentGateway(payment *Payment) Payment {
	payment.Status = PaymentStatusProcessing
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
