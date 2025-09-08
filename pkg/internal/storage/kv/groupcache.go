package kv

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/groupcache"

	"github.com/yeisme/notevault/pkg/configs"
)

// GroupcacheKV 基于 Groupcache 的 KV 实现.
type GroupcacheKV struct {
	cache  *groupcache.Group    // Groupcache 缓存组
	peers  *groupcache.HTTPPool // 对等节点池
	getter groupcache.Getter    // 获取器
	data   map[string][]byte    // 本地存储数据
	mu     sync.RWMutex         // 保护 data 的读写锁
}

// groupcacheGetter 实现 groupcache.Getter 接口.
type groupcacheGetter struct {
	kv *GroupcacheKV
}

func (g *groupcacheGetter) Get(ctx context.Context, key string, dest groupcache.Sink) error {
	g.kv.mu.RLock()
	value, exists := g.kv.data[key]
	g.kv.mu.RUnlock()

	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	if err := dest.SetBytes(value); err != nil {
		return fmt.Errorf("failed to set bytes to sink: %w", err)
	}

	return nil
}

// NewGroupcacheKV 创建 Groupcache KV 实例.
func NewGroupcacheKV(ctx context.Context, config any) (KVStore, error) {
	gcConfig, ok := config.(*configs.GroupcacheKVConfig)
	if !ok {
		return nil, fmt.Errorf("invalid Groupcache config")
	}

	kv := &GroupcacheKV{
		data: make(map[string][]byte),
	}

	// 创建 getter
	kv.getter = &groupcacheGetter{kv: kv}

	// 创建缓存组
	kv.cache = groupcache.NewGroup(gcConfig.Name, gcConfig.CacheBytes, kv.getter)

	// 如果有对等节点，设置 HTTP 池
	if len(gcConfig.Peers) > 0 {
		kv.peers = groupcache.NewHTTPPoolOpts(gcConfig.Self, &groupcache.HTTPPoolOptions{})
		kv.peers.Set(gcConfig.Peers...)
	}

	return kv, nil
}

// Get 获取键的值.
func (g *GroupcacheKV) Get(ctx context.Context, key string) ([]byte, error) {
	var data []byte

	err := g.cache.Get(ctx, key, groupcache.AllocatingByteSliceSink(&data))
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	// 返回副本
	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}

// Set 设置键的值.
func (g *GroupcacheKV) Set(ctx context.Context, key string, value []byte, _ time.Duration) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 复制值
	g.data[key] = make([]byte, len(value))
	copy(g.data[key], value)

	return nil
}

// Delete 删除键.
func (g *GroupcacheKV) Delete(ctx context.Context, key string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.data, key)

	return nil
}

// Exists 检查键是否存在.
func (g *GroupcacheKV) Exists(ctx context.Context, key string) (bool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	_, exists := g.data[key]

	return exists, nil
}

// Keys 获取所有键.
func (g *GroupcacheKV) Keys(ctx context.Context, pattern string) ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	keys := make([]string, 0, len(g.data))
	for key := range g.data {
		if pattern == "" || key == pattern {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Close 关闭缓存.
func (g *GroupcacheKV) Close() error {
	// Groupcache 没有显式的关闭方法
	return nil
}

func init() {
	RegisterKVFactory(KVTypeGroupcache, NewGroupcacheKV)
}
