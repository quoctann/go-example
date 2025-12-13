-- Xóa trigger trước
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Xóa bảng
DROP TABLE IF EXISTS users;