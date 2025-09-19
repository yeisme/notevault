package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/yeisme/notevault/pkg/configs"
)

// NATSKV 基于 NATS KV 的 KV 实现.
type NATSKV struct {
	js     nats.JetStreamContext
	kv     nats.KeyValue
	bucket string
	conn   *nats.Conn
}

// NewNATSKV 创建 NATS KV 实例.
func NewNATSKV(ctx context.Context, config any) (KVStore, error) {
	natsConfig, ok := config.(*configs.NATSKVConfig)
	if !ok {
		return nil, fmt.Errorf("invalid NATS config")
	}

	// 连接到 NATS
	opts := []nats.Option{}
	if natsConfig.User != "" {
		opts = append(opts, nats.UserInfo(natsConfig.User, natsConfig.Password))
	}

	nc, err := nats.Connect(natsConfig.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// 创建 JetStream 上下文
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// 创建或获取 KV bucket（bucket 级 TTL 如果配置层未来需要，可在这里增加）
	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: natsConfig.Bucket,
		// TTL: natsConfig.TTL,
	})
	if err != nil {
		// 如果 bucket 已存在，获取它
		kv, err = js.KeyValue(natsConfig.Bucket)
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("failed to create/get KV bucket: %w", err)
		}
	}

	return &NATSKV{
		js:     js,
		kv:     kv,
		bucket: natsConfig.Bucket,
		conn:   nc,
	}, nil
}

// Get 获取键的值.
func (n *NATSKV) Get(ctx context.Context, key string) ([]byte, error) {
	entry, err := n.kv.Get(key)
	if err == nats.ErrKeyNotFound {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	now := time.Now()

	val, expired, _, derr := decodeWithTTL(entry.Value(), now)
	if derr != nil {
		return nil, derr
	}

	if expired {
		// lazy delete expired entry
		_ = n.kv.Delete(key)
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return val, nil
}

// Set 设置键的值.
func (n *NATSKV) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	encoded, wrapped, err := encodeWithTTL(value, ttl)
	if err != nil {
		return err
	}

	_ = wrapped // reserved for future conditional usage

	_, err = n.kv.Put(key, encoded)
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}

	return nil
}

// Delete 删除键.
func (n *NATSKV) Delete(ctx context.Context, key string) error {
	err := n.kv.Delete(key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// Exists 检查键是否存在.
func (n *NATSKV) Exists(ctx context.Context, key string) (bool, error) {
	entry, err := n.kv.Get(key)
	if err == nats.ErrKeyNotFound {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	_, expired, _, derr := decodeWithTTL(entry.Value(), time.Now())
	if derr != nil {
		return false, derr
	}

	if expired {
		_ = n.kv.Delete(key)
		return false, nil
	}

	return true, nil
}

// Keys 获取所有键.
func (n *NATSKV) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := n.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	// 简单过滤，如果需要复杂模式匹配可以使用第三方库
	result := make([]string, 0)

	for _, key := range keys {
		if pattern != "" && key != pattern {
			continue
		}
		// check ttl lazily
		if entry, e := n.kv.Get(key); e == nil {
			if _, expired, _, derr := decodeWithTTL(entry.Value(), time.Now()); derr == nil {
				if expired {
					_ = n.kv.Delete(key)
					continue
				}
			}
		}

		result = append(result, key)
	}

	return result, nil
}

// Close 关闭 NATS 连接.
func (n *NATSKV) Close() error {
	n.conn.Close()
	return nil
}

func init() {
	RegisterKVFactory(KVTypeNATS, NewNATSKV)
}
