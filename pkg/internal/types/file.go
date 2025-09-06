package types

type UploadFileRequest struct {
	FileName string `form:"file_name" json:"file_name" rule:"required,max=255"` // 文件名
	FileType string `form:"file_type" json:"file_type" rule:"required,max=10"`  // 文件类型（扩展名）
	FileSize int64  `form:"file_size" json:"file_size"`                         // 文件大小（字节）
}

// PresignedUploadResult 预签名上传结果.
type PresignedUploadResult struct {
	ObjectKey string `json:"object_key"`
	PutURL    string `json:"put_url"`
	ExpiresIn int    `json:"expires_in"` // 过期时间（秒）
}
