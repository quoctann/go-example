-- Index cho email (thường xuyên query)
CREATE INDEX idx_users_email ON users(email);

-- Index cho status (filter theo status)
CREATE INDEX idx_users_status ON users(status);

-- Composite index cho queries phức tạp
CREATE INDEX idx_users_status_created ON users(status, created_at DESC);

-- Partial index (chỉ index users active)
CREATE INDEX idx_users_active ON users(email) WHERE status = 'active';