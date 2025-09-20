package types

// GetFilesURLRequest 批量获取文件访问 URL 请求.
type GetFilesURLRequest struct {
	// 对象列表（支持单个/批量）
	Objects []GetFileURLItem `binding:"required" json:"objects"`
	// 过期时间（秒），可选；缺省使用服务默认值
	ExpirySeconds int `json:"expiry_seconds,omitempty"`
}

// GetFileURLItem 单个对象的访问 URL 请求项.
type GetFileURLItem struct {
	ObjectKey string `binding:"required" json:"object_key"`

	// 以下为可选的响应头控制参数（S3 预签名支持的常见字段）
	ResponseContentType        string `json:"response_content_type,omitempty"`
	ResponseContentDisposition string `json:"response_content_disposition,omitempty"`
	ResponseCacheControl       string `json:"response_cache_control,omitempty"`
	ResponseContentLanguage    string `json:"response_content_language,omitempty"`
	ResponseContentEncoding    string `json:"response_content_encoding,omitempty"`
}

// GetFilesURLResponse 批量获取文件访问 URL 响应.
type GetFilesURLResponse struct {
	Results []PresignedDownloadItem `json:"results"`
}

// PresignedDownloadItem 预签名下载结果项.
type PresignedDownloadItem struct {
	ObjectKey string `json:"object_key"`
	GetURL    string `json:"get_url"`
	ExpiresIn int    `json:"expires_in"`
}

// DownloadFilesRequest 直传下载请求（支持单个/批量）.
// 当 `archive=true` 且包含多个对象时，服务端将以 zip 流式返回.
type DownloadFilesRequest struct {
	Objects     []DownloadObjectItem `binding:"required"            json:"objects"`
	Archive     bool                 `json:"archive,omitempty"`      // 可选：当为 true 且对象数量>1 时，打包为 zip 返回
	ArchiveName string               `json:"archive_name,omitempty"` // 可选：打包文件名（仅当 Archive=true 或多文件时生效），例如 my-photos.zip
}

// DownloadObjectItem 下载对象项.
type DownloadObjectItem struct {
	ObjectKey string `binding:"required" json:"object_key"`
	// 可选：指定在 zip 中的文件名；未提供则使用对象键的最后一段
	FileName string `json:"file_name,omitempty"`
}

// ObjectInfo 对象信息（用于返回给客户端展示）.
type ObjectInfo struct {
	ObjectKey    string            `json:"object_key"`
	Size         int64             `json:"size"`
	ETag         string            `json:"etag,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	LastModified string            `json:"last_modified,omitempty"` // RFC3339
	VersionID    string            `json:"version_id,omitempty"`
	StorageClass string            `json:"storage_class,omitempty"`
	Bucket       string            `json:"bucket,omitempty"`
	UserMetadata map[string]string `json:"user_metadata,omitempty"`
}

// DownloadFilesResponse 批量下载时可返回每个对象的 info.
// 当直传单文件时，响应为二进制流，不使用该结构；当打包下载或纯信息请求时才使用.
type DownloadFilesResponse struct {
	Files []ObjectInfo `json:"files"`
}
