package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
)

// PostgresRepository implements PaymentRepository for PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	// Open database connection
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)                 // Maximum open connections
	db.SetMaxIdleConns(5)                  // Maximum idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Connection lifetime

	return &PostgresRepository{db: db}, nil
}

// Create inserts a new payment into the database
func (r *PostgresRepository) Create(payment *data.Payment) error {
	query := `
		INSERT INTO payments (
			id, order_id, amount, currency, status, method,
			customer_id, idempotency_key, transaction_id, error_message,
			created_at, processed_at, failed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Exec(
		query,
		payment.ID,
		payment.OrderID,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.Method,
		payment.CustomerID,
		payment.IdempotencyKey,
		payment.TransactionID,
		payment.ErrorMessage,
		payment.CreatedAt,
		payment.ProcessedAt,
		payment.FailedAt,
	)

	if err != nil {
		// Check for duplicate idempotency key
		if isDuplicateKeyError(err) {
			return data.ErrDuplicateIdempotencyKey
		}
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// GetByID retrieves a payment by its ID
func (r *PostgresRepository) GetByID(id string) (*data.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, method,
		       customer_id, idempotency_key, transaction_id, error_message,
		       created_at, processed_at, failed_at
		FROM payments
		WHERE id = $1
	`

	payment := &data.Payment{}

	// Use sql.NullString for nullable fields
	var transactionID, errorMessage sql.NullString
	var processedAt, failedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.Method,
		&payment.CustomerID,
		&payment.IdempotencyKey,
		&transactionID,
		&errorMessage,
		&payment.CreatedAt,
		&processedAt,
		&failedAt,
	)

	if err == sql.ErrNoRows {
		return nil, data.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Convert sql.Null types to Go types
	if transactionID.Valid {
		payment.TransactionID = transactionID.String
	}
	if errorMessage.Valid {
		payment.ErrorMessage = errorMessage.String
	}
	if processedAt.Valid {
		payment.ProcessedAt = &processedAt.Time
	}
	if failedAt.Valid {
		payment.FailedAt = &failedAt.Time
	}

	return payment, nil
}

// GetAll retrieves all payments
func (r *PostgresRepository) GetAll() ([]*data.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, method,
		       customer_id, idempotency_key, transaction_id, error_message,
		       created_at, processed_at, failed_at
		FROM payments
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments: %w", err)
	}
	defer rows.Close()

	return r.scanPayments(rows)
}

// GetByOrderID retrieves all payments for a specific order
func (r *PostgresRepository) GetByOrderID(orderID string) ([]*data.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, method,
		       customer_id, idempotency_key, transaction_id, error_message,
		       created_at, processed_at, failed_at
		FROM payments
		WHERE order_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments by order: %w", err)
	}
	defer rows.Close()

	return r.scanPayments(rows)
}

// GetByCustomerID retrieves all payments for a specific customer
func (r *PostgresRepository) GetByCustomerID(customerID string) ([]*data.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, method,
		       customer_id, idempotency_key, transaction_id, error_message,
		       created_at, processed_at, failed_at
		FROM payments
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments by customer: %w", err)
	}
	defer rows.Close()

	return r.scanPayments(rows)
}

// GetByIdempotencyKey retrieves a payment by idempotency key
func (r *PostgresRepository) GetByIdempotencyKey(key string) (*data.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, method,
		       customer_id, idempotency_key, transaction_id, error_message,
		       created_at, processed_at, failed_at
		FROM payments
		WHERE idempotency_key = $1
	`

	payment := &data.Payment{}

	// Use sql.NullString for nullable fields
	var transactionID, errorMessage sql.NullString
	var processedAt, failedAt sql.NullTime

	err := r.db.QueryRow(query, key).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.Method,
		&payment.CustomerID,
		&payment.IdempotencyKey,
		&transactionID,
		&errorMessage,
		&payment.CreatedAt,
		&processedAt,
		&failedAt,
	)

	if err == sql.ErrNoRows {
		return nil, data.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment by idempotency key: %w", err)
	}

	// Convert sql.Null types to Go types
	if transactionID.Valid {
		payment.TransactionID = transactionID.String
	}
	if errorMessage.Valid {
		payment.ErrorMessage = errorMessage.String
	}
	if processedAt.Valid {
		payment.ProcessedAt = &processedAt.Time
	}
	if failedAt.Valid {
		payment.FailedAt = &failedAt.Time
	}

	return payment, nil
}

// UpdateStatus updates payment status and related fields
func (r *PostgresRepository) UpdateStatus(payment *data.Payment) error {
	query := `
		UPDATE payments
		SET status = $1,
		    transaction_id = $2,
		    error_message = $3,
		    processed_at = $4,
		    failed_at = $5
		WHERE id = $6
	`

	result, err := r.db.Exec(
		query,
		payment.Status,
		payment.TransactionID,
		payment.ErrorMessage,
		payment.ProcessedAt,
		payment.FailedAt,
		payment.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Check if any row was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return data.ErrPaymentNotFound
	}

	return nil
}

// GetStatistics retrieves payment statistics
func (r *PostgresRepository) GetStatistics() (*PaymentStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_payments,
			COUNT(*) FILTER (WHERE status = 'completed') as successful_payments,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_payments,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_payments,
			COALESCE(SUM(amount) FILTER (WHERE status = 'completed'), 0) as total_amount,
			COALESCE(AVG(amount) FILTER (WHERE status = 'completed'), 0) as avg_amount,
			COUNT(DISTINCT customer_id) as unique_customers
		FROM payments
	`

	stats := &PaymentStats{}
	err := r.db.QueryRow(query).Scan(
		&stats.TotalPayments,
		&stats.SuccessfulPayments,
		&stats.FailedPayments,
		&stats.PendingPayments,
		&stats.TotalAmount,
		&stats.AverageAmount,
		&stats.UniqueCustomers,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// scanPayments is a helper function to scan multiple payment rows
func (r *PostgresRepository) scanPayments(rows *sql.Rows) ([]*data.Payment, error) {
	var payments []*data.Payment

	for rows.Next() {
		payment := &data.Payment{}

		// Use sql.NullString for nullable fields
		var transactionID, errorMessage sql.NullString
		var processedAt, failedAt sql.NullTime

		err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.Method,
			&payment.CustomerID,
			&payment.IdempotencyKey,
			&transactionID,
			&errorMessage,
			&payment.CreatedAt,
			&processedAt,
			&failedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}

		// Convert sql.Null types to Go types
		if transactionID.Valid {
			payment.TransactionID = transactionID.String
		}
		if errorMessage.Valid {
			payment.ErrorMessage = errorMessage.String
		}
		if processedAt.Valid {
			payment.ProcessedAt = &processedAt.Time
		}
		if failedAt.Valid {
			payment.FailedAt = &failedAt.Time
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payments: %w", err)
	}

	return payments, nil
}

// isDuplicateKeyError checks if error is a duplicate key violation
func isDuplicateKeyError(err error) bool {
	// PostgreSQL error code 23505 is unique_violation
	return err != nil && (err.Error() == "pq: duplicate key value violates unique constraint \"payments_idempotency_key_key\"" ||
		err.Error() == "pq: duplicate key value violates unique constraint \"payments_pkey\"")
}
