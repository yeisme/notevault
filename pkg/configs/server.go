package configs

import (
	"time"

	"github.com/spf13/viper"
)

const (
	DefaultPort         = 8080      // 监听端口
	DefaultHost         = "0.0.0.0" // 监听地址
	DefaultReloadConfig = true      // 是否启用配置热重载
	DefaultDebug        = false     // 是否启用调试模式
	DefaultTimeout      = 30        // 超时时间，单位秒
)

type (
	// ServerConfig 服务器配置.
	ServerConfig struct {
		Port         int    `mapstructure:"port"          rule:"min=1,max=65535"`
		Host         string `mapstructure:"host"          rule:"ip"`
		ReloadConfig bool   `mapstructure:"reload_config"`
		Debug        bool   `mapstructure:"debug"`
		Timeout      int    `mapstructure:"timeout"       rule:"min=1,max=300"`
	}
)

// GetTimeoutDuration 返回超时时间作为time.Duration.
func (s *ServerConfig) GetTimeoutDuration() time.Duration {
	return time.Duration(s.Timeout) * time.Second
}

// setDefaults 设置服务器配置的默认值.
func (s *ServerConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", DefaultPort)
	v.SetDefault("server.host", DefaultHost)
	v.SetDefault("server.log_level", DefaultLogLevel)
	v.SetDefault("server.reload_config", DefaultReloadConfig)
	v.SetDefault("server.debug", DefaultDebug)
	v.SetDefault("server.timeout", DefaultTimeout)
}
