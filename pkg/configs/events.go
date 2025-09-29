package configs

import "github.com/spf13/viper"

// EventsConfig 控制事件发布的开关（全局与分主题）。
type EventsConfig struct {
	Enabled bool               `mapstructure:"enabled"` // 总开关
	Object  ObjectEventsConfig `mapstructure:"object"`
}

// ObjectEventsConfig 针对对象存储领域的事件开关。
type ObjectEventsConfig struct {
	Stored      bool `mapstructure:"stored"`
	Updated     bool `mapstructure:"updated"`
	Deleted     bool `mapstructure:"deleted"`
	Versioned   bool `mapstructure:"versioned"`
	Restored    bool `mapstructure:"restored"`
	Moved       bool `mapstructure:"moved"`
	Accessed    bool `mapstructure:"accessed"`
	StorageFull bool `mapstructure:"storage_full"`
}

func (c *EventsConfig) setDefaults(v *viper.Viper) {
	// 总开关：默认启用事件系统
	v.SetDefault("events.enabled", true)

	// 对象领域的事件：默认仅开启最小必要集，避免噪声过大
	v.SetDefault("events.object.stored", true)
	v.SetDefault("events.object.deleted", true)

	// 可选事件：默认关闭，按需开启
	v.SetDefault("events.object.updated", false)
	v.SetDefault("events.object.versioned", false)
	v.SetDefault("events.object.restored", false)
	v.SetDefault("events.object.moved", false)
	v.SetDefault("events.object.accessed", false)     // 访问事件量可能很大，默认关闭
	v.SetDefault("events.object.storage_full", false) // 告警事件，建议由监控触发后再开启
}
