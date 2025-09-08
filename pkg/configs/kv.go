package configs

import (
	"github.com/spf13/viper"
)

// KVConfig 键值存储配置.
type KVConfig struct {
	Type       string             `mapstructure:"type"       rule:"oneof=memory,redis,nats,groupcache"`
	Redis      RedisKVConfig      `mapstructure:"redis"`
	NATS       NATSKVConfig       `mapstructure:"nats"`
	Groupcache GroupcacheKVConfig `mapstructure:"groupcache"`
}

// RedisKVConfig Redis KV 配置.
type RedisKVConfig struct {
	Addr     string `mapstructure:"addr"     rule:"hostname_port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"       rule:"min=0,max=15"`
}

// NATSKVConfig NATS KV 配置.
type NATSKVConfig struct {
	URL      string `mapstructure:"url"      rule:"hostname_port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Bucket   string `mapstructure:"bucket"   rule:"required"`
}

// GroupcacheKVConfig Groupcache KV 配置.
type GroupcacheKVConfig struct {
	Name       string   `mapstructure:"name"        rule:"required"`
	CacheBytes int64    `mapstructure:"cache_bytes" rule:"min=1048576"` // 最小1MB
	Peers      []string `mapstructure:"peers"`
	Self       string   `mapstructure:"self"        rule:"hostname_port"`
}

// GetKVType 返回当前配置的 KV 类型.
func (c *KVConfig) GetKVType() string {
	return c.Type
}

// setDefaults 设置 KV 配置的默认值.
func (c *KVConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("kv.type", "groupcache")

	// Redis 默认值
	v.SetDefault("kv.redis.addr", "localhost:6379")
	v.SetDefault("kv.redis.password", "")
	v.SetDefault("kv.redis.db", 0)

	// NATS 默认值
	v.SetDefault("kv.nats.url", "localhost:4222")
	v.SetDefault("kv.nats.user", "")
	v.SetDefault("kv.nats.password", "")
	v.SetDefault("kv.nats.bucket", "notevault-kv")

	const maxGroupcacheCacheBytes = 512 * 1024 * 1024 // 512MB
	// Groupcache 默认值
	v.SetDefault("kv.groupcache.name", "notevault-cache")
	v.SetDefault("kv.groupcache.cache_bytes", maxGroupcacheCacheBytes)
	v.SetDefault("kv.groupcache.peers", []string{})
	v.SetDefault("kv.groupcache.self", "http://localhost:8080")
}
