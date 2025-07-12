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
    member_id UUID NOT NULL UNIQUE,
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

-- 为 member 表创建索引
CREATE INDEX IF NOT EXISTS idx_member_status ON member(status);
CREATE INDEX IF NOT EXISTS idx_member_login_time ON member(login_time);
CREATE INDEX IF NOT EXISTS idx_member_created_at ON member(created_at);
CREATE INDEX IF NOT EXISTS idx_member_updated_at ON member(updated_at);

-- 为状态和登录时间组合查询创建复合索引
CREATE INDEX IF NOT EXISTS idx_member_status_login_time ON member(status, login_time);

SELECT 'CREATE DATABASE streaming_db'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'streaming_db'
) \gexec

-- 切换到目标数据库
\c streaming_db;

-- 创建表
CREATE TABLE IF NOT EXISTS videos (
    id          SERIAL PRIMARY KEY,
    title       VARCHAR(255),
    description TEXT,
    file_name   TEXT,            -- 對應 FileName, 存 MinIO 物件名稱
    type        VARCHAR(50),     -- 影片型態: "short" or "long"
    status      VARCHAR(50),     -- "uploaded", "processing", "ready"
    view_count  INT DEFAULT 0    -- 預設0次觀看
);

-- 为 videos 表创建索引
CREATE INDEX IF NOT EXISTS idx_videos_type ON videos(type);
CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);
CREATE INDEX IF NOT EXISTS idx_videos_view_count ON videos(view_count DESC);
CREATE INDEX IF NOT EXISTS idx_videos_title ON videos(title);
CREATE INDEX IF NOT EXISTS idx_videos_file_name ON videos(file_name);

-- 为常见查询创建复合索引
CREATE INDEX IF NOT EXISTS idx_videos_type_status ON videos(type, status);
CREATE INDEX IF NOT EXISTS idx_videos_status_view_count ON videos(status, view_count DESC);

-- 为标题搜索创建文本搜索索引（支持模糊查询）
CREATE INDEX IF NOT EXISTS idx_videos_title_gin ON videos USING gin(to_tsvector('english', title));