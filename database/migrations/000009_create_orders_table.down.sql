DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP INDEX IF EXISTS idx_orders_user_created;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_order_number;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP TABLE IF EXISTS orders CASCADE;