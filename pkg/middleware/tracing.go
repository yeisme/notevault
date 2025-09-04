package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/yeisme/notevault/pkg/tracing"
)

// TracingMiddleware 创建Gin的分布式追踪中间件.
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求上下文中获取或创建span
		ctx, span := tracing.StartSpan(c.Request.Context(), "http.request",
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.scheme", c.Request.URL.Scheme),
				attribute.String("http.host", c.Request.Host),
				attribute.String("http.path", c.Request.URL.Path),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.remote_addr", c.ClientIP()),
			),
		)
		defer span.End()

		// 将span的context设置到gin的context中，以便后续使用
		c.Request = c.Request.WithContext(ctx)

		// 执行下一个中间件/处理器
		c.Next()

		// 记录响应信息
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
		)

		// 如果有错误，记录错误
		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}
