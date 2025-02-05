-- 检查并创建数据库
SELECT 'CREATE DATABASE member_db'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'member_db'
) \gexec

-- 切换到目标数据库
\c member_db;

-- 创建表
CREATE TABLE IF NOT EXISTS member (
    id SERIAL PRIMARY KEY,
    member_id VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    status SMALLINT,
    login_time TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 为 status 字段添加注释
COMMENT ON COLUMN member.status IS '状态: 0=offline, 1=online, 2=ban ,3=delete';

-- 创建用于自动更新时间戳的触发函数
CREATE OR REPLACE FUNCTION update_member_timestamps()
RETURNS TRIGGER AS $$
BEGIN
    -- 当插入记录时设置 created_at 和 updated_at
    IF TG_OP = 'INSERT' THEN
        NEW.created_at = CURRENT_TIMESTAMP;
        NEW.updated_at = CURRENT_TIMESTAMP;
    END IF;

    -- 当更新记录时更新 updated_at
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = CURRENT_TIMESTAMP;
    END IF;

    -- 如果 status = 1，更新 login_time
    IF NEW.status = 1 THEN
        NEW.login_time = CURRENT_TIMESTAMP;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为 member 表创建触发器
CREATE TRIGGER update_member_timestamps
BEFORE INSERT OR UPDATE ON member
FOR EACH ROW
EXECUTE FUNCTION update_member_timestamps();