package data

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusConfirmed = "confirmed"
	OrderStatusCancelled = "cancelled"
	OrderStatusFailed    = "failed"
)

var (
	ErrOrderNotFound      = fmt.Errorf("order not found")
	ErrInvalidTotal       = fmt.Errorf("invalid order total")
	ErrProductNotFound    = fmt.Errorf("product not found")
	ErrProductUnavailable = fmt.Errorf("product not available")
)

// Product represents a menu item
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Emoji       string    `json:"emoji"`
	Category    string    `json:"category"`
	Available   bool      `json:"available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrderItemRequest is what the client sends (no prices!)
type OrderItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

// OrderItem is stored in the database (with server-validated prices)
type OrderItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
}

// CreateOrderRequest is what the client sends
type CreateOrderRequest struct {
	CustomerID      string             `json:"customer_id" validate:"required"`
	CustomerEmail   string             `json:"customer_email" validate:"required,email"`
	Items           []OrderItemRequest `json:"items" validate:"required,min=1,dive"`
	Currency        string             `json:"currency" validate:"required,currency"`
	PaymentMethod   string             `json:"payment_method" validate:"required,payment_method"`
	ShippingAddress string             `json:"shipping_address" validate:"required"`
}

func (r *CreateOrderRequest) FromJSON(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	return decoder.Decode(r)
}

func (r *CreateOrderRequest) Validate() error {
	validate := validator.New()
	validate.RegisterValidation("currency", validateCurrency)
	validate.RegisterValidation("payment_method", validatePaymentMethod)
	return validate.Struct(r)
}

// Order is the full order (stored in DB)
type Order struct {
	ID              string      `json:"id"`
	CustomerID      string      `json:"customer_id"`
	CustomerEmail   string      `json:"customer_email"`
	Items           []OrderItem `json:"items"`
	TotalAmount     float64     `json:"total_amount"`
	Currency        string      `json:"currency"`
	Status          string      `json:"status"`
	PaymentID       string      `json:"payment_id,omitempty"`
	PaymentMethod   string      `json:"payment_method"`
	ShippingAddress string      `json:"shipping_address"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

func (o *Order) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(o)
}

func (o *Order) CalculateTotal() {
	var total float64 = 0
	for i := range o.Items {
		o.Items[i].Subtotal = float64(o.Items[i].Quantity) * o.Items[i].UnitPrice
		total += o.Items[i].Subtotal
	}
	o.TotalAmount = total
}

func validateCurrency(fl validator.FieldLevel) bool {
	validCurrencies := map[string]bool{
		"USD": true,
		"EUR": true,
		"GBP": true,
		"AED": true,
		"EGP": true,
	}
	_, ok := validCurrencies[fl.Field().String()]
	return ok
}

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

type Orders []*Order

func (o *Orders) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(o)
}

type Products []*Product

func (p *Products) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p)
}

func GenerateOrderID() string {
	return "ord_" + uuid.New().String()
}

func GenerateIdempotencyKey(orderID string) string {
	return "idem_" + orderID + "_" + uuid.New().String()
}

func Now() time.Time {
	return time.Now().UTC()
}
