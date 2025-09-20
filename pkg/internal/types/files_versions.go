package types

// FileVersionInfo 表示单个文件版本的元信息.
type FileVersionInfo struct {
	ObjectKey    string            `json:"object_key"`
	VersionID    string            `json:"version_id"`
	IsLatest     bool              `json:"is_latest"`
	Size         int64             `json:"size"`
	ETag         string            `json:"etag,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	LastModified string            `json:"last_modified,omitempty"` // RFC3339
	StorageClass string            `json:"storage_class,omitempty"`
	Bucket       string            `json:"bucket,omitempty"`
	UserMetadata map[string]string `json:"user_metadata,omitempty"`
}

// ListFileVersionsResponse 获取文件版本列表响应.
type ListFileVersionsResponse struct {
	FileID   string            `json:"file_id"`
	Versions []FileVersionInfo `json:"versions"`
	Total    int               `json:"total"`
}

// CreateFileVersionRequest 创建新版本请求.
// 这里的语义：从现有对象拷贝为一个新版本（或指定版本ID作为基准）.
// 为保持最小实现，我们允许可选的 user_metadata/content_type 覆盖.
type CreateFileVersionRequest struct {
	ObjectKey   string            `binding:"required"             json:"object_key"`
	BaseVersion string            `json:"base_version,omitempty"`  // 可选：基于哪个版本创建
	ContentType string            `json:"content_type,omitempty"`  // 可选：覆盖内容类型
	UserMeta    map[string]string `json:"user_metadata,omitempty"` // 可选：覆盖用户元数据
}

// CreateFileVersionResponse 创建版本响应.
type CreateFileVersionResponse struct {
	ObjectKey string `json:"object_key"`
	VersionID string `json:"version_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// DeleteFileVersionResponse 删除指定版本响应.
type DeleteFileVersionResponse struct {
	ObjectKey string `json:"object_key"`
	VersionID string `json:"version_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// RestoreFileVersionResponse 恢复指定版本为最新版本的响应.
type RestoreFileVersionResponse struct {
	ObjectKey   string `json:"object_key"`
	FromVersion string `json:"from_version"`
	RestoredAs  string `json:"restored_as"` // 新产生的最新版本ID（如后端生成）
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}
