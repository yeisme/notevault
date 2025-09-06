package types

// UploadFilesRequestPolicy 批量文件上传请求.
type UploadFilesRequestPolicy struct {
	Files []UploadFileItem `binding:"required" json:"files"`
}

// UploadFilesResponsePolicy 预签名上传结果.
type UploadFilesResponsePolicy struct {
	Results []PresignedUploadItem `json:"results"`
}

// UploadFileItem 单个文件上传请求.
type UploadFileItem struct {
	FileName           string            `json:"file_name"`
	ContentType        string            `json:"content_type,omitempty"`        // 可选：内容类型
	MaxSize            int64             `json:"max_size,omitempty"`            // 可选：最大文件大小（字节）
	MinSize            int64             `json:"min_size,omitempty"`            // 可选：最小文件大小（字节）
	KeyStartsWith      string            `json:"key_starts_with,omitempty"`     // 可选：对象键前缀
	ContentDisposition string            `json:"content_disposition,omitempty"` // 可选：内容处置
	ContentEncoding    string            `json:"content_encoding,omitempty"`    // 可选：内容编码
	UserMetadata       map[string]string `json:"user_metadata,omitempty"`       // 可选：用户元数据
}

// PresignedUploadItem 预签名上传单个结果项.
type PresignedUploadItem struct {
	ObjectKey string            `json:"object_key"` // 对象键 (上传后的路径)
	PutURL    string            `json:"put_url"`    // 上传 URL
	FormData  map[string]string `json:"form_data"`  // 表单数据
	ExpiresIn int               `json:"expires_in"` // 过期时间 (秒)
}

// GetFilesURLRequest 批量获取文件访问 URL 请求.
type GetFilesURLRequest struct {
	// 对象列表（支持单个/批量）
	Objects []GetFileURLItem `binding:"required" json:"objects"`
	// 过期时间（秒），可选；缺省使用服务默认值
	ExpirySeconds int `json:"expiry_seconds,omitempty"`
}

// GetFileURLItem 单个对象的访问 URL 请求项。
type GetFileURLItem struct {
	ObjectKey string `binding:"required" json:"object_key"`

	// 以下为可选的响应头控制参数（S3 预签名支持的常见字段）
	ResponseContentType        string `json:"response_content_type,omitempty"`
	ResponseContentDisposition string `json:"response_content_disposition,omitempty"`
	ResponseCacheControl       string `json:"response_cache_control,omitempty"`
	ResponseContentLanguage    string `json:"response_content_language,omitempty"`
	ResponseContentEncoding    string `json:"response_content_encoding,omitempty"`
}

// GetFilesURLResponse 批量获取文件访问 URL 响应。
type GetFilesURLResponse struct {
	Results []PresignedDownloadItem `json:"results"`
}

// PresignedDownloadItem 预签名下载结果项。
type PresignedDownloadItem struct {
	ObjectKey string `json:"object_key"`
	GetURL    string `json:"get_url"`
	ExpiresIn int    `json:"expires_in"`
}
