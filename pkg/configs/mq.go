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
}

// GetMQType 返回当前配置的消息队列类型.
func (c *MQConfig) GetMQType() MQType {
	return c.Type
}

// setDefaults 设置MQ配置的默认值.
func (c *MQConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("queue.type", MQTypeNATS)
	v.SetDefault("queue.url", DefaultMQURL)
	v.SetDefault("queue.user", DefaultMQUser)
	v.SetDefault("queue.password", DefaultMQPassword)
	v.SetDefault("queue.cluster_id", DefaultMQClusterID)
	v.SetDefault("queue.client_id", DefaultMQClientID)
	v.SetDefault("queue.max_reconnects", DefaultMaxReconnects)
	v.SetDefault("queue.reconnect_wait", DefaultReconnectWait)
}
