// Package xcode 提供统一的业务错误码定义。
//
// 本文件定义了应用程序使用的所有业务错误码，分为四类：
//   - 成功码 (1000~1999)
//   - 客户端错误 (2000~2999)
//   - 服务端错误 (3000~3999)
//   - 业务错误 (4000~4999)
package xcode

import (
	"fmt"
	"net/http"
)

// Xcode 是业务错误码类型。
type Xcode uint32

// 成功码 (1000~1999)
const (
	Success       Xcode = 1000 // 请求成功
	CreateSuccess Xcode = 1001 // 创建成功
	DeleteSuccess Xcode = 1002 // 删除成功
	UpdateSuccess Xcode = 1003 // 更新成功
)

// 客户端错误码 (2000~2999)
const (
	ParamError      Xcode = 2000 // 参数错误
	MissingParam    Xcode = 2001 // 缺少必要参数
	MethodNotAllow  Xcode = 2002 // 请求方法不支持
	Unauthorized    Xcode = 2003 // 未认证
	Forbidden       Xcode = 2004 // 无权限
	NotFound        Xcode = 2005 // 资源不存在
	ErrInvalidParam Xcode = 2006 // 无效参数
)

// 服务端错误码 (3000~3999)
const (
	ServerError     Xcode = 3000 // 服务器内部错误
	DatabaseError   Xcode = 3001 // 数据库错误
	CacheError      Xcode = 3002 // 缓存服务错误
	ExternalAPIFail Xcode = 3003 // 外部服务调用失败
	TimeoutError    Xcode = 3004 // 请求超时
)

// 业务错误码 (4000~4999)
const (
	FileUploadFail          Xcode = 4000 // 文件上传失败
	FileTypeInvalid         Xcode = 4001 // 文件格式不支持
	UserAlreadyExist        Xcode = 4002 // 用户已存在
	UserNotExist            Xcode = 4003 // 用户不存在
	PasswordError           Xcode = 4004 // 密码错误
	TokenExpired            Xcode = 4005 // Token 已过期
	TokenInvalid            Xcode = 4006 // Token 无效
	PermissionDenied        Xcode = 4007 // 权限不足
	PermissionAlreadyExist  Xcode = 4008 // 权限已存在
	LoginFailed             Xcode = 4009 // 登录失败
	LLMProviderNotFound     Xcode = 4010 // 模型配置不存在
	LLMProviderDisabled     Xcode = 4011 // 模型已禁用
	LLMProviderInUse        Xcode = 4012 // 模型正在使用中
	LLMImportInvalidJSON    Xcode = 4013 // JSON 格式无效
	LLMImportValidationFail Xcode = 4014 // 导入配置验证失败
)

// Msg 返回错误码对应的中文消息。
func (c Xcode) Msg() string {
	switch c {
	case Success:
		return "请求成功"
	case CreateSuccess:
		return "创建成功"
	case DeleteSuccess:
		return "删除成功"
	case UpdateSuccess:
		return "更新成功"

	case ParamError:
		return "参数错误"
	case MissingParam:
		return "缺少必要参数"
	case MethodNotAllow:
		return "请求方法不支持"
	case Unauthorized:
		return "未认证"
	case Forbidden:
		return "无权限"
	case NotFound:
		return "资源不存在"

	case ServerError:
		return "服务器内部错误"
	case DatabaseError:
		return "数据库错误"
	case CacheError:
		return "缓存服务错误"
	case ExternalAPIFail:
		return "外部服务调用失败"
	case TimeoutError:
		return "请求超时"

	case FileUploadFail:
		return "文件上传失败"
	case FileTypeInvalid:
		return "文件格式不支持"
	case UserAlreadyExist:
		return "用户已存在"
	case UserNotExist:
		return "用户不存在"
	case PasswordError:
		return "密码错误"
	case TokenExpired:
		return "Token 已过期"
	case TokenInvalid:
		return "Token 无效"
	case PermissionDenied:
		return "权限不足"
	case LLMProviderNotFound:
		return "模型配置不存在"
	case LLMProviderDisabled:
		return "模型已禁用"
	case LLMProviderInUse:
		return "模型正在使用中"
	case LLMImportInvalidJSON:
		return "JSON 格式无效"
	case LLMImportValidationFail:
		return "导入配置验证失败"
	default:
		return "未知错误"
	}
}

// CodeError 封装业务错误码和消息。
type CodeError struct {
	Code Xcode  `json:"code"` // 业务错误码
	Msg  string `json:"msg"`  // 错误消息
}

// Error 实现 error 接口。
func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.Code, e.Msg)
}

// New 创建新的 CodeError。
func New(code Xcode, msg string) error {
	return &CodeError{Code: code, Msg: msg}
}

// NewErrCode 使用错误码创建 CodeError，使用默认消息。
func NewErrCode(code Xcode) error {
	return &CodeError{Code: code, Msg: code.Msg()}
}

// NewErrCodeMsg 使用错误码和自定义消息创建 CodeError。
func NewErrCodeMsg(code Xcode, msg string) error {
	return &CodeError{Code: code, Msg: msg}
}

// FromError 将 error 转换为 CodeError。
//
// 如果 err 为 nil 返回 nil。
// 如果 err 已经是 *CodeError 则直接返回。
// 否则包装为 ServerError。
func FromError(err error) *CodeError {
	if err == nil {
		return nil
	}
	if e, ok := err.(*CodeError); ok {
		return e
	}
	return &CodeError{Code: ServerError, Msg: err.Error()}
}

// HttpStatus 将业务错误码转换为 HTTP 状态码。
func (c Xcode) HttpStatus() int {
	switch c {
	case Success, CreateSuccess, DeleteSuccess, UpdateSuccess:
		return http.StatusOK
	case ParamError, MissingParam, FileTypeInvalid:
		return http.StatusBadRequest
	case Unauthorized, TokenExpired, TokenInvalid:
		return http.StatusUnauthorized
	case Forbidden, PermissionDenied:
		return http.StatusForbidden
	case NotFound, UserNotExist:
		return http.StatusNotFound
	case LLMProviderNotFound:
		return http.StatusNotFound
	case LLMProviderDisabled, LLMProviderInUse, LLMImportInvalidJSON, LLMImportValidationFail:
		return http.StatusBadRequest
	case MethodNotAllow:
		return http.StatusMethodNotAllowed
	case TimeoutError:
		return http.StatusRequestTimeout
	case ServerError, DatabaseError, CacheError, ExternalAPIFail, FileUploadFail:
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}
