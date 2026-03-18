// Package middleware 提供 HTTP 中间件实现。
//
// 本文件实现请求日志中间件，记录每个请求的方法、路径、状态码和耗时。
package middleware

import (
	"time"

	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/gin-gonic/gin"
)

// Logger 返回请求日志中间件。
//
// 记录信息包括：请求方法、路径、状态码、追踪 ID、耗时。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()
		myCtx := runtimectx.FromContext(ctx)

		c.Next()

		traceID := ""
		if myCtx != nil {
			traceID = myCtx.TraceID
		}
		logger.L().Info(
			"http request",
			logger.String("method", c.Request.Method),
			logger.String("path", c.Request.URL.Path),
			logger.Int("status", c.Writer.Status()),
			logger.String("trace_id", traceID),
			logger.String("latency", time.Since(start).String()),
		)
	}
}
