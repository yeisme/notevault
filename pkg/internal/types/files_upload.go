package types

// UploadFilesRequestPolicy 批量文件上传请求.
type UploadFilesRequestPolicy struct {
	Files []UploadFileItem `binding:"required" json:"files"`
}

// UploadFilesResponsePolicy 预签名上传结果.
type UploadFilesResponsePolicy struct {
	Results []PresignedUploadItem `json:"results"`
}

// UploadFilesRequest 批量文件上传请求 (PUT 不带策略).
type UploadFilesRequest struct {
	Files []UploadFileItem `binding:"required" json:"files"`
}

// UploadFilesResponse 预签名上传结果 (PUT).
type UploadFilesResponse struct {
	Results []PresignedPutItem `json:"results"`
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

// PresignedPutItem 预签名 PUT 上传单个结果项.
type PresignedPutItem struct {
	ObjectKey string `json:"object_key"` // 对象键 (上传后的路径)
	PutURL    string `json:"put_url"`    // 上传 URL
	ExpiresIn int    `json:"expires_in"` // 过期时间 (秒)
}

// UploadFileMetadata 上传文件元数据.
type UploadFileMetadata struct {
	FileName     string            `form:"file_name"     json:"file_name,omitempty"`     // 可选：文件名
	Tags         map[string]string `form:"tags"          json:"tags,omitempty"`          // 可选：标签
	Description  string            `form:"description"   json:"description,omitempty"`   // 可选：描述
	ContentType  string            `form:"content_type"  json:"content_type,omitempty"`  // 可选：内容类型
	Category     string            `form:"category"      json:"category,omitempty"`      // 可选：分类
	Folder       string            `form:"folder"        json:"folder,omitempty"`        // 可选：文件夹
	IsPublic     bool              `form:"is_public"     json:"is_public,omitempty"`     // 可选：是否公开
	ExpiryDays   int               `form:"expiry_days"   json:"expiry_days,omitempty"`   // 可选：过期天数
	LastModified string            `form:"last_modified" json:"last_modified,omitempty"` // 可选：最后修改时间 (RFC3339格式)
}

// UploadFileResponse 单个文件上传响应.
type UploadFileResponse struct {
	ObjectKey    string            `json:"object_key"`
	Hash         string            `json:"hash"`
	Size         int64             `json:"size"`
	ETag         string            `json:"etag,omitempty"`
	LastModified string            `json:"last_modified,omitempty"`
	VersionID    string            `json:"version_id,omitempty"`
	Bucket       string            `json:"bucket,omitempty"`
	Location     string            `json:"location,omitempty"`
	FileName     string            `json:"file_name,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	Description  string            `json:"description,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Success      bool              `json:"success"`
	Error        string            `json:"error,omitempty"`
}

// UploadBatchFilesResponse 批量文件上传响应.
type UploadBatchFilesResponse struct {
	Results    []UploadFileResponse `json:"results"`
	Total      int                  `json:"total"`
	Successful int                  `json:"successful"`
	Failed     int                  `json:"failed"`
}
