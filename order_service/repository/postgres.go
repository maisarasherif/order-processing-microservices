package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/maisarasherif/order-processing-microservices/order_service/data"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresRepository{db: db}, nil
}

// ============ PRODUCT OPERATIONS ============

func (r *PostgresRepository) GetProduct(id string) (*data.Product, error) {
	query := `
		SELECT id, name, description, price, emoji, category, available, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	product := &data.Product{}
	var description sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&product.ID,
		&product.Name,
		&description,
		&product.Price,
		&product.Emoji,
		&product.Category,
		&product.Available,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, data.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	if description.Valid {
		product.Description = description.String
	}

	return product, nil
}

func (r *PostgresRepository) GetAllProducts() ([]*data.Product, error) {
	query := `
		SELECT id, name, description, price, emoji, category, available, created_at, updated_at
		FROM products
		ORDER BY category, name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	return r.scanProducts(rows)
}

func (r *PostgresRepository) GetAvailableProducts() ([]*data.Product, error) {
	query := `
		SELECT id, name, description, price, emoji, category, available, created_at, updated_at
		FROM products
		WHERE available = true
		ORDER BY category, name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query available products: %w", err)
	}
	defer rows.Close()

	return r.scanProducts(rows)
}

func (r *PostgresRepository) scanProducts(rows *sql.Rows) ([]*data.Product, error) {
	var products []*data.Product

	for rows.Next() {
		product := &data.Product{}
		var description sql.NullString

		err := rows.Scan(
			&product.ID,
			&product.Name,
			&description,
			&product.Price,
			&product.Emoji,
			&product.Category,
			&product.Available,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		if description.Valid {
			product.Description = description.String
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	return products, nil
}

// ============ ORDER OPERATIONS ============

func (r *PostgresRepository) Create(order *data.Order) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	query := `
		INSERT INTO orders (
			id, customer_id, customer_email, items, total_amount,
			currency, status, payment_method, shipping_address,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = tx.Exec(
		query,
		order.ID,
		order.CustomerID,
		order.CustomerEmail,
		itemsJSON,
		order.TotalAmount,
		order.Currency,
		order.Status,
		order.PaymentMethod,
		order.ShippingAddress,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(id string) (*data.Order, error) {
	query := `
		SELECT id, customer_id, customer_email, items, total_amount,
		       currency, status, payment_id, payment_method,
		       shipping_address, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	order := &data.Order{}
	var itemsJSON []byte
	var paymentID sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&order.ID,
		&order.CustomerID,
		&order.CustomerEmail,
		&itemsJSON,
		&order.TotalAmount,
		&order.Currency,
		&order.Status,
		&paymentID,
		&order.PaymentMethod,
		&order.ShippingAddress,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, data.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %w", err)
	}

	if paymentID.Valid {
		order.PaymentID = paymentID.String
	}

	return order, nil
}

func (r *PostgresRepository) GetAll() ([]*data.Order, error) {
	query := `
		SELECT id, customer_id, customer_email, items, total_amount,
		       currency, status, payment_id, payment_method,
		       shipping_address, created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *PostgresRepository) GetByCustomerID(customerID string) ([]*data.Order, error) {
	query := `
		SELECT id, customer_id, customer_email, items, total_amount,
		       currency, status, payment_id, payment_method,
		       shipping_address, created_at, updated_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *PostgresRepository) scanOrders(rows *sql.Rows) ([]*data.Order, error) {
	var orders []*data.Order

	for rows.Next() {
		order := &data.Order{}
		var itemsJSON []byte
		var paymentID sql.NullString

		err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&order.CustomerEmail,
			&itemsJSON,
			&order.TotalAmount,
			&order.Currency,
			&order.Status,
			&paymentID,
			&order.PaymentMethod,
			&order.ShippingAddress,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("failed to unmarshal items: %w", err)
		}

		if paymentID.Valid {
			order.PaymentID = paymentID.String
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}

func (r *PostgresRepository) UpdateStatus(id, status string) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.Exec(query, status, data.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return data.ErrOrderNotFound
	}

	return nil
}

func (r *PostgresRepository) UpdatePaymentID(orderID, paymentID string) error {
	query := `
		UPDATE orders
		SET payment_id = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.Exec(query, paymentID, data.Now(), orderID)
	if err != nil {
		return fmt.Errorf("failed to update payment ID: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return data.ErrOrderNotFound
	}

	return nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}
