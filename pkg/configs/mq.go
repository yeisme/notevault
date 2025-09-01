package configs

import (
	"github.com/spf13/viper"
)

// MQType 消息队列类型.
type MQType string

const (
	MQTypeNATS MQType = "nats"

	// 默认MQ连接配置.
	DefaultMaxReconnects = 5 //nolint:mnd // 默认最大重连次数.
	DefaultReconnectWait = 5 //nolint:mnd // 默认重连等待时间（秒）.
)

// MQConfig 消息队列配置.
type MQConfig struct {
	Type          MQType `mapstructure:"type"`
	URL           string `mapstructure:"url"`
	User          string `mapstructure:"user"`
	Password      string `mapstructure:"password"`
	ClusterID     string `mapstructure:"cluster_id"`
	ClientID      string `mapstructure:"client_id"`
	MaxReconnects int    `mapstructure:"max_reconnects"`
	ReconnectWait int    `mapstructure:"reconnect_wait"`
}

// GetMQType 返回当前配置的消息队列类型.
func (c *MQConfig) GetMQType() MQType {
	return c.Type
}

// setDefaults 设置MQ配置的默认值.
func (c *MQConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("queue.type", MQTypeNATS)
	v.SetDefault("queue.url", "localhost:4222")
	v.SetDefault("queue.user", "")
	v.SetDefault("queue.password", "")
	v.SetDefault("queue.cluster_id", "notevault-cluster")
	v.SetDefault("queue.client_id", "notevault-app")
	v.SetDefault("queue.max_reconnects", DefaultMaxReconnects)
	v.SetDefault("queue.reconnect_wait", DefaultReconnectWait)
}
