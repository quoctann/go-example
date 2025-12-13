-- Ví dụ: Normalize dữ liệu email - chuyển về lowercase
-- Migration này SAFE vì không phụ thuộc cột không tồn tại

UPDATE users 
SET email = LOWER(email)
WHERE email != LOWER(email);

-- Hoặc ví dụ khác: Set default status cho records cũ
UPDATE users 
SET status = 'active'
WHERE status IS NULL;

-- Thêm constraint sau khi clean data
ALTER TABLE users 
ALTER COLUMN status SET NOT NULL;