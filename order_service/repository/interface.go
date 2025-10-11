package repository

import (
	"github.com/maisarasherif/order-processing-microservices/order_service/data"
)

type OrderRepository interface {
	// Order operations
	Create(order *data.Order) error
	GetByID(id string) (*data.Order, error)
	GetAll() ([]*data.Order, error)
	GetByCustomerID(customerID string) ([]*data.Order, error)
	UpdateStatus(id, status string) error
	UpdatePaymentID(orderID, paymentID string) error

	// Product operations
	GetProduct(id string) (*data.Product, error)
	GetAllProducts() ([]*data.Product, error)
	GetAvailableProducts() ([]*data.Product, error)

	Close() error
}
