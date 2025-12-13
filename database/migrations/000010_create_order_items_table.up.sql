-- Tạo bảng order_items (chi tiết đơn hàng)
CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    
    -- Snapshot data tại thời điểm order (để tránh thay đổi giá)
    product_name VARCHAR(255) NOT NULL,
    product_sku VARCHAR(100),
    price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    subtotal DECIMAL(10, 2) NOT NULL CHECK (subtotal >= 0), -- price * quantity
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index cho order_id (query items by order)
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- Index cho product_id (analytics)
CREATE INDEX idx_order_items_product_id ON order_items(product_id);

COMMENT ON TABLE order_items IS 'Chi tiết sản phẩm trong đơn hàng';
COMMENT ON COLUMN order_items.product_name IS 'Snapshot tên sản phẩm tại thời điểm đặt hàng';
COMMENT ON COLUMN order_items.price IS 'Snapshot giá tại thời điểm đặt hàng';