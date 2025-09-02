// Package storage 处理存储操作，如上传、下载和删除文件到S3，数据库等.
//
// Example:
//
// 初始化
//
//	 ctx := context.Background()
//	 mgr, err := storage.Init(ctx)
//
//		if err != nil {
//		    // 处理错误
//		}
//
// 获取存储客户端
//
//	s3Client := mgr.GetS3Client()
//	dbClient := mgr.GetDBClient()
package storage

import (
	"context"
	"sync"

	"github.com/yeisme/notevault/pkg/configs"
	dbc "github.com/yeisme/notevault/pkg/internal/storage/db"
	s3c "github.com/yeisme/notevault/pkg/internal/storage/s3"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// Manager 聚合所有存储资源.
type Manager struct {
	S3 *s3c.Client
	DB *dbc.Client
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

		// DB
		if dbi, e := dbc.New(ctx, &cfg.DB); e != nil {
			err = e
		} else {
			m.DB = dbi
		}

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

// GetS3Client 获取 S3 客户端.
func (m *Manager) GetS3Client() *s3c.Client {
	return m.S3
}

// GetDBClient 获取 DB 客户端.
func (m *Manager) GetDBClient() *dbc.Client {
	return m.DB
}
