package types

// MetaUpdateRequest 单对象元数据更新请求（不含对象键，对象键来自路由 path 参数）。
type MetaUpdateRequest struct {
	Tags        map[string]string `json:"tags,omitempty"`
	Description string            `json:"description,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	Category    string            `json:"category,omitempty"`
	IsPublic    *bool             `json:"is_public,omitempty"`
}

// MetaBatchRequest 批量获取对象元数据请求。
type MetaBatchRequest struct {
	ObjectKeys []string `binding:"required" json:"object_keys"`
}

// MetaBatchResponse 批量元数据响应。
type MetaBatchResponse struct {
	Files []ObjectInfo `json:"files"`
}

// MetaURLRequest 获取元数据（HEAD）预签名 URL 请求。
type MetaURLRequest struct {
	ExpirySeconds int `json:"expiry_seconds,omitempty"`
}
