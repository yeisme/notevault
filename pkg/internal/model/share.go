package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	itypes "github.com/yeisme/notevault/pkg/internal/types"
)

// Share 数据库模型：以 DB 为真源，部分字段以 JSON 文本存储以保持实现简单。
// 注意：后续如需复杂查询/统计，可拆为 share_objects / share_users 关联表。
type Share struct {
	ShareID         string         `gorm:"primaryKey;size:64" json:"share_id"`
	Owner           string         `gorm:"size:255;index"     json:"owner"`
	ObjectKeysJSON  string         `gorm:"type:text"          json:"-"`
	AllowDownload   bool           `json:"allow_download"`
	PasswordHash    string         `gorm:"size:128"           json:"-"`
	PermissionsJSON string         `gorm:"type:text"          json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	ExpireAt        *time.Time     `gorm:"index"              json:"expire_at,omitempty"`
	DeletedAt       gorm.DeletedAt `gorm:"index"              json:"-"`
}

// ShareRecord 供 service 层使用的内部结构（与 service/shares.go 中保持一致）。
// 这里重复定义一个轻量结构，避免 service 直接依赖 model 的 JSON 细节。
type ShareRecord struct {
	ShareID       string
	Owner         string
	ObjectKeys    []string
	CreatedAt     time.Time
	ExpireAt      *time.Time
	AllowDownload bool
	PasswordHash  string
	Permissions   itypes.SharePermissions
}

// ToRecord 将 DB 模型反序列化为 ShareRecord。
func (s *Share) ToRecord() (*ShareRecord, error) {
	var keys []string
	if s.ObjectKeysJSON != "" {
		if err := json.Unmarshal([]byte(s.ObjectKeysJSON), &keys); err != nil {
			return nil, fmt.Errorf("unmarshal object_keys: %w", err)
		}
	}

	var perms itypes.SharePermissions
	if s.PermissionsJSON != "" {
		if err := json.Unmarshal([]byte(s.PermissionsJSON), &perms); err != nil {
			return nil, fmt.Errorf("unmarshal permissions: %w", err)
		}
	}

	return &ShareRecord{
		ShareID:       s.ShareID,
		Owner:         s.Owner,
		ObjectKeys:    keys,
		CreatedAt:     s.CreatedAt,
		ExpireAt:      s.ExpireAt,
		AllowDownload: s.AllowDownload,
		PasswordHash:  s.PasswordHash,
		Permissions:   perms,
	}, nil
}

// FromRecord 将 ShareRecord 序列化为 DB 模型。
func FromRecord(r *ShareRecord) (*Share, error) {
	objBytes, err := json.Marshal(r.ObjectKeys)
	if err != nil {
		return nil, fmt.Errorf("marshal object_keys: %w", err)
	}

	permBytes, err := json.Marshal(r.Permissions)
	if err != nil {
		return nil, fmt.Errorf("marshal permissions: %w", err)
	}

	return &Share{
		ShareID:         r.ShareID,
		Owner:           r.Owner,
		ObjectKeysJSON:  string(objBytes),
		AllowDownload:   r.AllowDownload,
		PasswordHash:    r.PasswordHash,
		PermissionsJSON: string(permBytes),
		CreatedAt:       r.CreatedAt,
		ExpireAt:        r.ExpireAt,
	}, nil
}
