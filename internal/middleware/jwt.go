package middleware

import (
	"net/http"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/utils"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

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
