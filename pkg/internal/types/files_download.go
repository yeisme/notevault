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
