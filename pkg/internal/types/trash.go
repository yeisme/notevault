package types

import "time"

// TrashListResponse 回收站列表响应.
type TrashListResponse struct {
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Size  int          `json:"size"`
	Files []ObjectInfo `json:"files"`
}

// TrashBatchRequest 批量恢复/删除请求.
type TrashBatchRequest struct {
	ObjectKeys []string `binding:"required" json:"object_keys"`
}

// TrashActionResponse 通用动作响应.
type TrashActionResponse struct {
	Affected int    `json:"affected"`
	Message  string `json:"message,omitempty"`
}

// TrashAutoCleanRequest 自动清理请求.
// 可指定 before（RFC3339）或 days（整数，表示清理 N 天前删除的）.
type TrashAutoCleanRequest struct {
	Before string `json:"before,omitempty"`
	Days   int    `json:"days,omitempty"`
}

// ParseBefore 返回解析后的时间与是否提供.
func (r *TrashAutoCleanRequest) ParseBefore() (time.Time, bool) {
	if r.Before != "" {
		if t, err := time.Parse(time.RFC3339, r.Before); err == nil {
			return t, true
		}
	}

	if r.Days > 0 {
		return time.Now().UTC().Add(-time.Duration(r.Days) * 24 * time.Hour), true
	}

	return time.Time{}, false
}
