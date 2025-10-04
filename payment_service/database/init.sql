-- Payment Service Database Schema
-- This script runs automatically when PostgreSQL container starts for the first time

-- Enable UUID extension (useful for generating unique IDs)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Main payments table
CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(50) PRIMARY KEY,
    order_id VARCHAR(50) NOT NULL,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    method VARCHAR(50) NOT NULL,
    customer_id VARCHAR(50) NOT NULL,
    idempotency_key VARCHAR(100) UNIQUE NOT NULL,
    transaction_id VARCHAR(100),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
-- These speed up common queries
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_customer_id ON payments(customer_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_idempotency_key ON payments(idempotency_key);

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_payments_customer_status ON payments(customer_id, status);

-- Payment audit log - tracks all status changes
CREATE TABLE IF NOT EXISTS payment_audit_log (
    id SERIAL PRIMARY KEY,
    payment_id VARCHAR(50) NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    old_status VARCHAR(20),
    new_status VARCHAR(20) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by VARCHAR(100),
    notes TEXT,
    metadata JSONB  -- Store additional context as JSON
);

-- Index for audit queries
CREATE INDEX IF NOT EXISTS idx_audit_payment_id ON payment_audit_log(payment_id);
CREATE INDEX IF NOT EXISTS idx_audit_changed_at ON payment_audit_log(changed_at DESC);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-update updated_at on payments table
CREATE TRIGGER update_payments_updated_at 
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically create audit log entry when payment status changes
CREATE OR REPLACE FUNCTION log_payment_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if status actually changed
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO payment_audit_log (
            payment_id,
            old_status,
            new_status,
            notes
        ) VALUES (
            NEW.id,
            OLD.status,
            NEW.status,
            CASE 
                WHEN NEW.status = 'failed' THEN NEW.error_message
                ELSE NULL
            END
        );
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-log status changes
CREATE TRIGGER log_payment_status_change_trigger
    AFTER UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION log_payment_status_change();

-- Create view for payment analytics (useful for dashboards)
CREATE OR REPLACE VIEW payment_statistics AS
SELECT
    COUNT(*) as total_payments,
    COUNT(*) FILTER (WHERE status = 'completed') as successful_payments,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_payments,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_payments,
    SUM(amount) FILTER (WHERE status = 'completed') as total_amount_processed,
    AVG(amount) FILTER (WHERE status = 'completed') as average_payment_amount,
    COUNT(DISTINCT customer_id) as unique_customers,
    currency
FROM payments
GROUP BY currency;

-- Grant permissions to payment service user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO paymentuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO paymentuser;

-- Insert some test data (optional - remove in production)
INSERT INTO payments (
    id, order_id, amount, currency, status, method, 
    customer_id, idempotency_key, transaction_id
) VALUES 
    ('pay_test_1', 'order_test_1', 99.99, 'USD', 'completed', 'credit_card', 
     'cust_test_1', 'test_key_1', 'txn_test_1'),
    ('pay_test_2', 'order_test_2', 49.50, 'EUR', 'completed', 'paypal', 
     'cust_test_2', 'test_key_2', 'txn_test_2')
ON CONFLICT (idempotency_key) DO NOTHING;

-- Create indexes on JSONB fields if using metadata
CREATE INDEX IF NOT EXISTS idx_audit_metadata ON payment_audit_log USING GIN (metadata);

-- Performance: Analyze tables for query optimization
ANALYZE payments;
ANALYZE payment_audit_log;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Payment Service Database initialized successfully!';
    RAISE NOTICE 'Tables created: payments, payment_audit_log';
    RAISE NOTICE 'Views created: payment_statistics';
    RAISE NOTICE 'Triggers created: status change logging, timestamp updates';
END $$;