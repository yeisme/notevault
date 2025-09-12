package types

// CreateFolderRequest 创建文件夹请求.
type CreateFolderRequest struct {
	Name        string `binding:"required"           json:"name"` // 文件夹名称
	Path        string `json:"path,omitempty"`                    // 父路径（可选）
	Description string `json:"description,omitempty"`             // 文件夹描述
	User        string `json:"user,omitempty"`                    // 所属用户（可选，默认当前用户，添加前缀）
}

// CreateFolderResponse 创建文件夹响应.
type CreateFolderResponse struct {
	FolderID  string `json:"folder_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	FullPath  string `json:"full_path"`
	CreatedAt string `json:"created_at"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// RenameFolderRequest 重命名文件夹请求.
type RenameFolderRequest struct {
	NewName string `binding:"required" json:"new_name"` // 新文件夹名称
}

// RenameFolderResponse 重命名文件夹响应.
type RenameFolderResponse struct {
	FolderID  string `json:"folder_id"`
	OldName   string `json:"old_name"`
	NewName   string `json:"new_name"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// DeleteFolderRequest 删除文件夹请求.
type DeleteFolderRequest struct {
	Recursive bool `json:"recursive,omitempty"` // 是否递归删除子文件夹和文件
}

// DeleteFolderResponse 删除文件夹响应.
type DeleteFolderResponse struct {
	FolderID     string `json:"folder_id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	DeletedFiles int    `json:"deleted_files,omitempty"` // 删除的文件数量
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}
