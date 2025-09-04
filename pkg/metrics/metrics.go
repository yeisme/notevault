// Package metrics 提供监控指标功能.
// 支持Prometheus标准，收集应用和系统指标.
//
// Example:
//
//	import "github.com/yeisme/notevault/pkg/metrics"
//
//	err := metrics.InitMetrics(config.Metrics)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 记录指标
//	metrics.RequestCounter.WithLabelValues("GET", "/api/notes").Inc()
//	metrics.RequestDuration.WithLabelValues("GET", "/api/notes").Observe(0.1)
package metrics

import (
	"net/http"
	_ "net/http/pprof" // 自动注册pprof端点

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yeisme/notevault/pkg/configs"
)

// 全局指标变量.
var (
	// RequestCounter HTTP请求计数器.
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	// RequestDuration HTTP请求持续时间.
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// ActiveConnections 活跃连接数.
	ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)

	// registry Prometheus注册表.
	registry = prometheus.NewRegistry()
)

// InitMetrics 初始化Metrics.
func InitMetrics(config configs.MetricsConfig) error {
	if !config.Enabled {
		return nil
	}

	// 注册标准收集器
	if config.RuntimeMetrics {
		registry.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)
	}

	// 注册自定义指标
	registry.MustRegister(RequestCounter, RequestDuration, ActiveConnections)

	// TODO 注册自定义指标
	for _, metric := range config.CustomMetrics {
		// 这里可以根据metric名称注册自定义指标
		// 暂时留空，未来扩展
		_ = metric
	}

	return nil
}

// StartMetricsServer 启动Metrics HTTP服务器.
func StartMetricsServer(config configs.MetricsConfig, debugEngine *gin.Engine) error {
	if !config.Enabled {
		return nil
	}

	debugEngine.GET("/metrics", gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))

	// 如果启用pprof，注册pprof端点
	if config.Pprof {
		debugEngine.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
	}

	return nil
}

// GetRegistry 获取Prometheus注册表.
func GetRegistry() *prometheus.Registry {
	return registry
}

// NewCounter 创建新的计数器指标.
func NewCounter(name, help string, labels []string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	registry.MustRegister(counter)

	return counter
}

// NewGauge 创建新的仪表盘指标.
func NewGauge(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	registry.MustRegister(gauge)

	return gauge
}

// NewHistogram 创建新的直方图指标.
func NewHistogram(name, help string, labels []string) *prometheus.HistogramVec {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	registry.MustRegister(histogram)

	return histogram
}
