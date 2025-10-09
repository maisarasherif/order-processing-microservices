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
	ErrOrderNotFound = fmt.Errorf("order not found")
	ErrInvalidTotal  = fmt.Errorf("invalid order total")
)

type OrderItem struct {
	ProductID   string  `json:"product_id" validate:"required"`
	ProductName string  `json:"product_name" validate:"required"`
	Quantity    int     `json:"quantity" validate:"required,gt=0"`
	UnitPrice   float64 `json:"unit_price" validate:"required,gt=0"`
	Subtotal    float64 `json:"subtotal"`
}

type Order struct {
	ID              string      `json:"id"`
	CustomerID      string      `json:"customer_id" validate:"required"`
	CustomerEmail   string      `json:"customer_email" validate:"required,email"`
	Items           []OrderItem `json:"items" validate:"required,min=1,dive"` // dive validates each item
	TotalAmount     float64     `json:"total_amount"`
	Currency        string      `json:"currency" validate:"required,currency"`
	Status          string      `json:"status"`
	PaymentID       string      `json:"payment_id,omitempty"`
	PaymentMethod   string      `json:"payment_method" validate:"required,payment_method"`
	ShippingAddress string      `json:"shipping_address" validate:"required"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

func (o *Order) FromJSON(r io.Reader) error {

	decoder := json.NewDecoder(r)

	return decoder.Decode(o)
}

func (o *Order) ToJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(o)
}

func (o *Order) Validate() error {

	validate := validator.New()

	validate.RegisterValidation("currency", validateCurrency)
	validate.RegisterValidation("payment_method", validatePaymentMethod)

	return validate.Struct(o)
}

func (o *Order) CalculateTotal() {
	var total float64 = 0

	for _, item := range o.Items {
		item.Subtotal = float64(item.Quantity) * item.UnitPrice
		total += item.Subtotal
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
		"apple_pay":     true,
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

func GenerateOrderID() string {
	return "ord_" + uuid.New().String()
}

func GenerateIdempotencyKey(orderID string) string {
	return "idem_" + orderID + "_" + uuid.New().String()
}

func Now() time.Time {
	return time.Now().UTC()
}
