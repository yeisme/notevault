// Package configs 管理应用程序配置，包括Tracing的配置信息.
// Tracing配置支持OpenTelemetry等分布式追踪系统.
//
// Example:
//
//	config := configs.GetConfig()
//	tracingConfig := config.Tracing
//	if tracingConfig.Enabled {
//		// 初始化Tracing
//	}
package configs

import (
	"time"

	"github.com/spf13/viper"
)

const (
	// DefaultMaxBatchSize 默认最大批量大小.
	DefaultMaxBatchSize = 512
	// DefaultMaxQueueSize 默认最大队列大小.
	DefaultMaxQueueSize = 2048
)

// TracingConfig Tracing相关配置.
type TracingConfig struct {
	Enabled        bool              `mapstructure:"enabled"`         // 是否启用Tracing
	ServiceName    string            `mapstructure:"service_name"`    // 服务名称
	ServiceVersion string            `mapstructure:"service_version"` // 服务版本
	ExporterType   string            `mapstructure:"exporter_type"`   // 导出器类型，如 "otlp-http", "otlp-grpc", "zipkin"
	Endpoint       string            `mapstructure:"endpoint"`        // 导出器端点
	SampleRate     float64           `mapstructure:"sample_rate"`     // 采样率，0.0-1.0
	BatchTimeout   time.Duration     `mapstructure:"batch_timeout"`   // 批量超时
	MaxBatchSize   int               `mapstructure:"max_batch_size"`  // 最大批量大小
	MaxQueueSize   int               `mapstructure:"max_queue_size"`  // 最大队列大小
	ResourceLabels map[string]string `mapstructure:"resource_labels"` // 资源标签
}

// setDefaults 设置Tracing配置的默认值.
func (c *TracingConfig) setDefaults(v *viper.Viper) {
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.service_name", "notevault")
	v.SetDefault("tracing.service_version", AppVersion)
	v.SetDefault("tracing.exporter_type", "otlp-http")
	v.SetDefault("tracing.endpoint", "http://localhost:4318")
	v.SetDefault("tracing.sample_rate", 1.0)
	v.SetDefault("tracing.batch_timeout", "5s")
	v.SetDefault("tracing.max_batch_size", DefaultMaxBatchSize)
	v.SetDefault("tracing.max_queue_size", DefaultMaxQueueSize)
	v.SetDefault("tracing.resource_labels", map[string]string{
		"service.name":    "notevault",
		"service.version": "1.0.0",
	})
}
