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
//	mqClient := mgr.GetMQClient()
package storage

import (
	"context"
	"sync"

	dbc "github.com/yeisme/notevault/pkg/internal/storage/db"
	mqc "github.com/yeisme/notevault/pkg/internal/storage/mq"
	s3c "github.com/yeisme/notevault/pkg/internal/storage/s3"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// Manager 聚合所有存储资源.
type Manager struct {
	s3 *s3c.Client
	db *dbc.Client
	mq *mqc.Client
}

var (
	mgr     *Manager
	mgrOnce sync.Once
)

// Init 初始化默认存储，使用全局配置.重复调用只返回已初始化实例.
func Init(ctx context.Context) (*Manager, error) {
	var err error

	mgrOnce.Do(func() {
		m := &Manager{}

		// DB
		if dbi, e := dbc.New(ctx); e != nil {
			nlog.Logger().Error().Err(e).Msg("init db failed")
		} else {
			m.db = dbi
		}

		// S3
		if s3i, e := s3c.New(ctx); e != nil {
			nlog.Logger().Error().Err(e).Msg("init s3 failed")

			return
		} else {
			m.s3 = s3i
		}

		// MQ
		if mqMgr, e := mqc.New(ctx); e != nil {
			nlog.Logger().Error().Err(e).Msg("init mq failed")

			return
		} else {
			m.mq = mqMgr
		}

		mgr = m

		nlog.Logger().Info().Msg("storage manager initialized")
	})

	return mgr, err
}

// GetS3Client 获取 S3 客户端.
func (m *Manager) GetS3Client() *s3c.Client {
	return m.s3
}

// GetDBClient 获取 DB 客户端.
func (m *Manager) GetDBClient() *dbc.Client {
	return m.db
}

// GetMQClient 获取 MQ 客户端.
func (m *Manager) GetMQClient() *mqc.Client {
	return m.mq
}
