// Package middleware 提供 HTTP 中间件实现。
//
// 本文件实现 CORS 跨域处理中间件，根据配置设置跨域响应头。
package middleware

import (
	"fmt"
	"net/http"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/gin-gonic/gin"
)

// Cors 返回 CORS 跨域处理中间件。
//
// 根据配置设置以下响应头：
//   - Access-Control-Allow-Origin
//   - Access-Control-Allow-Methods
//   - Access-Control-Allow-Headers
//   - Access-Control-Expose-Headers
//   - Access-Control-Allow-Credentials
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", config.CFG.Cors.AllowOrigins[0]) // 可将将 * 替换为指定的域名
		c.Header("Access-Control-Allow-Methods", utils.SlicesToString(config.CFG.Cors.AllowMethods, ","))
		c.Header("Access-Control-Allow-Headers", utils.SlicesToString(config.CFG.Cors.AllowHeaders, ","))
		c.Header("Access-Control-Expose-Headers", utils.SlicesToString(config.CFG.Cors.ExposeHeaders, ","))
		c.Header("Access-Control-Allow-Credentials", fmt.Sprintf("%t", config.CFG.Cors.AllowCredentials))
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
