// Package todo 提供向后兼容的类型重导出。
//
// 所有类型已迁移至 internal/ai/common/todo 包。
// 本文件保留向后兼容性，新代码应直接使用 common/todo 包。
package todo

import (
	"github.com/cy77cc/OpsPilot/internal/ai/common/todo"
)

// 函数重导出。
var (
	NewWriteOpsTodosMiddleware = todo.NewWriteOpsTodosMiddleware
)

// 常量重导出。
const (
	SessionKeyOpsTodos = todo.SessionKeyOpsTodos
)

// 类型重导出。
type (
	OpsTODO = todo.OpsTODO
)
