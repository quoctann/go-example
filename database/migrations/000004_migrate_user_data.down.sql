-- Rollback: Bỏ constraint
ALTER TABLE users 
ALTER COLUMN status DROP NOT NULL;

-- Note: Không thể rollback data changes (email lowercase)
-- Best practice: Backup trước khi chạy data migration