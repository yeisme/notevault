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
	"errors"
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
	var collectedErrs []error

	mgrOnce.Do(func() {
		m := &Manager{}

		// DB（失败不阻断后续）
		if dbi, e := dbc.New(ctx); e != nil {
			nlog.Logger().Error().Err(e).Msg("init db failed")
			collectedErrs = append(collectedErrs, e)
		} else {
			m.db = dbi
		}

		// S3（失败也不提前 return，保证 MQ 仍被尝试初始化）
		if s3i, e := s3c.New(ctx); e != nil {
			nlog.Logger().Error().Err(e).Msg("init s3 failed")
			collectedErrs = append(collectedErrs, e)
		} else {
			m.s3 = s3i
		}

		// MQ（同样收集错误）
		if mqMgr, e := mqc.New(ctx); e != nil {
			nlog.Logger().Error().Err(e).Msg("init mq failed")
			collectedErrs = append(collectedErrs, e)
		} else {
			m.mq = mqMgr
		}

		// 仅当至少有一个成功时才赋值 mgr；否则保持 nil 便于上层感知
		if m.db != nil || m.s3 != nil || m.mq != nil {
			mgr = m

			nlog.Logger().Info().Msg("storage manager initialized (partial possible)")
		}
	})

	var err error
	if len(collectedErrs) > 0 {
		// 使用 errors.Join 聚合（Go 1.20+）
		err = errors.Join(collectedErrs...)
	}

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
