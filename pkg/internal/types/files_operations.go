package types

// DeleteFilesRequest 删除文件请求.
type DeleteFilesRequest struct {
	ObjectKeys []string `binding:"required" json:"object_keys"` // 要删除的文件对象键列表
}

// DeleteFilesResponse 删除文件响应.
type DeleteFilesResponse struct {
	Results []DeleteFileResult `json:"results"`
	Total   int                `json:"total"`
	Success int                `json:"success"`
	Failed  int                `json:"failed"`
}

// DeleteFileResult 删除单个文件结果.
type DeleteFileResult struct {
	ObjectKey string `json:"object_key"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// UpdateFilesMetadataRequest 更新文件元数据请求.
type UpdateFilesMetadataRequest struct {
	Items []UpdateFileMetadataItem `binding:"required" json:"items"`
}

// UpdateFileMetadataItem 更新单个文件的元数据.
type UpdateFileMetadataItem struct {
	ObjectKey   string            `binding:"required"            json:"object_key"`
	Tags        map[string]string `json:"tags,omitempty"`
	Description string            `json:"description,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	Category    string            `json:"category,omitempty"`
	IsPublic    *bool             `json:"is_public,omitempty"` // 使用指针以区分未设置和false
}

// UpdateFilesMetadataResponse 更新文件元数据响应.
type UpdateFilesMetadataResponse struct {
	Results []UpdateFileMetadataResult `json:"results"`
	Total   int                        `json:"total"`
	Success int                        `json:"success"`
	Failed  int                        `json:"failed"`
}

// UpdateFileMetadataResult 更新单个文件元数据结果.
type UpdateFileMetadataResult struct {
	ObjectKey string `json:"object_key"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// CopyFilesRequest 复制文件请求.
type CopyFilesRequest struct {
	Items []CopyFileItem `binding:"required" json:"items"`
}

// CopyFileItem 复制单个文件.
type CopyFileItem struct {
	SourceKey      string `binding:"required" json:"source_key"`      // 源文件对象键
	DestinationKey string `binding:"required" json:"destination_key"` // 目标文件对象键
}

// CopyFilesResponse 复制文件响应.
type CopyFilesResponse struct {
	Results []CopyFileResult `json:"results"`
	Total   int              `json:"total"`
	Success int              `json:"success"`
	Failed  int              `json:"failed"`
}

// CopyFileResult 复制单个文件结果.
type CopyFileResult struct {
	SourceKey      string `json:"source_key"`
	DestinationKey string `json:"destination_key"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}

// MoveFilesRequest 移动文件请求.
type MoveFilesRequest struct {
	Items []MoveFileItem `binding:"required" json:"items"`
}

// MoveFileItem 移动单个文件.
type MoveFileItem struct {
	SourceKey      string `binding:"required" json:"source_key"`      // 源文件对象键
	DestinationKey string `binding:"required" json:"destination_key"` // 目标文件对象键
}

// MoveFilesResponse 移动文件响应.
type MoveFilesResponse struct {
	Results []MoveFileResult `json:"results"`
	Total   int              `json:"total"`
	Success int              `json:"success"`
	Failed  int              `json:"failed"`
}

// MoveFileResult 移动单个文件结果.
type MoveFileResult struct {
	SourceKey      string `json:"source_key"`
	DestinationKey string `json:"destination_key"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}
