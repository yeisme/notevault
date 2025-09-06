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
