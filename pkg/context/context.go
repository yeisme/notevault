// Package context 拓展上下文功能，将日志、服务等集成到上下文中，方便在应用程序各处传递和使用.
package context

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"

	"github.com/yeisme/notevault/pkg/internal/storage"
	dbc "github.com/yeisme/notevault/pkg/internal/storage/db"
	kvc "github.com/yeisme/notevault/pkg/internal/storage/kv"
	mqc "github.com/yeisme/notevault/pkg/internal/storage/mq"
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

// GetManager 从 context 中获取 Manager.
func GetManager(ctx context.Context) *storage.Manager {
	if mgr, ok := ctx.Value(StorageManagerKey).(*storage.Manager); ok {
		return mgr
	}

	return nil
}

// GetS3Client 从 context 中获取 S3 客户端.
func GetS3Client(ctx context.Context) *s3c.Client {
	if mgr := GetManager(ctx); mgr != nil {
		return mgr.GetS3Client()
	}

	return nil
}

// GetDBClient 从 context 中获取 DB 客户端.
func GetDBClient(ctx context.Context) *dbc.Client {
	if mgr := GetManager(ctx); mgr != nil {
		return mgr.GetDBClient()
	}

	return nil
}

// GetMQClient 从 context 中获取 MQ 客户端.
func GetMQClient(ctx context.Context) *mqc.Client {
	if mgr := GetManager(ctx); mgr != nil {
		return mgr.GetMQClient()
	}

	return nil
}

// GetKVClient 从 context 中获取 KV 客户端.
func GetKVClient(ctx context.Context) *kvc.Client {
	if mgr := GetManager(ctx); mgr != nil {
		return mgr.GetKVClient()
	}

	return nil
}

// WithTraceContext 创建带有追踪上下文的logger.
func WithTraceContext(ctx context.Context, logger zerolog.Logger) zerolog.Logger {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		return logger.With().
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String()).
			Logger()
	}

	return logger
}
