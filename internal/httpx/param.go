// Package httpx 提供 HTTP 请求处理相关的工具函数。
//
// 本文件实现参数提取和类型转换功能，
// 用于从 gin.Context 中提取用户 ID、路由参数和查询参数。
package httpx

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// UIDFromCtx 从 gin.Context 中提取用户 ID。
//
// 用户 ID 由 JWT 中间件设置到上下文中。
// 如果不存在或无法转换，返回 0。
func UIDFromCtx(c *gin.Context) uint64 {
	v, ok := c.Get("uid")
	if !ok {
		return 0
	}
	return ToUint64(v)
}

// ToUint64 将常见数值类型转换为 uint64。
//
// 支持的类型：uint, uint64, int, int64, float64。
// 负数返回 0。
func ToUint64(v any) uint64 {
	switch x := v.(type) {
	case uint:
		return uint64(x)
	case uint64:
		return x
	case int:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case int64:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case float64:
		if x < 0 {
			return 0
		}
		return uint64(x)
	default:
		return 0
	}
}

// UintFromParam 从路由路径参数中解析 uint 值。
//
// 如果参数缺失或不是有效数字，返回 0。
func UintFromParam(c *gin.Context, key string) uint {
	v, _ := strconv.ParseUint(c.Param(key), 10, 64)
	return uint(v)
}

// UintFromQuery 从查询字符串参数中解析 uint 值。
//
// 自动去除首尾空白。如果参数缺失或不是有效数字，返回 0。
func UintFromQuery(c *gin.Context, key string) uint {
	v, _ := strconv.ParseUint(strings.TrimSpace(c.Query(key)), 10, 64)
	return uint(v)
}
