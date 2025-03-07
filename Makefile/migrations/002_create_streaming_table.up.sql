-- 確保你已經安裝 migrate 指令工具，否則 exec.Command("migrate", ...) 會找不到指令。
-- go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

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

-- 插入測試數據
INSERT INTO videos (title, description, file_name, type, status, view_count) VALUES
('Sample Video 1', 'This is a test video.', 'sample1.mp4', 'short', 'ready', 100),
('Sample Video 2', 'Another sample video.', 'sample2.mp4', 'long', 'ready', 250),
('Sample Video 3', 'This video is processing.', 'sample3.mp4', 'short', 'ready', 50),
('Sample Video 4', 'This is an educational video.', 'sample4.mp4', 'long', 'ready', 300),
('Sample Video 5', 'A video about nature.', 'sample5.mp4', 'short', 'ready', 75),
('Sample Video 6', 'A comedy skit.', 'sample6.mp4', 'long', 'ready', 400),
('Sample Video 7', 'A documentary clip.', 'sample7.mp4', 'short', 'ready', 30),
('Sample Video 8', 'A gaming montage.', 'sample8.mp4', 'long', 'ready', 600),
('Sample Video 9', 'A tech tutorial.', 'sample9.mp4', 'short', 'ready', 150),
('Sample Video 10', 'An interview with a celebrity.', 'sample10.mp4', 'long', 'ready', 700),
('Sample Video 11', 'An interview with a celebrity.', 'sample10.mp4', 'long', 'upload', 700),
('Sample Video 12', 'An interview with a celebrity.', 'sample10.mp4', 'long', 'processing', 700);