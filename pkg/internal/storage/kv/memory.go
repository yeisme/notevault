package kv

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryKV 基于 sync.Map 的内存 KV 实现.
type MemoryKV struct {
	data sync.Map // 并发安全的 map
}

// NewMemoryKV 创建内存 KV 实例.
func NewMemoryKV(ctx context.Context, config any) (KVStore, error) {
	// 内存实现不需要特殊配置
	return &MemoryKV{}, nil
}

// Get 获取键的值.
func (m *MemoryKV) Get(ctx context.Context, key string) ([]byte, error) {
	value, exists := m.data.Load(key)
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	data, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid value type for key: %s", key)
	}

	// 返回副本
	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}

// Set 设置键的值.
func (m *MemoryKV) Set(ctx context.Context, key string, value []byte, _ time.Duration) error {
	// 复制值
	data := make([]byte, len(value))
	copy(data, value)

	m.data.Store(key, data)

	return nil
}

// Delete 删除键.
func (m *MemoryKV) Delete(ctx context.Context, key string) error {
	m.data.Delete(key)
	return nil
}

// Exists 检查键是否存在.
func (m *MemoryKV) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.data.Load(key)
	return exists, nil
}

// Keys 获取所有键.
func (m *MemoryKV) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys := make([]string, 0)

	m.data.Range(func(key, value any) bool {
		k, ok := key.(string)
		if !ok {
			return true // 继续遍历
		}

		if pattern == "" || k == pattern {
			keys = append(keys, k)
		}

		return true
	})

	return keys, nil
}

// Close 关闭存储（内存实现无需操作）.
func (m *MemoryKV) Close() error {
	return nil
}

func init() {
	RegisterKVFactory(KVTypeMemory, NewMemoryKV)
}
