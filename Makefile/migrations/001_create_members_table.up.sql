-- 確保你已經安裝 migrate 指令工具，否則 exec.Command("migrate", ...) 會找不到指令。
-- go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

-- 建立 member 表
CREATE TABLE IF NOT EXISTS member (
    id SERIAL PRIMARY KEY,
    member_id UUID NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    status SMALLINT,
    login_time TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入初始測試用戶
INSERT INTO member (member_id, email, password, status)
VALUES                                                            
    ('550e8400-e29b-41d4-a716-446655440000', 'test@example.com', '$2a$10$3lIU5pVHmInUlcP4sD2pAO3MXkFiOeJpuASVMNmdFIFwiMxMwkyiq', 0);