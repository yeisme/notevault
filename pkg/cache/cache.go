// Package cache 提供基于键值存储的泛型缓存实现.
//
// 该包提供了类型安全的缓存操作，支持任意类型的缓存值.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/yeisme/notevault/pkg/internal/storage/kv"
)

// Cache 基于KV存储的缓存实现.
// 支持通过 Option 传入清理间隔、默认 TTL 与本地 TTL 包装.
type Cache struct {
	kvStore         kv.KVStore
	cleanupInterval time.Duration
	defaultTTL      time.Duration
	useLocalTTL     bool

	stopCh chan struct{}
}

// Option 配置 Cache 的可选参数.
type Option func(*Cache)

// WithCleanupInterval 设置定期清理间隔；传入 0 则不启动清理协程.
func WithCleanupInterval(d time.Duration) Option {
	return func(c *Cache) { c.cleanupInterval = d }
}

// WithDefaultTTL 设置默认的 TTL（保留以便将来扩展）.
func WithDefaultTTL(d time.Duration) Option {
	return func(c *Cache) { c.defaultTTL = d }
}

// WithLocalTTL 启用本地 TTL 包装.
func WithLocalTTL(enable bool) Option {
	return func(c *Cache) { c.useLocalTTL = enable }
}

// NewCache 创建一个新的缓存实例，并可通过 Option 配置行为.
func NewCache(kvStore kv.KVStore, opts ...Option) *Cache {
	c := &Cache{
		kvStore:         kvStore,
		cleanupInterval: 0,
		defaultTTL:      0,
		useLocalTTL:     false,
		stopCh:          make(chan struct{}),
	}

	for _, o := range opts {
		o(c)
	}

	if c.cleanupInterval > 0 {
		go c.startCleanup()
	}

	return c
}

// Get 泛型获取缓存值.如果启用了本地 TTL，会解析包装并检查过期时间.
func Get[T any](ctx context.Context, c *Cache, key string) (T, error) {
	var zero T

	data, err := c.kvStore.Get(ctx, key)
	if err != nil {
		return zero, err
	}

	if c != nil && c.useLocalTTL {
		var item struct {
			Payload   json.RawMessage `json:"p"`
			ExpiresAt int64           `json:"e,omitempty"`
		}
		if err := sonic.Unmarshal(data, &item); err != nil {
			return zero, fmt.Errorf("failed to unmarshal cache wrapper: %w", err)
		}

		if item.ExpiresAt > 0 && time.Now().UnixNano() > item.ExpiresAt {
			_ = c.kvStore.Delete(ctx, key)
			return zero, fmt.Errorf("cache: miss or expired")
		}

		var value T
		if err := sonic.Unmarshal(item.Payload, &value); err != nil {
			return zero, fmt.Errorf("failed to unmarshal cache payload: %w", err)
		}

		return value, nil
	}

	var value T
	if err := sonic.Unmarshal(data, &value); err != nil {
		return zero, fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return value, nil
}

// Set 泛型设置缓存值.启用本地 TTL 时会把数据包装成 {p: payload, e: expiresAt}.
func Set[T any](ctx context.Context, c *Cache, key string, value T, ttl time.Duration) error {
	if c != nil && c.useLocalTTL {
		payload, err := sonic.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache payload: %w", err)
		}

		var expiresAt int64
		if ttl > 0 {
			expiresAt = time.Now().Add(ttl).UnixNano()
		}

		item := struct {
			Payload   json.RawMessage `json:"p"`
			ExpiresAt int64           `json:"e,omitempty"`
		}{Payload: payload, ExpiresAt: expiresAt}

		data, err := sonic.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal cache wrapper: %w", err)
		}

		return c.kvStore.Set(ctx, key, data, ttl)
	}

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

// Close 停止清理并关闭底层 KV（如果支持）.
func (c *Cache) Close() error {
	if c == nil {
		return nil
	}

	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
	}

	return c.kvStore.Close()
}

// startCleanup 启动周期性清理任务.
func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), c.cleanupInterval)
			c.runCleanupOnce(ctx)
			cancel()
		case <-c.stopCh:
			return
		}
	}
}

// runCleanupOnce 执行一次清理操作：遍历键并删除本地判断为过期的项.
func (c *Cache) runCleanupOnce(ctx context.Context) {
	if c == nil || !c.useLocalTTL {
		return
	}

	keys, err := c.kvStore.Keys(ctx, "*")
	if err != nil {
		return
	}

	for _, key := range keys {
		data, err := c.kvStore.Get(ctx, key)
		if err != nil {
			continue
		}

		var item struct {
			ExpiresAt int64 `json:"e,omitempty"`
		}
		if err := sonic.Unmarshal(data, &item); err != nil {
			continue
		}

		if item.ExpiresAt > 0 && time.Now().UnixNano() > item.ExpiresAt {
			_ = c.kvStore.Delete(ctx, key)
		}
	}
}
