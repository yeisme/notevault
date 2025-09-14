package configs

import "github.com/spf13/viper"

const (
	// 默认熔断器配置.
	DefaultCBEnabled           = false
	DefaultCBFailureRate       = 0.5
	DefaultCBMinRequests       = 20
	DefaultCBIntervalSeconds   = 60
	DefaultCBTimeoutSeconds    = 30
	DefaultCBMaxRequestsInHalf = 5
)

// CircuitBreakerConfig 熔断器配置.
type CircuitBreakerConfig struct {
	Enabled           bool    `mapstructure:"enabled"`
	FailureRate       float64 `mapstructure:"failure_rate"`         // 连续窗口失败比例阈值 [0,1]
	MinRequests       uint32  `mapstructure:"min_requests"`         // 进入统计的最小请求数
	IntervalSeconds   int     `mapstructure:"interval_seconds"`     // 滑动窗口统计周期
	TimeoutSeconds    int     `mapstructure:"timeout_seconds"`      // 打开状态持续时间（自动半开）
	MaxRequestsInHalf uint32  `mapstructure:"max_requests_in_half"` // 半开状态允许的并发请求数
}

func (c *CircuitBreakerConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("circuit_breaker.enabled", DefaultCBEnabled)
	v.SetDefault("circuit_breaker.failure_rate", DefaultCBFailureRate)
	v.SetDefault("circuit_breaker.min_requests", DefaultCBMinRequests)
	v.SetDefault("circuit_breaker.interval_seconds", DefaultCBIntervalSeconds)
	v.SetDefault("circuit_breaker.timeout_seconds", DefaultCBTimeoutSeconds)
	v.SetDefault("circuit_breaker.max_requests_in_half", DefaultCBMaxRequestsInHalf)
}
