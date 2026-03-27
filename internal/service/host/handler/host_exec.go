// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"fmt"
	"strings"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// SSHCheck 检查主机 SSH 连接。
//
// @Summary 检查 SSH 连接
// @Description 检查主机的 SSH 连接是否可达，需要 host:read 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/{id}/ssh/check [post]
func (h *Handler) SSHCheck(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
	if err != nil {
		httpx.OK(c, gin.H{"reachable": false, "message": err.Error()})
		return
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		httpx.OK(c, gin.H{"reachable": false, "message": err.Error()})
		return
	}
	_ = cli.Close()
	httpx.OK(c, gin.H{"reachable": true})
}

// SSHExec 在主机上执行 SSH 命令。
//
// @Summary 执行 SSH 命令
// @Description 通过 SSH 在指定主机上执行命令，返回标准输出、错误输出和退出码，需要 host:write、host:execute 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body object true "命令请求 {command: string}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/{id}/ssh/exec [post]
func (h *Handler) SSHExec(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:execute", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Command string `json:"command" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
	if err != nil {
		httpx.OK(c, gin.H{"stdout": "", "stderr": err.Error(), "exit_code": 1})
		return
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		httpx.OK(c, gin.H{"stdout": "", "stderr": err.Error(), "exit_code": 1})
		return
	}
	defer cli.Close()
	out, err := sshclient.RunCommand(cli, req.Command)
	if err != nil {
		httpx.OK(c, gin.H{"stdout": out, "stderr": err.Error(), "exit_code": 1})
		return
	}
	httpx.OK(c, gin.H{"stdout": out, "stderr": "", "exit_code": 0})
}

// BatchExec 批量在多台主机上执行 SSH 命令。
//
// @Summary 批量执行 SSH 命令
// @Description 在多台主机上并行执行相同的 SSH 命令，返回各主机的执行结果，需要 host:write、host:execute 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body object true "批量命令请求 {host_ids: [], command: string}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Router /hosts/batch/exec [post]
func (h *Handler) BatchExec(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:execute", "host:*") {
		return
	}
	var req struct {
		HostIDs []uint64 `json:"host_ids"`
		Command string   `json:"command" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	results := map[string]any{}
	for _, id := range req.HostIDs {
		node, err := h.hostService.Get(c.Request.Context(), id)
		if err != nil {
			results[fmt.Sprintf("%d", id)] = gin.H{"stdout": "", "stderr": "host not found", "exit_code": 1}
			continue
		}
		privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
		if err != nil {
			results[fmt.Sprintf("%d", id)] = gin.H{"stdout": "", "stderr": err.Error(), "exit_code": 1}
			continue
		}
		password := strings.TrimSpace(node.SSHPassword)
		if strings.TrimSpace(privateKey) != "" {
			password = ""
		}
		cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
		if err != nil {
			results[fmt.Sprintf("%d", id)] = gin.H{"stdout": "", "stderr": err.Error(), "exit_code": 1}
			continue
		}
		out, err := sshclient.RunCommand(cli, req.Command)
		_ = cli.Close()
		if err != nil {
			results[fmt.Sprintf("%d", id)] = gin.H{"stdout": out, "stderr": err.Error(), "exit_code": 1}
			continue
		}
		results[fmt.Sprintf("%d", id)] = gin.H{"stdout": out, "stderr": "", "exit_code": 0}
	}
	httpx.OK(c, results)
}

// loadNodePrivateKey 加载主机关联的 SSH 私钥。
//
// 从数据库加载主机关联的 SSH 密钥，处理加密存储的私钥解密。
//
// 参数:
//   - c: Gin 上下文
//   - node: 主机模型
//
// 返回:
//   - privateKey: 私钥内容
//   - passphrase: 私钥密码
//   - error: 错误信息
func (h *Handler) loadNodePrivateKey(c *gin.Context, node *model.Node) (string, string, error) {
	if node == nil || node.SSHKeyID == nil {
		return "", "", nil
	}
	var key model.SSHKey
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Select("id", "private_key", "passphrase", "encrypted").
		Where("id = ?", uint64(*node.SSHKeyID)).
		First(&key).Error; err != nil {
		return "", "", err
	}
	passphrase := strings.TrimSpace(key.Passphrase)
	if !key.Encrypted {
		return strings.TrimSpace(key.PrivateKey), passphrase, nil
	}
	privateKey, err := utils.DecryptText(strings.TrimSpace(key.PrivateKey), config.CFG.Security.EncryptionKey)
	if err != nil {
		return "", "", err
	}
	return privateKey, passphrase, nil
}
