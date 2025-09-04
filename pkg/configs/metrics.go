// Package configs 管理应用程序配置，包括Metrics的配置信息.
// Metrics配置支持Prometheus等监控系统.
//
// Example:
//
//	config := configs.GetConfig()
//	metricsConfig := config.Metrics
//	if metricsConfig.Enabled {
//		// 初始化Metrics
//	}
package configs

import (
	"time"

	"github.com/spf13/viper"
)

// MetricsConfig Metrics相关配置.
type MetricsConfig struct {
	Enabled         bool              `mapstructure:"enabled"`          // 是否启用Metrics
	ServiceName     string            `mapstructure:"service_name"`     // 服务名称
	ServiceVersion  string            `mapstructure:"service_version"`  // 服务版本
	ExporterType    string            `mapstructure:"exporter_type"`    // 导出器类型，如 "prometheus", "statsd"
	Endpoint        string            `mapstructure:"endpoint"`         // 导出器端点
	CollectInterval time.Duration     `mapstructure:"collect_interval"` // 收集间隔
	RuntimeMetrics  bool              `mapstructure:"runtime_metrics"`  // 是否收集运行时指标
	CustomMetrics   []string          `mapstructure:"custom_metrics"`   // 自定义指标列表
	Labels          map[string]string `mapstructure:"labels"`           // 默认标签
}

// setDefaults 设置Metrics配置的默认值.
func (c *MetricsConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("metrics.enabled", false)
	v.SetDefault("metrics.service_name", "notevault")
	v.SetDefault("metrics.service_version", "1.0.0")
	v.SetDefault("metrics.exporter_type", "prometheus")
	v.SetDefault("metrics.endpoint", ":9090")
	v.SetDefault("metrics.collect_interval", "15s")
	v.SetDefault("metrics.runtime_metrics", true)
	v.SetDefault("metrics.custom_metrics", []string{})
	v.SetDefault("metrics.labels", map[string]string{
		"service": "notevault",
		"version": "1.0.0",
	})
}
