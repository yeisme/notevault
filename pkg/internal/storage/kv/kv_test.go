package kv_test

import (
	"context"
	crand "crypto/rand"
	"fmt"
	mrand "math/rand"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yeisme/notevault/pkg/configs"
	"github.com/yeisme/notevault/pkg/internal/storage/kv"
)

func BenchmarkMemoryKV(b *testing.B) {
	store, err := kv.NewKVStore(context.Background(), kv.KVTypeMemory, nil)
	if err != nil {
		b.Fatalf("create memory kv: %v", err)
	}

	benchKV(b, "memory", store)
	benchKVParallel(b, "memory", store)
	_ = store.Close()
}

func BenchmarkGroupcacheKV(b *testing.B) {
	cfg := &configs.GroupcacheKVConfig{
		Name:       "bench-groupcache",
		CacheBytes: 32 * 1024 * 1024, // 32MB
		Peers:      []string{},
		Self:       "http://127.0.0.1:0",
	}

	store, err := kv.NewKVStore(context.Background(), kv.KVTypeGroupcache, cfg)
	if err != nil {
		b.Fatalf("create groupcache kv: %v", err)
	}

	benchKV(b, "groupcache", store)
	benchKVParallel(b, "groupcache", store)
	_ = store.Close()
}

// Optional: enable with ENABLE_REDIS_BENCH=1 and REDIS_ADDR set (default 127.0.0.1:6379).
func BenchmarkRedisKV(b *testing.B) {
	if os.Getenv("ENABLE_REDIS_BENCH") == "" {
		b.Skip("set ENABLE_REDIS_BENCH=1 to enable")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	cfg := &configs.RedisKVConfig{Addr: addr, Password: "", DB: 0}

	store, err := kv.NewKVStore(context.Background(), kv.KVTypeRedis, cfg)
	if err != nil {
		b.Skipf("redis not available: %v", err)
		return
	}

	benchKV(b, "redis", store)
	benchKVParallel(b, "redis", store)
	_ = store.Close()
}

// Optional: enable with ENABLE_NATS_BENCH=1 and NATS_URL set (default nats://127.0.0.1:4222)
func BenchmarkNATSKV(b *testing.B) {
	if os.Getenv("ENABLE_NATS_BENCH") == "" {
		b.Skip("set ENABLE_NATS_BENCH=1 to enable")
	}

	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://127.0.0.1:4222"
	}

	bucket := os.Getenv("NATS_BUCKET")
	if bucket == "" {
		bucket = "bench-kv"
	}

	cfg := &configs.NATSKVConfig{URL: url, User: "", Password: "", Bucket: bucket}

	store, err := kv.NewKVStore(context.Background(), kv.KVTypeNATS, cfg)
	if err != nil {
		b.Skipf("nats not available: %v", err)
		return
	}

	benchKV(b, "nats", store)
	benchKVParallel(b, "nats", store)
	_ = store.Close()
}

// randBytes returns n random bytes, seeded reproducibly for bench.
func randBytes(n int) []byte {
	b := make([]byte, n)
	// Try crypto/rand; if it fails (unlikely in tests), fallback to deterministic PRNG.
	if _, err := crand.Read(b); err != nil {
		mr := mrand.New(mrand.NewSource(42))
		for i := range b {
			b[i] = byte(mr.Intn(256))
		}
	}

	return b
}

// benchKV 执行基本的 Set/Get/Delete 基准测试.
func benchKV(b *testing.B, name string, store kv.KVStore) {
	ctx := context.Background()
	sizes := []int{32, 1024, 64 * 1024}
	ttls := []time.Duration{0, 5 * time.Second}

	for _, size := range sizes {
		payload := randBytes(size)
		for _, ttl := range ttls {
			b.Run(fmt.Sprintf("%s/size=%d/ttl=%s", name, size, ttl), func(b *testing.B) {
				// ensure clean
				b.ReportAllocs()

				for i := 0; b.Loop(); i++ {
					// Use hyphens to ensure keys are valid for NATS KV
					key := fmt.Sprintf("bench-%s-%d", name, i)
					if err := store.Set(ctx, key, payload, ttl); err != nil {
						b.Fatalf("set failed: %v", err)
					}

					if _, err := store.Get(ctx, key); err != nil {
						b.Fatalf("get failed: %v", err)
					}

					if err := store.Delete(ctx, key); err != nil {
						b.Fatalf("delete failed: %v", err)
					}
				}
			})
		}
	}
}

// benchKVParallel 执行并行的 Set/Get/Delete 基准测试.
func benchKVParallel(b *testing.B, name string, store kv.KVStore) {
	ctx := context.Background()
	size := 1024
	payload := randBytes(size)

	var ctr uint64

	b.Run(fmt.Sprintf("%s/parallel", name), func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				i := atomic.AddUint64(&ctr, 1)

				// Use hyphens to ensure keys are valid for NATS KV
				key := fmt.Sprintf("bench-%s-p-%d", name, i)
				if err := store.Set(ctx, key, payload, 0); err != nil {
					b.Fatalf("set failed: %v", err)
				}

				if _, err := store.Get(ctx, key); err != nil {
					b.Fatalf("get failed: %v", err)
				}

				if err := store.Delete(ctx, key); err != nil {
					b.Fatalf("delete failed: %v", err)
				}
			}
		})
	})
}
