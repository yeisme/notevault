// Package cache 提供基于键值存储的泛型缓存实现.
//
// 该包提供了类型安全的缓存操作，支持任意类型的缓存值.
// 底层使用JSON序列化/反序列化，支持TTL（生存时间）设置.
//
// 基本用法:
//
//	// 创建缓存实例
//	kvStore := // 获取KV存储实例
//	cache := cache.NewCache(kvStore)
//
//	// 缓存用户数据
//	user := User{ID: 1, Name: "Alice"}
//	err := cache.Set(ctx, "user:1", user, time.Hour)
//
//	// 获取缓存数据
//	cachedUser, err := cache.Get[User](ctx, "user:1")
//
//	// 使用GetOrSet模式
//	user, err := cache.GetOrSet(ctx, "user:1", func() (User, error) {
//	    return fetchUserFromDB(1)
//	}, time.Hour)
//
// 支持的KV存储类型:
//   - Redis
//   - NATS KV
//   - Groupcache
//   - 内存存储（如果实现）
//
// 线程安全:
//
//	该包不提供额外的线程安全保证，取决于底层的KV存储实现.
//	大多数KV存储（如Redis）是线程安全的.
//
// 性能考虑:
//   - JSON序列化/反序列化有一定的性能开销
//   - 对于高性能场景，未来考虑使用二进制序列化格式
//   - GetOrSet模式可以减少重复计算，但要注意getter函数的复杂度
//
// 错误处理:
//   - 网络错误、连接错误等会通过error返回
//   - 序列化/反序列化错误会被包装并返回
//   - 缓存未命中不会被视为错误
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/yeisme/notevault/pkg/internal/storage/kv"
)

// Cache 基于KV存储的缓存实现.
type Cache struct {
	kvStore kv.KVStore
}

// NewCache 创建一个新的缓存实例.
func NewCache(kvStore kv.KVStore) *Cache {
	return &Cache{
		kvStore: kvStore,
	}
}

// Get 泛型获取缓存值.
func Get[T any](ctx context.Context, c *Cache, key string) (T, error) {
	var zero T

	data, err := c.kvStore.Get(ctx, key)
	if err != nil {
		return zero, err
	}

	var value T
	if err := sonic.Unmarshal(data, &value); err != nil {
		return zero, fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return value, nil
}

// Set 泛型设置缓存值.
func Set[T any](ctx context.Context, c *Cache, key string, value T, ttl time.Duration) error {
	data, err := sonic.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	return c.kvStore.Set(ctx, key, data, ttl)
}

// Delete 删除缓存键.
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.kvStore.Delete(ctx, key)
}

// Exists 检查缓存键是否存在.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	return c.kvStore.Exists(ctx, key)
}

// GetOrSet 获取缓存值，如果不存在则设置.
func GetOrSet[T any](ctx context.Context, c *Cache, key string, getter func() (T, error), ttl time.Duration) (T, error) {
	var zero T

	// 尝试获取
	if value, err := Get[T](ctx, c, key); err == nil {
		return value, nil
	}

	// 获取新值
	value, err := getter()
	if err != nil {
		return zero, err
	}

	// 设置缓存
	if setErr := Set(ctx, c, key, value, ttl); setErr != nil {
		// 缓存失败，但仍返回值
		return value, nil
	}

	return value, nil
}

// Clear 清空缓存（如果支持）.
func (c *Cache) Clear(ctx context.Context) error {
	keys, err := c.kvStore.Keys(ctx, "*")
	if err != nil {
		return err
	}

	for _, key := range keys {
		// 部分KV存储可能不支持删除所有键
		if delErr := c.kvStore.Delete(ctx, key); delErr != nil {
			return delErr
		}
	}

	return nil
}
