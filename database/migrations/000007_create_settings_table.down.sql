DROP TRIGGER IF EXISTS update_settings_updated_at ON settings;
DROP INDEX IF EXISTS idx_settings_key;
DROP TABLE IF EXISTS settings CASCADE;