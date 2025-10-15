-- Notification Service Database Schema
-- Creates tables for storing notification history and status

-- Enable UUID extension for generating unique IDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Main notifications table
-- Stores all notification records (sent and failed)
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(50) NOT NULL DEFAULT 'receipt',
    customer_email VARCHAR(255) NOT NULL,
    customer_id VARCHAR(50),
    order_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
    sent_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
-- Speed up queries by customer, order, and status
CREATE INDEX IF NOT EXISTS idx_notifications_customer_email ON notifications(customer_email);
CREATE INDEX IF NOT EXISTS idx_notifications_customer_id ON notifications(customer_id);
CREATE INDEX IF NOT EXISTS idx_notifications_order_id ON notifications(order_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

-- Composite index for common query pattern (order + status)
CREATE INDEX IF NOT EXISTS idx_notifications_order_status ON notifications(order_id, status);

-- Grant permissions to notification service user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO notifuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO notifuser;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Notification Service Database initialized successfully!';
    RAISE NOTICE 'Tables created: notifications';
END $$;