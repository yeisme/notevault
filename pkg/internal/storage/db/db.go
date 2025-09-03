// Package db 处理数据库存储操作.
package db

import (
	"context"
	"fmt"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/yeisme/notevault/pkg/configs"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// DialectorFactory 定义创建 dialector 的函数类型.
type DialectorFactory func(dsn string) gorm.Dialector

// dialectorFactories 存储数据库类型到 dialector 工厂的映射.
var dialectorFactories = map[configs.DBType]DialectorFactory{}

// RegisterDialectorFactory 注册数据库 dialector 工厂函数.
func RegisterDialectorFactory(dbType configs.DBType, factory DialectorFactory) {
	dialectorFactories[dbType] = factory
}

// GetRegisteredDBTypes 返回已注册的数据库类型列表.
func GetRegisteredDBTypes() []configs.DBType {
	types := make([]configs.DBType, 0, len(dialectorFactories))
	for dbType := range dialectorFactories {
		types = append(types, dbType)
	}

	return types
}

// DBManager 数据库实例管理器.
type DBManager struct {
	instances *Client
	mu        sync.RWMutex
}

var globalDBManager = &DBManager{
	instances: nil,
}

// Client 包装 GORM DB 客户端.
type Client struct {
	*gorm.DB
}

func New(ctx context.Context, cfg *configs.DBConfig) (*Client, error) {
	globalDBManager.mu.Lock()
	defer globalDBManager.mu.Unlock()

	dsn := cfg.GetDSN()
	if dsn == "" {
		return nil, fmt.Errorf("failed to generate DSN for database type: %s", cfg.Type)
	}

	factory, exists := dialectorFactories[cfg.Type]
	if !exists {
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	dialector := factory(dsn)

	// 配置 GORM 日志
	gormLogger := logger.New(
		nlog.Logger(),
		logger.Config{
			SlowThreshold:             0, // 慢查询阈值，0表示不记录慢查询
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:               gormLogger,
		PrepareStmt:          true,
		FullSaveAssociations: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层 SQL DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	// 测试连接
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	client := &Client{DB: db}

	nlog.Logger().Info().
		Str("type", cfg.GetDBType()).
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Msg("database connected successfully")

	return client, nil
}

// GetDB 返回 GORM DB 实例.
func (c *Client) GetDB() *gorm.DB {
	return c.DB
}
