-- Tạo bảng orders
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    order_number VARCHAR(50) NOT NULL UNIQUE, -- ORD-20231215-001
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'shipped', 'delivered', 'cancelled')),
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    tax_amount DECIMAL(10, 2) DEFAULT 0 CHECK (tax_amount >= 0),
    shipping_amount DECIMAL(10, 2) DEFAULT 0 CHECK (shipping_amount >= 0),
    discount_amount DECIMAL(10, 2) DEFAULT 0 CHECK (discount_amount >= 0),
    
    -- Shipping info
    shipping_name VARCHAR(255),
    shipping_email VARCHAR(255),
    shipping_phone VARCHAR(20),
    shipping_address TEXT,
    shipping_city VARCHAR(100),
    shipping_country_id INTEGER REFERENCES countries(id),
    
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger update updated_at
CREATE TRIGGER update_orders_updated_at 
    BEFORE UPDATE ON orders 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Index cho user_id (query orders by user)
CREATE INDEX idx_orders_user_id ON orders(user_id);

-- Index cho order_number (lookup by order number)
CREATE INDEX idx_orders_order_number ON orders(order_number);

-- Index cho status
CREATE INDEX idx_orders_status ON orders(status);

-- Composite index cho user's recent orders
CREATE INDEX idx_orders_user_created ON orders(user_id, created_at DESC);

COMMENT ON TABLE orders IS 'Bảng đơn hàng';
COMMENT ON COLUMN orders.order_number IS 'Mã đơn hàng unique cho khách hàng';