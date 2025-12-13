DROP TRIGGER IF EXISTS update_products_updated_at ON products;
DROP INDEX IF EXISTS idx_products_price;
DROP INDEX IF EXISTS idx_products_attributes;
DROP INDEX IF EXISTS idx_products_active;
DROP INDEX IF EXISTS idx_products_sku;
DROP INDEX IF EXISTS idx_products_category_id;
DROP TABLE IF EXISTS products CASCADE;