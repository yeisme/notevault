package configs

import "github.com/spf13/viper"

const (
	// 默认速率限制配置.
	DefaultRateLimitEnabled = false
	DefaultRateLimitRPS     = 50.0
	DefaultRateLimitBurst   = 100
	DefaultRateLimitKey     = "ip"
)

// RateLimitConfig 速率限制配置.
type RateLimitConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	RPS     float64 `mapstructure:"rps"`   // 每秒允许的请求数
	Burst   int     `mapstructure:"burst"` // 突发容量
	// Key 选择限流维度：global（全局）、ip（按客户端IP）、header:Header-Name（按请求头）
	Key string `mapstructure:"key"`
}

func (c *RateLimitConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("rate_limit.enabled", DefaultRateLimitEnabled)
	v.SetDefault("rate_limit.rps", DefaultRateLimitRPS)
	v.SetDefault("rate_limit.burst", DefaultRateLimitBurst)
	v.SetDefault("rate_limit.key", DefaultRateLimitKey)
}
