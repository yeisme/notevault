// Package context 拓展上下文功能，将日志、服务等集成到上下文中，方便在应用程序各处传递和使用.
package context

import (
	"context"

	"github.com/yeisme/notevault/pkg/internal/storage"
	dbc "github.com/yeisme/notevault/pkg/internal/storage/db"
	s3c "github.com/yeisme/notevault/pkg/internal/storage/s3"
)

type ContextKey string

const (
	StorageManagerKey ContextKey = "storageManager"
)

// WithStorageManager 将 Manager 存储到 context 中.
func WithStorageManager(ctx context.Context, mgr *storage.Manager) context.Context {
	return context.WithValue(ctx, StorageManagerKey, mgr)
}

// GetManagerFromContext 从 context 中获取 Manager.
func GetManagerFromContext(ctx context.Context) *storage.Manager {
	if mgr, ok := ctx.Value(StorageManagerKey).(*storage.Manager); ok {
		return mgr
	}

	return nil
}

// GetS3ClientFromContext 从 context 中获取 S3 客户端.
func GetS3ClientFromContext(ctx context.Context) *s3c.Client {
	if mgr := GetManagerFromContext(ctx); mgr != nil {
		return mgr.GetS3Client()
	}

	return nil
}

func GetDBClientFromContext(ctx context.Context) *dbc.Client {
	if mgr := GetManagerFromContext(ctx); mgr != nil {
		return mgr.GetDBClient()
	}

	return nil
}
