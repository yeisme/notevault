package configs

import (
	"github.com/spf13/viper"
)

// MQType 消息队列类型.
type MQType string

const (
	MQTypeNATS  MQType = "nats"
	MQTypeRedis MQType = "redis"

	DefaultMQURL         = "localhost:4222"
	DefaultMQUser        = ""
	DefaultMQPassword    = ""
	DefaultMaxReconnects = 5                   // 默认最大重连次数.
	DefaultReconnectWait = 5                   // 默认重连等待时间（秒）.
	DefaultMQClusterID   = "notevault-cluster" // 默认集群ID
	DefaultMQClientID    = "notevault-app"     // 默认客户端ID

	// JetStream 流配置常量.

	DefaultStreamMaxMsgs  = 1000000            // 默认流最大消息数
	DefaultStreamMaxBytes = 1024 * 1024 * 1024 // 默认流最大字节数 (1GB)
	DefaultStreamMaxAge   = 24                 // 默认流最大年龄 (小时)
	DefaultStreamReplicas = 1                  // 默认流副本数

	// 消费者配置常量.

	DefaultConsumerAckWait       = 30   // 默认消费者确认等待时间 (秒)
	DefaultConsumerMaxDeliver    = 3    // 默认消费者最大投递次数
	DefaultConsumerMaxAckPending = 1000 // 默认消费者最大待确认消息数

	// 队列配置常量.

	DefaultMaxPingsOut  = 3     // 默认最大ping输出次数
	DefaultPingInterval = 20    // 默认ping间隔 (秒)
	DefaultBufferSize   = 32768 // 默认缓冲区大小 (32KB)
	DefaultConnPoolSize = 10    // 默认连接池大小
)

// MQConfig 消息队列配置.
type MQConfig struct {
	Type   MQType         `mapstructure:"type"   rule:"oneof=nats redis"`
	Common MQCommonConfig `mapstructure:"common"`
	NATS   MQNATSConfig   `mapstructure:"nats"`
	Redis  MQRedisConfig  `mapstructure:"redis"`
}

// MQCommonConfig 通用MQ配置.
type MQCommonConfig struct {
	URL                string `mapstructure:"url"                  rule:"hostname_port"`
	User               string `mapstructure:"user"`
	Password           string `mapstructure:"password"`
	ClusterID          string `mapstructure:"cluster_id"`
	ClientID           string `mapstructure:"client_id"`
	MaxReconnects      int    `mapstructure:"max_reconnects"       rule:"min=0,max=100"`
	ReconnectWait      int    `mapstructure:"reconnect_wait"       rule:"min=1,max=300"`
	StrictConnect      bool   `mapstructure:"strict_connect"`
	MaxPingsOut        int    `mapstructure:"max_pings_out"        rule:"min=1,max=10"`
	PingInterval       int    `mapstructure:"ping_interval"        rule:"min=1,max=300"`
	ReconnectJitter    bool   `mapstructure:"reconnect_jitter"`
	ReconnectJitterTLS bool   `mapstructure:"reconnect_jitter_tls"`
	BufferSize         int    `mapstructure:"buffer_size"          rule:"min=1024,max=1048576"`
	ConnPoolSize       int    `mapstructure:"conn_pool_size"       rule:"min=1,max=100"`
	EnableMetrics      bool   `mapstructure:"enable_metrics"`
	Endpoint           string `mapstructure:"endpoint"`
}

// MQNATSConfig NATS MQ 配置.
type MQNATSConfig struct {
	JetStreamEnabled       bool     `mapstructure:"jetstream_enabled"`
	StreamName             string   `mapstructure:"stream_name"`
	SubjectPrefix          string   `mapstructure:"subject_prefix"`
	JetStreamAutoProvision bool     `mapstructure:"jetstream_auto_provision"`
	JetStreamTrackMsgID    bool     `mapstructure:"jetstream_track_msg_id"`
	JetStreamAckAsync      bool     `mapstructure:"jetstream_ack_async"`
	JetStreamDurablePrefix string   `mapstructure:"jetstream_durable_prefix"`
	StreamMaxMsgs          int64    `mapstructure:"stream_max_msgs"`
	StreamMaxBytes         int64    `mapstructure:"stream_max_bytes"`
	StreamMaxAge           int      `mapstructure:"stream_max_age"`
	StreamStorageType      string   `mapstructure:"stream_storage_type"`
	StreamReplicas         int      `mapstructure:"stream_replicas"`
	ConsumerAckWait        int      `mapstructure:"consumer_ack_wait"`
	ConsumerMaxDeliver     int      `mapstructure:"consumer_max_deliver"`
	ConsumerMaxAckPending  int      `mapstructure:"consumer_max_ack_pending"`
	JWT                    string   `mapstructure:"jwt"`
	NKey                   string   `mapstructure:"nkey"`
	ClusterURLs            []string `mapstructure:"cluster_urls"`
	LoadBalance            bool     `mapstructure:"load_balance"`
}

// MQRedisConfig Redis MQ 配置.
type MQRedisConfig struct {
	Addr     string `mapstructure:"addr"     rule:"hostname_port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"       rule:"min=0,max=15"`
}

// GetMQType 返回当前配置的消息队列类型.
func (c *MQConfig) GetMQType() MQType {
	return c.Type
}

// setDefaults 设置MQ配置的默认值.
func (c *MQConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("mq.type", MQTypeNATS)

	// Common 默认值
	v.SetDefault("mq.common.url", DefaultMQURL)
	v.SetDefault("mq.common.user", DefaultMQUser)
	v.SetDefault("mq.common.password", DefaultMQPassword)
	v.SetDefault("mq.common.cluster_id", DefaultMQClusterID)
	v.SetDefault("mq.common.client_id", DefaultMQClientID)
	v.SetDefault("mq.common.max_reconnects", DefaultMaxReconnects)
	v.SetDefault("mq.common.reconnect_wait", DefaultReconnectWait)
	v.SetDefault("mq.common.strict_connect", false)
	v.SetDefault("mq.common.max_pings_out", DefaultMaxPingsOut)
	v.SetDefault("mq.common.ping_interval", DefaultPingInterval)
	v.SetDefault("mq.common.reconnect_jitter", true)
	v.SetDefault("mq.common.reconnect_jitter_tls", true)
	v.SetDefault("mq.common.buffer_size", DefaultBufferSize)
	v.SetDefault("mq.common.conn_pool_size", DefaultConnPoolSize)
	v.SetDefault("mq.common.enable_metrics", true)
	v.SetDefault("mq.common.endpoint", ":9092")

	// NATS 默认值
	v.SetDefault("mq.nats.jetstream_enabled", true)
	v.SetDefault("mq.nats.stream_name", "notevault-stream")
	v.SetDefault("mq.nats.subject_prefix", "notevault.")
	v.SetDefault("mq.nats.jetstream_auto_provision", true)
	v.SetDefault("mq.nats.jetstream_track_msg_id", true)
	v.SetDefault("mq.nats.jetstream_ack_async", true)
	v.SetDefault("mq.nats.jetstream_durable_prefix", "notevault-durable")
	v.SetDefault("mq.nats.stream_max_msgs", DefaultStreamMaxMsgs)
	v.SetDefault("mq.nats.stream_max_bytes", DefaultStreamMaxBytes)
	v.SetDefault("mq.nats.stream_max_age", DefaultStreamMaxAge)
	v.SetDefault("mq.nats.stream_storage_type", "file")
	v.SetDefault("mq.nats.stream_replicas", DefaultStreamReplicas)
	v.SetDefault("mq.nats.consumer_ack_wait", DefaultConsumerAckWait)
	v.SetDefault("mq.nats.consumer_max_deliver", DefaultConsumerMaxDeliver)
	v.SetDefault("mq.nats.consumer_max_ack_pending", DefaultConsumerMaxAckPending)
	v.SetDefault("mq.nats.jwt", "")
	v.SetDefault("mq.nats.nkey", "")
	v.SetDefault("mq.nats.cluster_urls", []string{})
	v.SetDefault("mq.nats.load_balance", true)

	// Redis 默认值
	v.SetDefault("mq.redis.addr", "localhost:6379")
	v.SetDefault("mq.redis.password", "")
	v.SetDefault("mq.redis.db", 0)
}
