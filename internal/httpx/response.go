// Package httpx 提供 HTTP 请求处理相关的工具函数。
//
// 本文件实现统一的 JSON 响应格式，包含业务错误码、消息和数据。
// 所有响应均返回 HTTP 200，通过业务码区分成功与失败。
package httpx

import (
	"net/http"

	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// response 是统一的 JSON 响应结构。
type response struct {
	Code xcode.Xcode `json:"code"`        // 业务错误码
	Msg  string      `json:"msg"`         // 错误消息
	Data any         `json:"data,omitempty"` // 响应数据
}

// OK 写入成功响应（错误码 1000）。
//
// 始终返回 HTTP 200。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, response{
		Code: xcode.Success,
		Msg:  xcode.Success.Msg(),
		Data: data,
	})
}

// Fail 写入失败响应。
//
// 参数:
//   - code: 业务错误码
//   - msg: 自定义错误消息（为空时使用默认消息）
//
// 始终返回 HTTP 200。
func Fail(c *gin.Context, code xcode.Xcode, msg string) {
	if msg == "" {
		msg = code.Msg()
	}
	c.JSON(http.StatusOK, response{
		Code: code,
		Msg:  msg,
	})
}

// BindErr 写入参数绑定错误响应（错误码 2000）。
//
// 始终返回 HTTP 200。
func BindErr(c *gin.Context, err error) {
	Fail(c, xcode.ParamError, err.Error())
}

// ServerErr 写入服务器错误响应（错误码 3000）。
//
// 始终返回 HTTP 200。
func ServerErr(c *gin.Context, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	Fail(c, xcode.ServerError, msg)
}

// NotFound 写入资源未找到响应。
//
// 始终返回 HTTP 200。
func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Resource not found"
	}
	Fail(c, xcode.NotFound, msg)
}

// BadRequest 写入请求错误响应。
//
// 始终返回 HTTP 200。
func BadRequest(c *gin.Context, msg string) {
	Fail(c, xcode.ParamError, msg)
}
