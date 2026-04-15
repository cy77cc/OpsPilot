// Package middleware 提供 HTTP 中间件实现。
//
// 本文件实现 JWT 认证中间件，用于验证请求中的 JWT Token。
// 支持 Bearer 头和安全 Cookie 传输。
package middleware

import (
	"net/http"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	"github.com/gin-gonic/gin"
)

const authAccessCookieName = "opspilot_at"

// JWTAuth 返回 JWT 认证中间件。
//
// 验证流程：
//  1. 从 Authorization 头获取 Bearer Token（优先）
//  2. 若头不存在，则回退到安全 Cookie（opspilot_at）
//  3. 解析 Token 并将用户 ID 注入到 gin.Context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.Request.Header.Get("Authorization"))
		if !ok {
			cookieToken, err := c.Cookie(authAccessCookieName)
			if err == nil {
				token = strings.TrimSpace(cookieToken)
				ok = token != ""
			}
		}

		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, xcode.NewErrCode(xcode.Unauthorized))
			return
		}

		accessToken, err := utils.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, xcode.NewErrCode(xcode.TokenInvalid))
			return
		}

		c.Set("uid", accessToken.Uid)
		c.Next()
	}
}

func bearerToken(authHeader string) (string, bool) {
	if strings.TrimSpace(authHeader) == "" {
		return "", false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}
