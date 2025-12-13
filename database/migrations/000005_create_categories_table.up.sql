-- Tạo bảng categories (hỗ trợ nested categories)
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    parent_id INTEGER REFERENCES categories(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger update updated_at
CREATE TRIGGER update_categories_updated_at 
    BEFORE UPDATE ON categories 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Index cho parent_id (query subcategories)
CREATE INDEX idx_categories_parent_id ON categories(parent_id);

-- Index cho slug (SEO friendly URLs)
CREATE INDEX idx_categories_slug ON categories(slug);

-- Index cho active categories
CREATE INDEX idx_categories_active ON categories(is_active) WHERE is_active = true;

COMMENT ON TABLE categories IS 'Bảng danh mục sản phẩm (hỗ trợ nested)';
COMMENT ON COLUMN categories.parent_id IS 'ID của category cha (NULL = root category)';