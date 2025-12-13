-- Xóa constraint trước
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_phone_format;

-- Xóa index
DROP INDEX IF EXISTS idx_users_phone;

-- Xóa cột
ALTER TABLE users DROP COLUMN IF EXISTS phone;