package types

import "time"

// CreateShareRequest 创建分享所需参数.
type CreateShareRequest struct {
	// ObjectKeys 需要分享的对象键（S3 Key）列表，按创建时快照保存
	ObjectKeys []string `form:"object_keys" json:"object_keys"`
	// Password 可选访问密码（服务端仅存储密码哈希）
	Password string `form:"password" json:"password"`
	// ExpireDays 分享有效天数；>0 则按天计算过期时间，为 0 表示不过期
	ExpireDays int `form:"expire_days" json:"expire_days"`
	// AllowDownload 是否允许生成下载直链
	AllowDownload bool `form:"allow_download" json:"allow_download"`
}

// ShareInfo 分享的公开信息。
type ShareInfo struct {
	// ShareID 分享唯一标识（URL 公开使用）
	ShareID string `json:"share_id"`
	// Owner 分享拥有者（用户名或租户标识）
	Owner string `json:"owner"`
	// ObjectKeys 分享包含的对象键列表
	ObjectKeys []string `json:"object_keys"`
	// CreatedAt 分享创建时间（UTC）
	CreatedAt time.Time `json:"created_at"`
	// ExpireAt 分享过期时间（UTC，可为空表示不过期）
	ExpireAt *time.Time `json:"expire_at,omitempty"`
	// AllowDownload 是否允许下载直链
	AllowDownload bool `json:"allow_download"`
}

// CreateShareResponse 创建分享的响应体。
type CreateShareResponse struct {
	// Share 创建成功的分享信息
	Share ShareInfo `json:"share"`
}

// ListSharesResponse 分享列表响应体。
type ListSharesResponse struct {
	// Shares 当前用户的分享列表（已过滤过期项）
	Shares []ShareInfo `json:"shares"`
}

// AccessShareRequest 访问分享请求（用于校验可选密码）。
type AccessShareRequest struct {
	// Password 访问密码（若分享设置了密码）
	Password string `form:"password" json:"password"`
}

// SharePermissions 分享权限配置。
type SharePermissions struct {
	// AllowAnonymous 是否允许匿名访问（设了密码则仍需校验密码）
	AllowAnonymous bool `json:"allow_anonymous"`
	// Users 允许访问的用户白名单（Owner 总是隐含允许）
	Users []string `json:"users"`
}

// GetSharePermissionsResponse 获取分享权限响应体。
type GetSharePermissionsResponse struct {
	// ShareID 目标分享 ID
	ShareID string `json:"share_id"`
	// Permissions 权限配置
	Permissions SharePermissions `json:"permissions"`
}

// UpdateSharePermissionsRequest 更新分享权限请求体。
type UpdateSharePermissionsRequest struct {
	// AllowAnonymous 是否允许匿名访问
	AllowAnonymous bool `json:"allow_anonymous"`
	// Users 允许访问的用户白名单
	Users []string `json:"users"`
}

// AddShareUserRequest 添加分享用户请求体。
type AddShareUserRequest struct {
	// UserID 被添加的目标用户 ID/标识
	UserID string `form:"user_id" json:"user_id"`
}
