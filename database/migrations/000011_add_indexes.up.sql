
-- Full-text search index cho products
CREATE INDEX idx_products_name_search ON products USING GIN (to_tsvector('english', name));
CREATE INDEX idx_products_description_search ON products USING GIN (to_tsvector('english', description));

-- Partial index cho in-stock products
CREATE INDEX idx_products_in_stock ON products(id) WHERE stock > 0 AND is_active = true;

-- Composite index cho common queries
CREATE INDEX idx_products_category_price ON products(category_id, price);
CREATE INDEX idx_products_category_created ON products(category_id, created_at DESC);

-- Index cho email lookup (case-insensitive)
CREATE INDEX idx_users_email_lower ON users(LOWER(email));

-- Composite index cho user orders with date range
CREATE INDEX idx_orders_user_status_date ON orders(user_id, status, created_at DESC);

COMMENT ON INDEX idx_products_name_search IS 'Full-text search index cho tên sản phẩm';
COMMENT ON INDEX idx_products_in_stock IS 'Partial index chỉ cho products còn hàng';