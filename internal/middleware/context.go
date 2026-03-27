// Package middleware 提供 HTTP 中间件实现。
//
// 本文件实现上下文注入中间件，将自定义上下文注入到请求上下文中。
// 这样后续的 Service 层即使只拿到了 context.Context，也能取出追踪信息。
package middleware

import (
	"time"

	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContextMiddleware 返回上下文注入中间件。
//
// 功能：
//   - 初始化运行时上下文（runtimectx.Context）
//   - 生成或复用追踪 ID
//   - 将上下文注入到 gin.Context 中
func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 初始化自定义 Context
		myCtx := runtimectx.NewContext()
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		myCtx.TraceID = traceID
		myCtx.StartTime = time.Now().Unix()

		// 2. 将其注入到标准 Context 中
		// 这一点至关重要：这样后续的 Service 层即使只拿到了 context.Context，也能取出 myCtx
		ctx := runtimectx.WithContext(c.Request.Context(), myCtx)

		// 3. 更新 Gin 的 Request，以便后续 c.Request.Context() 能拿到包含 myCtx 的 context
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
