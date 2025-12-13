-- Tạo bảng settings (key-value config)
CREATE TABLE IF NOT EXISTS settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    type VARCHAR(20) DEFAULT 'string' CHECK (type IN ('string', 'integer', 'float', 'boolean', 'json')),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger update updated_at
CREATE TRIGGER update_settings_updated_at 
    BEFORE UPDATE ON settings 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Index cho key (lookup nhanh)
CREATE INDEX idx_settings_key ON settings(key);

COMMENT ON TABLE settings IS 'Bảng cấu hình hệ thống dạng key-value';
COMMENT ON COLUMN settings.type IS 'Kiểu dữ liệu: string, integer, float, boolean, json';