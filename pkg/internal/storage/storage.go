// Package storage 处理存储操作，如上传、下载和删除文件到S3，数据库等.
//
// Example:
//
// 初始化
//
//	 ctx := context.Background()
//	 ctx, err := storage.Init(ctx)
//
//		if err != nil {
//		    // 处理错误
//		}
//
// 获取 s3 客户端
//
//	mgr := storage.GetFromContext(ctx)
//	s3Client := storage.GetS3ClientFromContext(ctx)
package storage

import (
	"context"
	"sync"

	"github.com/yeisme/notevault/pkg/configs"
	s3c "github.com/yeisme/notevault/pkg/internal/storage/s3"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// Manager 聚合所有存储资源.
type Manager struct {
	S3 *s3c.Client
}

var (
	mgr     *Manager
	mgrOnce sync.Once
)

// Init 初始化默认存储，使用全局配置.重复调用只返回已初始化实例.
func Init(ctx context.Context) (*Manager, error) {
	var err error

	mgrOnce.Do(func() {
		cfg := configs.GetConfig()
		m := &Manager{}

		// S3
		if s3i, e := s3c.New(ctx, &cfg.S3); e != nil {
			err = e

			return
		} else {
			m.S3 = s3i
		}

		mgr = m

		nlog.Logger().Info().Msg("storage manager initialized")
	})

	return mgr, err
}

// Close 释放资源.
func (m *Manager) Close() {
	if m == nil {
		return
	}

	// S3 客户端目前无需显式关闭
}
