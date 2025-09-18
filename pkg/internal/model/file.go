package model

import (
	"time"

	"gorm.io/gorm"
)

// Files 文件模型.
type Files struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// 用户名或租户标识，和对象键一起唯一
	User string `gorm:"size:255;index:idx_user_key,unique;index" json:"user"`
	// 对象键（S3 key），确保在 user 下唯一
	ObjectKey   string `gorm:"size:1024;index:idx_user_key,unique;index" json:"object_key"`
	FileName    string `gorm:"size:512;index"                            json:"file_name"`
	Size        int64  `gorm:"index"                                     json:"size"`
	ETag        string `gorm:"size:64"                                   json:"etag"`
	ContentType string `gorm:"size:255;index"                            json:"content_type"`
	Category    string `gorm:"size:128;index"                            json:"category"`
	Description string `gorm:"type:text"                                 json:"description"`
	// Tags 以 JSON 字符串形式存储，便于模糊搜索；未来可替换为 JSONB
	TagsJSON     string `gorm:"type:text" json:"tags_json"`
	Bucket       string `gorm:"size:255"  json:"bucket"`
	VersionID    string `gorm:"size:255"  json:"version_id"`
	StorageClass string `gorm:"size:64"   json:"storage_class"`
	// 来自对象存储的最后修改时间
	LastModified time.Time `gorm:"index" json:"last_modified"`
	// 软删除与审计
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
