package repository

import (
	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
)

type PaymentRepository interface {
	Create(payment *data.Payment) error

	GetByID(id string) (*data.Payment, error)

	GetAll() ([]*data.Payment, error)

	GetByOrderID(orderID string) ([]*data.Payment, error)

	GetByCustomerID(customerID string) ([]*data.Payment, error)

	GetByIdempotencyKey(key string) (*data.Payment, error)

	UpdateStatus(payment *data.Payment) error

	GetStatistics() (*PaymentStats, error)

	Close() error
}

type PaymentStats struct {
	TotalPayments      int64   `json:"total_payments"`
	SuccessfulPayments int64   `json:"successful_payments"`
	FailedPayments     int64   `json:"failed_payments"`
	PendingPayments    int64   `json:"pending_payments"`
	TotalAmount        float64 `json:"total_amount_processed"`
	AverageAmount      float64 `json:"average_payment_amount"`
	UniqueCustomers    int64   `json:"unique_customers"`
	Currency           string  `json:"currency"`
}
