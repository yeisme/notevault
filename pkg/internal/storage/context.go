package storage

import (
	"context"

	s3c "github.com/yeisme/notevault/pkg/internal/storage/s3"
)

type contextKey string

const managerKey contextKey = "storageManager"

// WithManager 将 Manager 存储到 context 中.
func WithManager(ctx context.Context, mgr *Manager) context.Context {
	return context.WithValue(ctx, managerKey, mgr)
}

// GetManagerFromContext 从 context 中获取 Manager.
func GetManagerFromContext(ctx context.Context) *Manager {
	if mgr, ok := ctx.Value(managerKey).(*Manager); ok {
		return mgr
	}

	return nil
}

// GetS3ClientFromContext 从 context 中获取 S3 客户端.
func GetS3ClientFromContext(ctx context.Context) *s3c.Client {
	if mgr := GetManagerFromContext(ctx); mgr != nil {
		return mgr.S3
	}

	return nil
}
