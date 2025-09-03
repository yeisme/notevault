package configs

import (
	"github.com/spf13/viper"
)

// MQType 消息队列类型.
type MQType string

const (
	MQTypeNATS MQType = "nats"

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
	Type          MQType `mapstructure:"type"           rule:"oneof=nats"`
	URL           string `mapstructure:"url"            rule:"hostname_port"`
	User          string `mapstructure:"user"`
	Password      string `mapstructure:"password"`
	ClusterID     string `mapstructure:"cluster_id"`
	ClientID      string `mapstructure:"client_id"`
	MaxReconnects int    `mapstructure:"max_reconnects" rule:"min=0,max=100"`
	ReconnectWait int    `mapstructure:"reconnect_wait" rule:"min=1,max=300"`
	// 高性能和高优化配置
	MaxPingsOut        int  `mapstructure:"max_pings_out"        rule:"min=1,max=10"`
	PingInterval       int  `mapstructure:"ping_interval"        rule:"min=1,max=300"`
	ReconnectJitter    bool `mapstructure:"reconnect_jitter"`
	ReconnectJitterTLS bool `mapstructure:"reconnect_jitter_tls"`
	// 缓冲区和连接池配置
	BufferSize   int `mapstructure:"buffer_size"    rule:"min=1024,max=1048576"`
	ConnPoolSize int `mapstructure:"conn_pool_size" rule:"min=1,max=100"`
	// JetStream 配置
	JetStreamEnabled bool   `mapstructure:"jetstream_enabled"`
	StreamName       string `mapstructure:"stream_name"`
	SubjectPrefix    string `mapstructure:"subject_prefix"`
	// JetStream 高级配置
	JetStreamAutoProvision bool   `mapstructure:"jetstream_auto_provision"`
	JetStreamTrackMsgID    bool   `mapstructure:"jetstream_track_msg_id"`
	JetStreamAckAsync      bool   `mapstructure:"jetstream_ack_async"`
	JetStreamDurablePrefix string `mapstructure:"jetstream_durable_prefix"`
	// 流配置
	StreamMaxMsgs     int64  `mapstructure:"stream_max_msgs"`
	StreamMaxBytes    int64  `mapstructure:"stream_max_bytes"`
	StreamMaxAge      int    `mapstructure:"stream_max_age"`      // 小时
	StreamStorageType string `mapstructure:"stream_storage_type"` // file, memory
	StreamReplicas    int    `mapstructure:"stream_replicas"`
	// 消费者配置
	ConsumerAckWait       int `mapstructure:"consumer_ack_wait"` // 秒
	ConsumerMaxDeliver    int `mapstructure:"consumer_max_deliver"`
	ConsumerMaxAckPending int `mapstructure:"consumer_max_ack_pending"`
	// 认证配置
	JWT  string `mapstructure:"jwt"`
	NKey string `mapstructure:"nkey"`
	// 集群和扩展配置
	ClusterURLs []string `mapstructure:"cluster_urls"`
	LoadBalance bool     `mapstructure:"load_balance"`
	// 监控和指标
	EnableMetrics bool `mapstructure:"enable_metrics"`
}

// GetMQType 返回当前配置的消息队列类型.
func (c *MQConfig) GetMQType() MQType {
	return c.Type
}

// setDefaults 设置MQ配置的默认值.
func (c *MQConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("mq.type", MQTypeNATS)
	v.SetDefault("mq.url", DefaultMQURL)
	v.SetDefault("mq.user", DefaultMQUser)
	v.SetDefault("mq.password", DefaultMQPassword)
	v.SetDefault("mq.cluster_id", DefaultMQClusterID)
	v.SetDefault("mq.client_id", DefaultMQClientID)
	v.SetDefault("mq.max_reconnects", DefaultMaxReconnects)
	v.SetDefault("mq.reconnect_wait", DefaultReconnectWait)

	// 高性能和高优化默认值
	v.SetDefault("mq.max_pings_out", DefaultMaxPingsOut)
	v.SetDefault("mq.ping_interval", DefaultPingInterval)
	v.SetDefault("mq.reconnect_jitter", true)
	v.SetDefault("mq.reconnect_jitter_tls", true)
	v.SetDefault("mq.buffer_size", DefaultBufferSize)
	v.SetDefault("mq.conn_pool_size", DefaultConnPoolSize)

	// JetStream 默认值
	v.SetDefault("mq.jetstream_enabled", true)
	v.SetDefault("mq.stream_name", "notevault-stream")
	v.SetDefault("mq.subject_prefix", "notevault.")

	// JetStream 高级默认值
	v.SetDefault("mq.jetstream_auto_provision", true)
	v.SetDefault("mq.jetstream_track_msg_id", true)
	v.SetDefault("mq.jetstream_ack_async", true)
	v.SetDefault("mq.jetstream_durable_prefix", "notevault-durable")

	// 流默认值
	v.SetDefault("mq.stream_max_msgs", DefaultStreamMaxMsgs)
	v.SetDefault("mq.stream_max_bytes", DefaultStreamMaxBytes)
	v.SetDefault("mq.stream_max_age", DefaultStreamMaxAge)
	v.SetDefault("mq.stream_storage_type", "file")
	v.SetDefault("mq.stream_replicas", DefaultStreamReplicas)

	// 消费者默认值
	v.SetDefault("mq.consumer_ack_wait", DefaultConsumerAckWait)
	v.SetDefault("mq.consumer_max_deliver", DefaultConsumerMaxDeliver)
	v.SetDefault("mq.consumer_max_ack_pending", DefaultConsumerMaxAckPending)

	// 认证默认值
	v.SetDefault("mq.jwt", "")
	v.SetDefault("mq.nkey", "")

	// 集群和扩展默认值
	v.SetDefault("mq.cluster_urls", []string{})
	v.SetDefault("mq.load_balance", true)

	// 监控和指标默认值
	v.SetDefault("mq.enable_metrics", false)
}
