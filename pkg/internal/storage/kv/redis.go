//go:build !no_redis

package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/yeisme/notevault/pkg/configs"
)

// RedisKV 基于 Redis 的 KV 实现.
type RedisKV struct {
	client *redis.Client
}

// NewRedisKV 创建 Redis KV 实例.
func NewRedisKV(ctx context.Context, config any) (KVStore, error) {
	redisConfig, ok := config.(*configs.RedisKVConfig)
	if !ok {
		return nil, fmt.Errorf("invalid Redis config")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisKV{
		client: rdb,
	}, nil
}

// Get 获取键的值.
func (r *RedisKV) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	return []byte(result), nil
}

// Set 设置键的值.
func (r *RedisKV) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}

	return nil
}

// Delete 删除键.
func (r *RedisKV) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// Exists 检查键是否存在.
func (r *RedisKV) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return count > 0, nil
}

// Keys 获取匹配模式的键.
func (r *RedisKV) Keys(ctx context.Context, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	return keys, nil
}

// Close 关闭 Redis 连接.
func (r *RedisKV) Close() error {
	return r.client.Close()
}

func init() {
	RegisterKVFactory(KVTypeRedis, NewRedisKV)
}
