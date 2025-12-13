# Practices:

1. Luôn có up và down

2. Migration phải Idempotent - lũy đẳng, chạy nhiều lần ko lỗi - sai khác giá trị

    VD: sử dụng `CREATE TABLE IF NOT EXISTS ...` thay vì chỉ `CREATE TABLE`

3. Ko sửa migration đã deploy, muốn thay đổi phải tạo migration mới

4. Test trên local/dev trước khi deploy

5. Backup trước khi migrate production

    `pg_dump mydb > backup_before_migration_$(date +%Y%m%d).sql`

6. Migration có thể rollback nhanh

    VD: thêm cột với DEFAULT thay vì UPDATE toàn bộ bảng
    Good: `ALTER TABLE users ADD COLUMN age INT DEFAULT 0`

    Bad: `UPDATE users SET age = 0 WHERE age IS NULL`

7. Connection pool: tránh mở quá nhiều connections (exhaust db), reuse connections (giảm overhead), close connection cũ (tránh stale connections)

8. Luôn dùng context với timeout, tránh query chạy mãi không dừng, block resources

9. Sử dụng prepare statement/QueryContext (placeholder tùy db) để ngăn SQL injection

11. Với standard lib, nhớ xử lý null value
    
    Sẽ panic nếu scan value null: `var name string`

    Check value trước khi sử dụng: `var name sql.NullString`

12. Close resource đúng cách

    `rows, err := db.Query("SELECT * FROM users") ...`

    Quang trọng: `defer rows.Close()`

13. Index strategy: cân nhắc đánh index lên những cột thường xuyên trong WHERE, JOIN, ORDER BY, không tạo quá nhiều index sẽ làm chậm INSERT/UPDATE, cần monitor performance trước khi add index

    -- B-tree index (mặc định) - tốt cho equality và range queries:
    `CREATE INDEX idx_users_email ON users(email);`

    -- Partial index - chỉ index subset:
    `CREATE INDEX idx_active_users ON users(email) WHERE status = 'active';`

    -- Composite index - nhiều cột:
    `CREATE INDEX idx_users_status_created ON users(status, created_at DESC);`

    -- Unique index
    `CREATE UNIQUE INDEX idx_users_email_unique ON users(email);`

15. Schema versioning, nếu có sử dụng, nên check trước khi app được start
