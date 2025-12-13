-- Tạo bảng countries (master data)
CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    code VARCHAR(2) NOT NULL UNIQUE, -- ISO 3166-1 alpha-2 (VN, US, JP)
    name VARCHAR(100) NOT NULL,
    phone_code VARCHAR(10), -- +84, +1, +81
    currency VARCHAR(3), -- VND, USD, JPY
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index cho code (lookup by country code)
CREATE INDEX idx_countries_code ON countries(code);

-- Index cho active countries
CREATE INDEX idx_countries_active ON countries(is_active) WHERE is_active = true;

COMMENT ON TABLE countries IS 'Bảng master data quốc gia';
COMMENT ON COLUMN countries.code IS 'ISO 3166-1 alpha-2 country code';