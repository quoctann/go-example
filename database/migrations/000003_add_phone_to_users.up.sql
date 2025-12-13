-- Thêm cột phone
ALTER TABLE users 
ADD COLUMN phone VARCHAR(20);

-- Thêm index cho phone
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;

-- Thêm constraint kiểm tra format (optional)
ALTER TABLE users 
ADD CONSTRAINT check_phone_format 
CHECK (phone ~ '^[0-9+\-\(\) ]+$' OR phone IS NULL);