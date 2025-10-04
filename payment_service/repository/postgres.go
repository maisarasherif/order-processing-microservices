package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/maisarasherif/order-processing-microservices/payment_service/data"
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
		if isDuplicateKeyError(err) {
			return data.ErrDuplicateIdempotencyKey
		}
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(id string) (*data.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, method,
		       customer_id, idempotency_key, transaction_id, error_message,
		       created_at, processed_at, failed_at
		FROM payments
		WHERE id = $1
	`

	payment := &data.Payment{}
	err := r.db.QueryRow(query, id).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.Method,
		&payment.CustomerID,
		&payment.IdempotencyKey,
		&payment.TransactionID,
		&payment.ErrorMessage,
		&payment.CreatedAt,
		&payment.ProcessedAt,
		&payment.FailedAt,
	)

	if err == sql.ErrNoRows {
		return nil, data.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

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

func (r *PostgresRepository) GetByIdempotencyKey(key string) (*data.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, method,
		       customer_id, idempotency_key, transaction_id, error_message,
		       created_at, processed_at, failed_at
		FROM payments
		WHERE idempotency_key = $1
	`

	payment := &data.Payment{}
	err := r.db.QueryRow(query, key).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.Method,
		&payment.CustomerID,
		&payment.IdempotencyKey,
		&payment.TransactionID,
		&payment.ErrorMessage,
		&payment.CreatedAt,
		&payment.ProcessedAt,
		&payment.FailedAt,
	)

	if err == sql.ErrNoRows {
		return nil, data.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment by idempotency key: %w", err)
	}

	return payment, nil
}

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

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return data.ErrPaymentNotFound
	}

	return nil
}

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

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) scanPayments(rows *sql.Rows) ([]*data.Payment, error) {
	var payments []*data.Payment

	for rows.Next() {
		payment := &data.Payment{}
		err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.Method,
			&payment.CustomerID,
			&payment.IdempotencyKey,
			&payment.TransactionID,
			&payment.ErrorMessage,
			&payment.CreatedAt,
			&payment.ProcessedAt,
			&payment.FailedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payments: %w", err)
	}

	return payments, nil
}

func isDuplicateKeyError(err error) bool {

	return err != nil && (err.Error() == "pq: duplicate key value violates unique constraint \"payments_idempotency_key_key\"" ||
		err.Error() == "pq: duplicate key value violates unique constraint \"payments_pkey\"")
}
