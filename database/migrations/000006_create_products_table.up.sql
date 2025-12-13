-- Tạo bảng products
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sku VARCHAR(100) UNIQUE,
    slug VARCHAR(255) NOT NULL UNIQUE DEFAULT '',
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    stock INTEGER DEFAULT 0 CHECK (stock >= 0),
    is_active BOOLEAN DEFAULT true,
    attributes JSONB, -- Lưu attributes động (brand, color, size, etc.)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger update updated_at
CREATE TRIGGER update_products_updated_at 
    BEFORE UPDATE ON products 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Index cho category_id (query products by category)
CREATE INDEX idx_products_category_id ON products(category_id);

-- Index cho SKU (inventory lookup)
CREATE INDEX idx_products_sku ON products(sku);

-- Index cho slug (SEO)
CREATE INDEX idx_products_slug ON products(slug);

-- Index cho active products
CREATE INDEX idx_products_active ON products(is_active) WHERE is_active = true;

-- Index cho JSONB attributes (search by brand, color, etc.)
CREATE INDEX idx_products_attributes ON products USING GIN (attributes);

-- Index cho price range queries
CREATE INDEX idx_products_price ON products(price);

COMMENT ON TABLE products IS 'Bảng sản phẩm';
COMMENT ON COLUMN products.sku IS 'Stock Keeping Unit - mã unique cho inventory';
COMMENT ON COLUMN products.attributes IS 'Attributes động dạng JSON: {"brand": "Apple", "color": "Red"}';