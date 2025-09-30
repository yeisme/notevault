package configs

import "github.com/spf13/viper"

// AuthConfig 控制统一身份认证（优先支持 oauth2-proxy 注入的请求头）。
type AuthConfig struct {
	Enabled       bool     `mapstructure:"enabled"`         // 开启认证校验
	SkipPaths     []string `mapstructure:"skip_paths"`      // 跳过认证的路径前缀（如 /metrics、/api/v1/health）
	DevAllowQuery bool     `mapstructure:"dev_allow_query"` // 开发模式允许用 ?user= 便于本地调试
}

func (c *AuthConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("auth.enabled", true)
	v.SetDefault("auth.dev_allow_query", true)
	v.SetDefault("auth.skip_paths", []string{
		"/metrics",
		"/debug/pprof",
		"/api/v1/health",
		"/swagger",
	})
}
