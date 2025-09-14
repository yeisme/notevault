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
//	kvClient := mgr.GetKVClient()
package storage

import (
	"context"
	"errors"

	dbc "github.com/yeisme/notevault/pkg/internal/storage/db"
	kvc "github.com/yeisme/notevault/pkg/internal/storage/kv"
	mqc "github.com/yeisme/notevault/pkg/internal/storage/mq"
	s3c "github.com/yeisme/notevault/pkg/internal/storage/s3"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// Manager 聚合所有存储资源.
type Manager struct {
	s3 *s3c.Client
	db *dbc.Client
	mq *mqc.Client
	kv *kvc.Client
}

// Init 初始化存储管理器，使用全局配置.每次调用都创建新的实例以支持配置热重载.
func Init(ctx context.Context) (*Manager, error) {
	var collectedErrs []error

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

	if kvMgr, e := kvc.New(ctx); e != nil {
		nlog.Logger().Error().Err(e).Msg("init kv failed")
		collectedErrs = append(collectedErrs, e)
	} else {
		m.kv = kvMgr
	}

	var err error
	if len(collectedErrs) > 0 {
		// 使用 errors.Join 聚合（Go 1.20+）
		err = errors.Join(collectedErrs...)
	}

	nlog.Logger().Info().Msg("storage manager initialized")

	return m, err
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

// GetKVClient 获取 KV 客户端.
func (m *Manager) GetKVClient() *kvc.Client {
	return m.kv
}

// Close 关闭所有存储客户端连接，返回可能的错误（聚合）.
// TODO: 增加 Close 方法，关闭所有存储客户端连接. 在优雅关闭时调用.
func (m *Manager) Close() error {
	var collectedErrs []error

	if m.s3 != nil {
		if err := m.s3.Close(); err != nil {
			collectedErrs = append(collectedErrs, err)
		}
	}

	if m.db != nil {
		if err := m.db.Close(); err != nil {
			collectedErrs = append(collectedErrs, err)
		}
	}

	if m.mq != nil {
		if err := m.mq.Close(); err != nil {
			collectedErrs = append(collectedErrs, err)
		}
	}

	if m.kv != nil {
		if err := m.kv.Close(); err != nil {
			collectedErrs = append(collectedErrs, err)
		}
	}

	var err error
	if len(collectedErrs) > 0 {
		err = errors.Join(collectedErrs...)
	}

	return err
}
