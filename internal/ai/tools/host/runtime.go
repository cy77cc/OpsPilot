package host

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/ai/tools/common"
	sshclient "github.com/cy77cc/k8s-manage/internal/client/ssh"
	"github.com/cy77cc/k8s-manage/internal/model"
)

func runLocalCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	out, err := cmd.CombinedOutput()
	if cctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), errors.New("command timeout")
	}
	return strings.TrimSpace(string(out)), err
}

func runOnTarget(ctx context.Context, deps common.PlatformDeps, target, localName string, localArgs []string, remoteCmd string) (string, string, error) {
	node, err := resolveNodeByTarget(deps, target)
	if err != nil {
		return "", "target_check", err
	}
	if node == nil {
		out, err := runLocalCommand(ctx, 6*time.Second, localName, localArgs...)
		return out, "local", err
	}
	privateKey, passphrase, err := loadNodePrivateKey(deps, node)
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

func resolveNodeByTarget(deps common.PlatformDeps, target string) (*model.Node, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || trimmed == "localhost" {
		return nil, nil
	}
	if deps.DB == nil {
		return nil, errors.New("db unavailable")
	}
	var node model.Node
	if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil {
		if err := deps.DB.First(&node, id).Error; err == nil {
			return &node, nil
		}
	}
	if err := deps.DB.Where("ip = ? OR name = ? OR hostname = ?", trimmed, trimmed, trimmed).First(&node).Error; err != nil {
		return nil, errors.New("target not in host whitelist")
	}
	return &node, nil
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case uint64:
		return int(x)
	case json.Number:
		n, _ := strconv.Atoi(x.String())
		return n
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	default:
		return 0
	}
}
