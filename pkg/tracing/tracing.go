// Package tracing 提供分布式追踪功能.
// 支持OpenTelemetry标准，集成Jaeger、Zipkin等后端.
//
// Example:
//
//	import "github.com/yeisme/notevault/pkg/tracing"
//
//	err := tracing.InitTracer(config.Tracing)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer tracing.ShutdownTracer()
//
//	// 在代码中使用
//	ctx, span := tracing.StartSpan(ctx, "operation_name")
//	defer span.End()
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/yeisme/notevault/pkg/configs"
)

// tracerProvider 全局TracerProvider.
var tracerProvider *sdktrace.TracerProvider

// InitTracer 初始化Tracer.
func InitTracer(config configs.TracingConfig) error {
	if !config.Enabled {
		return nil
	}

	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			attribute.String("service.name", config.ServiceName),
			attribute.String("service.version", config.ServiceVersion),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// 根据导出器类型创建导出器
	var exporter sdktrace.SpanExporter

	switch config.ExporterType {
	case "otlp-http":
		exporter, err = otlptracehttp.New(context.Background(), otlptracehttp.WithEndpointURL(config.Endpoint))
		if err != nil {
			return fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
		}
	case "otlp-grpc":
		exporter, err = otlptracegrpc.New(context.Background(), otlptracegrpc.WithEndpoint(config.Endpoint))
		if err != nil {
			return fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
		}
	case "zipkin":
		exporter, err = zipkin.New(config.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to create zipkin exporter: %w", err)
		}
	default:
		return fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}

	// 创建TracerProvider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	otel.SetTracerProvider(tracerProvider)

	return nil
}

// ShutdownTracer 关闭Tracer.
func ShutdownTracer(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}

	return nil
}

// StartSpan 开始一个新的Span
// 关闭时调用 span.End().
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("notevault").Start(ctx, spanName, opts...)
}

// GetTracer 获取Tracer.
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
