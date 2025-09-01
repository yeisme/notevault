package configs

import (
	"time"

	"github.com/spf13/viper"
)

type (
	// ServerConfig 服务器配置
	ServerConfig struct {
		Port         int    `mapstructure:"port"`
		Host         string `mapstructure:"host"`
		LogLevel     string `mapstructure:"log_level"`
		ReloadConfig bool   `mapstructure:"reload_config"`
		Debug        bool   `mapstructure:"debug"`
		Timeout      int    `mapstructure:"timeout"`
	}
)

// setDefaults 设置服务器配置的默认值
func (s *ServerConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("port", 8080)
	v.SetDefault("host", "0.0.0.0")
	v.SetDefault("log_level", "info")
	v.SetDefault("reload_config", true)
	v.SetDefault("debug", false)
	v.SetDefault("timeout", 30)
}

// GetTimeoutDuration 返回超时时间作为time.Duration
func (s *ServerConfig) GetTimeoutDuration() time.Duration {
	return time.Duration(s.Timeout) * time.Second
}
