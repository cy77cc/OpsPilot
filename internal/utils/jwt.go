// Package utils 提供通用工具函数。
//
// 本文件实现 JWT Token 的生成和解析功能，使用 HMAC-SHA256 签名算法。
package utils

import (
	"errors"
	"time"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/golang-jwt/jwt/v5"
)

// MyClaims 自定义 JWT 声明，包含用户 ID。
type MyClaims struct {
	Uid uint `json:"uid"` // 用户 ID
	jwt.RegisteredClaims
}

// JWT 错误定义。
var (
	ErrTokenExpired     = errors.New("Token is expired")
	ErrTokenInvalid     = errors.New("Token is invalid")
	ErrTokenMalformed   = errors.New("Token is malformed")
	ErrTokenNotValidYet = errors.New("Token is not valid yet")
	ErrTokenNotValidId  = errors.New("Token is not valid id")
	ErrTokenSignature   = errors.New("Token signature is invalid")
	MySecret            = []byte(config.CFG.JWT.Secret) // JWT 签名密钥
)

// GenToken 使用 HMAC-SHA256 生成 JWT Token。
//
// 参数:
//   - id: 用户 ID
//   - isRefreshToken: 是否为刷新 Token
func GenToken(id uint, isRefreshToken bool) (string, error) {

	var tokenExpireDuration = config.CFG.JWT.Expire

	if isRefreshToken {
		tokenExpireDuration = config.CFG.JWT.RefreshExpire
	}

	c := MyClaims{
		id,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpireDuration)),
			Issuer:    config.CFG.JWT.Issuer,
		},
	}
	// 指定加密方式
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	// 使用密钥加密
	return token.SignedString(MySecret)
}

// ParseToken 解析token
func ParseToken(tokenString string) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&MyClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return MySecret, nil
		},
	)
	if err != nil {
		return nil, xcode.NewErrCode(xcode.TokenInvalid)
	}
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, xcode.NewErrCode(xcode.TokenInvalid)
}

// RefreshToken 刷新token
func RefreshToken(id uint, isRefreshToken bool) (string, error) {
	token, err := GenToken(id, isRefreshToken)
	if err != nil {
		return "", err
	}
	return token, nil
}
