// Package host 提供主机运维相关的运行时辅助函数。
//
// 本文件实现命令执行的核心运行时逻辑，包括：
//   - 本地命令执行
//   - 远程 SSH 命令执行
//   - 目标主机解析
package host

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// runLocalCommand 在本地执行命令。
//
// 参数:
//   - ctx: 上下文
//   - timeout: 超时时间
//   - name: 命令名称
//   - args: 命令参数
//
// 返回:
//   - 命令输出和错误
func 	runLocalCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	out, err := cmd.CombinedOutput()
	if cctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), errors.New("command timeout")
	}
	return strings.TrimSpace(string(out)), err
}

// runOnTarget 在指定目标上执行命令。
//
// 如果目标是 localhost 或空，则在本地执行；
// 否则通过 SSH 在远程主机执行。
//
// 参数:
//   - ctx: 上下文
//   - svcCtx: 平台依赖
//   - target: 目标主机（ID、IP 或主机名）
//   - localName: 本地命令名称
//   - localArgs: 本地命令参数
//   - remoteCmd: 远程命令字符串
//
// 返回:
//   - 输出内容
//   - 执行来源（"local" 或 "remote_ssh"）
//   - 错误
func runOnTarget(ctx context.Context, svcCtx *svc.ServiceContext, target, localName string, localArgs []string, remoteCmd string) (string, string, error) {
	node, err := resolveNodeByTarget(svcCtx, target)
	if err != nil {
		return "", "target_check", err
	}
	if node == nil {
		// 目标为 localhost，本地执行
		out, err := runLocalCommand(ctx, 6*time.Second, localName, localArgs...)
		return out, "local", err
	}
	// 远程 SSH 执行
	privateKey, passphrase, err := loadNodePrivateKey(svcCtx, node)
	if err != nil {
		return "", "remote_ssh_credential", err
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		return "", "remote_ssh", err
	}
	defer cli.Close()
	out, err := sshclient.RunCommand(cli, remoteCmd)
	return out, "remote_ssh", err
}

// resolveNodeByTarget 根据目标标识解析主机节点。
//
// 目标可以是：
//   - 空或 "localhost"：返回 nil（本地执行）
//   - 数字 ID：按 ID 查询
//   - IP 地址或主机名：按 IP/name/hostname 查询
//
// 如果目标不在白名单中，返回错误。
func resolveNodeByTarget(svcCtx *svc.ServiceContext, target string) (*model.Node, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || trimmed == "localhost" {
		return nil, nil
	}
	if svcCtx.DB == nil {
		return nil, errors.New("db unavailable")
	}
	var node model.Node
	// 尝试按 ID 解析
	if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil {
		if err := svcCtx.DB.First(&node, id).Error; err == nil {
			return &node, nil
		}
	}
	// 按 IP/name/hostname 查询
	if err := svcCtx.DB.Where("ip = ? OR name = ? OR hostname = ?", trimmed, trimmed, trimmed).First(&node).Error; err != nil {
		return nil, errors.New("target not in host whitelist")
	}
	return &node, nil
}
