CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Products catalog table
CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price NUMERIC(12, 2) NOT NULL CHECK (price >= 0),
    emoji VARCHAR(10),
    category VARCHAR(100),
    available BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(50) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    items JSONB NOT NULL,  
    total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount > 0),
    currency CHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'paid', 'confirmed', 'cancelled', 'failed')),
    payment_id VARCHAR(50),  
    payment_method VARCHAR(50) NOT NULL,
    shipping_address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for orders
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_payment_id ON orders(payment_id);
CREATE INDEX IF NOT EXISTS idx_orders_customer_status ON orders(customer_id, status);

-- Indexes for products
CREATE INDEX IF NOT EXISTS idx_products_available ON products(available);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- Trigger for orders updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_orders_updated_at 
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_products_updated_at 
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed initial products
INSERT INTO products (id, name, description, price, emoji, category, available) VALUES 
    ('burger', 'Classic Burger', 'Juicy beef patty with lettuce, tomato, and special sauce', 12.99, '🍔', 'mains', true),
    ('pizza', 'Pepperoni Pizza', 'Large pizza with pepperoni and mozzarella cheese', 15.99, '🍕', 'mains', true),
    ('fries', 'French Fries', 'Crispy golden fries with sea salt', 4.99, '🍟', 'sides', true),
    ('hotdog', 'Hot Dog', 'All-beef hot dog with your choice of toppings', 8.99, '🌭', 'mains', true),
    ('taco', 'Taco Supreme', 'Three tacos with seasoned beef, cheese, and salsa', 9.99, '🌮', 'mains', true),
    ('sushi', 'Sushi Roll', 'Fresh salmon and avocado roll (8 pieces)', 18.99, '🍣', 'mains', true),
    ('ramen', 'Ramen Bowl', 'Rich pork broth with noodles, egg, and vegetables', 13.99, '🍜', 'mains', true),
    ('icecream', 'Ice Cream', 'Premium vanilla ice cream with chocolate sauce', 5.99, '🍦', 'desserts', true)
ON CONFLICT (id) DO NOTHING;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO orderuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO orderuser;

DO $$
BEGIN
    RAISE NOTICE 'Order Service Database initialized successfully!';
    RAISE NOTICE 'Tables created: orders, products';
    RAISE NOTICE 'Seeded 8 products';
END $$;