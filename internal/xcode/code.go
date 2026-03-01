package xcode

import (
	"fmt"
	"net/http"
)

type Xcode uint32

// Success codes (1000~1999)
const (
	Success       Xcode = 1000 // Success
	CreateSuccess Xcode = 1001 // Create Success
	DeleteSuccess Xcode = 1002 // Delete Success
	UpdateSuccess Xcode = 1003 // Update Success
)

// Client errors (2000~2999)
const (
	ParamError      Xcode = 2000 // Parameter Error
	MissingParam    Xcode = 2001 // Missing Parameter
	MethodNotAllow  Xcode = 2002 // Method Not Allowed
	Unauthorized    Xcode = 2003 // Unauthorized
	Forbidden       Xcode = 2004 // Forbidden
	NotFound        Xcode = 2005 // Not Found
	ErrInvalidParam Xcode = 2006 // Invalid Parameter
)

// Server errors (3000~3999)
const (
	ServerError     Xcode = 3000 // Internal Server Error
	DatabaseError   Xcode = 3001 // Database Error
	CacheError      Xcode = 3002 // Cache Error
	ExternalAPIFail Xcode = 3003 // External API Failure
	TimeoutError    Xcode = 3004 // Timeout
)

// Business errors (4000~4999)
const (
	FileUploadFail         Xcode = 4000 // File Upload Failed
	FileTypeInvalid        Xcode = 4001 // Invalid File Type
	UserAlreadyExist       Xcode = 4002 // User Already Exists
	UserNotExist           Xcode = 4003 // User Not Exists
	PasswordError          Xcode = 4004 // Password Error
	TokenExpired           Xcode = 4005 // Token Expired
	TokenInvalid           Xcode = 4006 // Token Invalid
	PermissionDenied       Xcode = 4007 // Permission Denied
	PermissionAlreadyExist Xcode = 4008 // Permission Already Exists
	LoginFailed            Xcode = 4009 // Login Failed
)

// Msg returns the message corresponding to the Xcode
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
	default:
		return "未知错误"
	}
}

// CodeError wraps Xcode and message
type CodeError struct {
	Code Xcode  `json:"code"`
	Msg  string `json:"msg"`
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.Code, e.Msg)
}

// New creates a new CodeError
func New(code Xcode, msg string) error {
	return &CodeError{Code: code, Msg: msg}
}

// NewErrCode creates a CodeError from Xcode, using default message
func NewErrCode(code Xcode) error {
	return &CodeError{Code: code, Msg: code.Msg()}
}

// NewErrCodeMsg creates a CodeError from Xcode with custom message
func NewErrCodeMsg(code Xcode, msg string) error {
	return &CodeError{Code: code, Msg: msg}
}

// FromError converts an error to CodeError.
func FromError(err error) *CodeError {
	if err == nil {
		return nil
	}
	if e, ok := err.(*CodeError); ok {
		return e
	}
	return &CodeError{Code: ServerError, Msg: err.Error()}
}

// HttpStatus converts Xcode to HTTP status code
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

