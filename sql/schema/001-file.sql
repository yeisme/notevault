-- 文件基础信息表
CREATE TABLE IF NOT EXISTS files (
    file_id VARCHAR(128) PRIMARY KEY,
    -- 文件唯一ID (使用字符串存储UUID)
    user_id VARCHAR(64) NOT NULL,
    -- 文件所属用户ID
    file_name VARCHAR(255) NOT NULL,
    -- 文件名
    file_type VARCHAR(50) NOT NULL,
    -- 文件类型 (document, image, video, text等)
    content_type VARCHAR(100) NOT NULL,
    -- MIME类型
    size BIGINT NOT NULL,
    -- 文件大小(字节)
    path VARCHAR(500) NOT NULL,
    -- 存储路径
    created_at BIGINT NOT NULL,
    -- 创建时间(Unix时间戳)
    updated_at BIGINT NOT NULL,
    -- 更新时间(Unix时间戳)
    deleted_at BIGINT DEFAULT NULL,
    -- 软删除时间(Unix时间戳)，NULL表示未删除
    status SMALLINT NOT NULL DEFAULT 0,
    -- 文件状态: 0=正常, 1=存档, 2=管理员回收站, 3=待删除（用户回收站）
    trashed_at BIGINT DEFAULT NULL,
    -- 移入回收站时间(Unix时间戳)
    current_version INT NOT NULL DEFAULT 1,
    -- 当前版本号
    description TEXT -- 文件描述
);
-- PostgreSQL索引 (注意语法区别)
CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_files_file_type ON files(file_type);
CREATE INDEX idx_files_created_at ON files(created_at);
CREATE INDEX idx_files_updated_at ON files(updated_at);
-- 文件版本表
CREATE TABLE IF NOT EXISTS file_versions (
    version_id VARCHAR(192) PRIMARY KEY,
    -- 版本唯一ID
    file_id VARCHAR(64) NOT NULL,
    -- 关联的文件ID
    version_number INT NOT NULL,
    -- 版本号
    size BIGINT NOT NULL,
    -- 该版本文件大小
    path VARCHAR(500) NOT NULL,
    -- 该版本文件存储路径
    content_type VARCHAR(100) NOT NULL,
    -- 该版本MIME类型
    created_at BIGINT NOT NULL,
    -- 版本创建时间(Unix时间戳)
    deleted_at BIGINT DEFAULT NULL,
    -- 软删除时间(Unix时间戳)，NULL表示未删除
    status SMALLINT NOT NULL DEFAULT 0,
    -- 版本状态: 0=正常, 1=过时, 2=已替换
    commit_message TEXT,
    -- 版本提交信息
    -- 外键和约束
    CONSTRAINT fk_file_versions_file FOREIGN KEY (file_id) REFERENCES files(file_id) ON DELETE CASCADE,
    CONSTRAINT uk_file_version UNIQUE (file_id, version_number)
);
-- PostgreSQL索引
CREATE INDEX idx_file_versions_file_id ON file_versions(file_id);
-- 标签表
CREATE TABLE IF NOT EXISTS tags (
    tag_id VARCHAR(64) PRIMARY KEY,
    -- 标签ID
    name VARCHAR(100) NOT NULL,
    -- 标签名称
    -- 唯一约束
    CONSTRAINT uk_tag_name UNIQUE (name)
);
-- 文件-标签关联表
CREATE TABLE IF NOT EXISTS file_tags (
    file_id VARCHAR(64) NOT NULL,
    -- 文件ID
    tag_id VARCHAR(64) NOT NULL,
    -- 标签ID
    -- 联合主键
    PRIMARY KEY (file_id, tag_id),
    -- 外键
    CONSTRAINT fk_file_tags_file FOREIGN KEY (file_id) REFERENCES files(file_id) ON DELETE CASCADE,
    CONSTRAINT fk_file_tags_tag FOREIGN KEY (tag_id) REFERENCES tags(tag_id) ON DELETE CASCADE
);
-- 用户文件统计视图
CREATE VIEW user_file_stats AS
SELECT user_id,
    COUNT(*) as total_files,
    SUM(size) as total_size,
    MAX(updated_at) as last_activity
FROM files
GROUP BY user_id;