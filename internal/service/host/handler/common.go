// Package handler 提供主机管理服务的 HTTP 处理器。
//
// 本包包含所有主机相关的 HTTP Handler 实现，包括：
//   - 主机 CRUD 操作 (host_query.go, host_mutation.go)
//   - SSH 连接和命令执行 (host_exec.go)
//   - 终端会话管理 (terminal_session.go)
//   - 文件管理 (files_handler.go)
//   - 云主机导入 (cloud_handler.go)
//   - KVM 虚拟化 (virtualization_handler.go)
//   - SSH 密钥管理 (credentials_handler.go)
package handler

import (
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Handler 主机管理 HTTP 处理器。
//
// 聚合所有主机相关操作的 HTTP Handler，包括主机 CRUD、SSH 操作、
// 文件管理、终端会话、云主机导入等功能。
type Handler struct {
	// svcCtx 服务上下文，提供数据库、缓存等依赖
	svcCtx *svc.ServiceContext

	// hostService 主机业务逻辑服务
	hostService *hostlogic.HostService
}

// NewHandler 创建主机管理 Handler 实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{
		svcCtx:      svcCtx,
		hostService: hostlogic.NewHostService(svcCtx),
	}
}

// StartHealthCollector 启动主机健康检查收集器。
//
// 启动后台定时任务收集主机健康快照。
// 使用 sync.Once 确保只启动一次。
func (h *Handler) StartHealthCollector() {
	h.hostService.StartHealthSnapshotCollector()
}

// parseID 从 URL 路径参数中解析主机 ID。
//
// 参数:
//   - c: Gin 上下文
//
// 返回:
//   - id: 解析后的主机 ID
//   - ok: 解析是否成功，失败时已返回错误响应
func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return 0, false
	}
	return id, true
}

// getUID 从 Gin 上下文中获取当前用户 ID。
//
// 从 JWT 认证中间件注入的 "uid" 字段获取用户 ID。
// 支持多种数值类型转换 (uint, uint64, int, int64, float64)。
//
// 参数:
//   - c: Gin 上下文
//
// 返回: 用户 ID，未找到时返回 0
func getUID(c *gin.Context) uint64 {
	uid, ok := c.Get("uid")
	if !ok {
		return 0
	}
	switch v := uid.(type) {
	case uint:
		return uint64(v)
	case uint64:
		return v
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case float64:
		return uint64(v)
	default:
		return 0
	}
}
