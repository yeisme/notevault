package configs

import (
	"time"

	"github.com/spf13/viper"
)

const (
	DefaultPort         = 8080      // 监听端口
	DefaultHost         = "0.0.0.0" // 监听地址
	DefaultLogLevel     = "info"    // 日志级别
	DefaultReloadConfig = true      // 是否启用配置热重载
	DefaultDebug        = false     // 是否启用调试模式
	DefaultTimeout      = 30        // 超时时间，单位秒
)

type (
	// ServerConfig 服务器配置.
	ServerConfig struct {
		Port         int    `mapstructure:"port"`
		Host         string `mapstructure:"host"`
		LogLevel     string `mapstructure:"log_level"`
		ReloadConfig bool   `mapstructure:"reload_config"`
		Debug        bool   `mapstructure:"debug"`
		Timeout      int    `mapstructure:"timeout"`
	}
)

// GetTimeoutDuration 返回超时时间作为time.Duration.
func (s *ServerConfig) GetTimeoutDuration() time.Duration {
	return time.Duration(s.Timeout) * time.Second
}

// setDefaults 设置服务器配置的默认值.
func (s *ServerConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("port", DefaultPort)
	v.SetDefault("host", DefaultHost)
	v.SetDefault("log_level", DefaultLogLevel)
	v.SetDefault("reload_config", DefaultReloadConfig)
	v.SetDefault("debug", DefaultDebug)
	v.SetDefault("timeout", DefaultTimeout)
}
