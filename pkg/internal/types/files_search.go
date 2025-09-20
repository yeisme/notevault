// Package types 定义应用程序中使用的各种数据类型和结构体. 主要为 Request 和 Response 结构体.
package types

import "time"

// SearchFilesRequest 高级搜索请求（POST）.
// 若未指定时间范围，则不限制；Page 从 1 开始.
type SearchFilesRequest struct {
	// 关键字将在文件名、描述、标签值中进行 LIKE 匹配
	Keyword string `json:"keyword,omitempty"`
	// 文件夹前缀过滤（例如 "2025/09/" 或某目录前缀）
	Prefix string `json:"prefix,omitempty"`
	// 分类过滤
	Category string `json:"category,omitempty"`
	// 内容类型（MIME）过滤
	ContentType string `json:"content_type,omitempty"`
	// 最小/最大大小（字节）
	MinSize int64 `json:"min_size,omitempty"`
	MaxSize int64 `json:"max_size,omitempty"`
	// 时间范围（对象 last_modified）
	Start time.Time `json:"start_time,omitzero"`
	End   time.Time `json:"end_time,omitzero"`
	// 分页
	Page     int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`
	// 排序字段：last_modified|size，默认 last_modified desc
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"` // asc|desc
}

// SearchFilesResponse 高级搜索响应.
type SearchFilesResponse struct {
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Size  int          `json:"size"`
	Files []ObjectInfo `json:"files"`
}
