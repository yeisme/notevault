// Package kv 提供用于键值存储的接口和实现.
package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/yeisme/notevault/pkg/configs"
)

type Client struct {
	KVStore
}

// KVStore 定义键值存储接口.
type KVStore interface {
	// Get 获取键的值.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set 设置键的值，可选过期时间.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Delete 删除键.
	Delete(ctx context.Context, key string) error
	// Exists 检查键是否存在.
	Exists(ctx context.Context, key string) (bool, error)
	// Keys 获取所有键（可选，用于调试）.
	Keys(ctx context.Context, pattern string) ([]string, error)
	// Close 关闭存储连接.
	Close() error
}

// KVType 键值存储类型.
type KVType string

const (
	KVTypeRedis      KVType = "redis"
	KVTypeNATS       KVType = "nats"
	KVTypeGroupcache KVType = "groupcache"
)

// KVFactory 定义创建 KVStore 的工厂函数类型.
type KVFactory func(ctx context.Context, config any) (KVStore, error)

// kvFactories 存储 KV 类型到工厂的映射.
var kvFactories = make(map[KVType]KVFactory)

// RegisterKVFactory 注册 KV 工厂函数.
func RegisterKVFactory(kvType KVType, factory KVFactory) {
	kvFactories[kvType] = factory
}

// GetRegisteredKVTypes 返回已注册的 KV 类型列表.
func GetRegisteredKVTypes() []KVType {
	types := make([]KVType, 0, len(kvFactories))
	for kvType := range kvFactories {
		types = append(types, kvType)
	}

	return types
}

// NewKVStore 根据类型创建 KVStore 实例.
func NewKVStore(ctx context.Context, kvType KVType, config any) (KVStore, error) {
	factory, exists := kvFactories[kvType]
	if !exists {
		return nil, fmt.Errorf("unsupported KV type: %s", kvType)
	}

	return factory(ctx, config)
}

// NewKVClient 创建并返回一个新的 KVClient 实例.
func NewKVClient(ctx context.Context) (*Client, error) {
	cfg := configs.GetConfig().KV

	store, err := NewKVStore(ctx, KVType(cfg.Type), cfg)
	if err != nil {
		return nil, err
	}

	return &Client{KVStore: store}, nil
}
