package data

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Order status constants - these are like enum values in other languages
const (
	OrderStatusPending   = "pending"   // Order created, waiting for payment
	OrderStatusPaid      = "paid"      // Payment successful
	OrderStatusConfirmed = "confirmed" // Order confirmed by system
	OrderStatusCancelled = "cancelled" // Order cancelled
	OrderStatusFailed    = "failed"    // Payment failed
)

// Error types - Go uses errors as values, not exceptions
var (
	ErrOrderNotFound = fmt.Errorf("order not found")
	ErrInvalidTotal  = fmt.Errorf("invalid order total")
)

// OrderItem represents a single item in an order
// The `json:"..."` tags tell Go how to convert this to/from JSON
// The `validate:"..."` tags define validation rules
type OrderItem struct {
	ProductID   string  `json:"product_id" validate:"required"`
	ProductName string  `json:"product_name" validate:"required"`
	Quantity    int     `json:"quantity" validate:"required,gt=0"` // gt=0 means greater than 0
	UnitPrice   float64 `json:"unit_price" validate:"required,gt=0"`
	Subtotal    float64 `json:"subtotal"`
}

// Order represents a customer order
type Order struct {
	ID              string      `json:"id"`
	CustomerID      string      `json:"customer_id" validate:"required"`
	CustomerEmail   string      `json:"customer_email" validate:"required,email"`
	Items           []OrderItem `json:"items" validate:"required,min=1,dive"` // dive validates each item
	TotalAmount     float64     `json:"total_amount"`
	Currency        string      `json:"currency" validate:"required,currency"`
	Status          string      `json:"status"`
	PaymentID       string      `json:"payment_id,omitempty"` // omitempty = don't include if empty
	PaymentMethod   string      `json:"payment_method" validate:"required,payment_method"`
	ShippingAddress string      `json:"shipping_address" validate:"required"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// FromJSON deserializes an order from JSON
// In Go, methods are functions with a receiver (o *Order)
// The * means it's a pointer - we can modify the original order
func (o *Order) FromJSON(r io.Reader) error {
	// decoder reads JSON from the io.Reader (like http.Request.Body)
	decoder := json.NewDecoder(r)
	// Decode() fills our order struct with data from JSON
	return decoder.Decode(o)
}

// ToJSON serializes an order to JSON
// This time we don't need a pointer because we're only reading, not modifying
func (o *Order) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(o)
}

// Validate checks if the order data is valid
func (o *Order) Validate() error {
	// Create a new validator
	validate := validator.New()

	// Register custom validation functions
	// These tell the validator how to check our custom rules
	validate.RegisterValidation("currency", validateCurrency)
	validate.RegisterValidation("payment_method", validatePaymentMethod)

	// Validate the struct
	return validate.Struct(o)
}

// CalculateTotal computes the total from all items
// This is a method that modifies the order (notice the pointer receiver)
func (o *Order) CalculateTotal() {
	var total float64 = 0

	// Range is Go's way to iterate over slices (like arrays)
	// The first value (i) is the index, but we don't need it, so we use _
	// The second value is the actual item
	for _, item := range o.Items {
		// Calculate subtotal for each item
		item.Subtotal = float64(item.Quantity) * item.UnitPrice
		total += item.Subtotal
	}

	o.TotalAmount = total
}

// validateCurrency checks if currency code is valid
// validator.FieldLevel is an interface that gives us access to the field being validated
func validateCurrency(fl validator.FieldLevel) bool {
	// map[string]bool is like a Set or Dictionary in other languages
	// map[KeyType]ValueType
	validCurrencies := map[string]bool{
		"USD": true,
		"EUR": true,
		"GBP": true,
		"AED": true,
	}

	// fl.Field() gets the field value, .String() converts it to string
	// The second return value (ok) tells us if the key exists in the map
	_, ok := validCurrencies[fl.Field().String()]
	return ok
}

// validatePaymentMethod checks if payment method is supported
func validatePaymentMethod(fl validator.FieldLevel) bool {
	validMethods := map[string]bool{
		"credit_card":   true,
		"debit_card":    true,
		"paypal":        true,
		"bank_transfer": true,
	}
	_, ok := validMethods[fl.Field().String()]
	return ok
}

// Orders is a collection of Order pointers
// []*Order means a slice (dynamic array) of pointers to Order
type Orders []*Order

// ToJSON serializes multiple orders to JSON
func (o *Orders) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(o)
}

// GenerateOrderID creates a unique order ID using UUID
// uuid.New() generates a random UUID, .String() converts it to text
func GenerateOrderID() string {
	return "ord_" + uuid.New().String()
}

// GenerateIdempotencyKey creates a unique key for payment idempotency
// Idempotency means we can make the same request multiple times safely
// It's like clicking "submit payment" twice - it should only charge once
func GenerateIdempotencyKey(orderID string) string {
	return "idem_" + orderID + "_" + uuid.New().String()
}

// Now returns current UTC time
func Now() time.Time {
	return time.Now().UTC()
}
