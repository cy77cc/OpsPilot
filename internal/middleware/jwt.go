// Package middleware 提供 HTTP 中间件实现。
//
// 本文件实现 JWT 认证中间件，用于验证请求中的 JWT Token。
// 支持从 Authorization 头或 URL 查询参数获取 Token。
package middleware

import (
	"net/http"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// JWTAuth 返回 JWT 认证中间件。
//
// 验证流程：
//  1. 从 Authorization 头获取 Bearer Token
//  2. 如果没有，尝试从 URL 查询参数 token 获取
//  3. 解析 Token 并将用户 ID 注入到 gin.Context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessTokenH := c.Request.Header.Get("Authorization")
		if accessTokenH == "" {
			if qToken := strings.TrimSpace(c.Query("token")); qToken != "" {
				accessTokenH = "Bearer " + qToken
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, xcode.NewErrCode(xcode.Unauthorized))
				return
			}
		}

		parts := strings.SplitN(accessTokenH, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, xcode.NewErrCode(xcode.Unauthorized))
			return
		}

		accessToken, err := utils.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, xcode.NewErrCode(xcode.TokenInvalid))
			return
		}

		c.Set("uid", accessToken.Uid)
		c.Next()
	}
}
