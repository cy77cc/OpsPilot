package deployment

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	sshclient "github.com/cy77cc/k8s-manage/internal/client/ssh"
	"github.com/cy77cc/k8s-manage/internal/config"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/utils"
)

func (l *Logic) applyComposeRelease(ctx context.Context, target *model.DeploymentTarget, releaseID uint, manifest string) (string, error) {
	node, err := l.pickComposeNode(ctx, target.ID)
	if err != nil {
		return "", err
	}
	privateKey, passphrase, err := l.loadNodePrivateKey(ctx, node)
	if err != nil {
		return "", err
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		return "", err
	}
	defer cli.Close()

	workDir := fmt.Sprintf("/tmp/opspilot/releases/%d", releaseID)
	composeFile := fmt.Sprintf("%s/docker-compose.yaml", workDir)
	encoded := base64.StdEncoding.EncodeToString([]byte(manifest))
	cmd := fmt.Sprintf("command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 && mkdir -p %s && echo '%s' | base64 -d > %s && docker compose -f %s pull && docker compose -f %s up -d && docker compose -f %s ps", workDir, encoded, composeFile, composeFile, composeFile, composeFile)
	out, err := sshclient.RunCommand(cli, cmd)
	if err != nil {
		return out, err
	}
	return out, nil
}

func (l *Logic) pickComposeNode(ctx context.Context, targetID uint) (*model.Node, error) {
	var links []model.DeploymentTargetNode
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("target_id = ? AND status = ?", targetID, "active").
		Order("CASE WHEN role = 'manager' THEN 0 ELSE 1 END, id ASC").
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("compose target has no active nodes")
	}
	var node model.Node
	if err := l.svcCtx.DB.WithContext(ctx).First(&node, links[0].HostID).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (l *Logic) loadNodePrivateKey(ctx context.Context, node *model.Node) (string, string, error) {
	if node == nil || node.SSHKeyID == nil {
		return "", "", nil
	}
	var key model.SSHKey
	if err := l.svcCtx.DB.WithContext(ctx).
		Select("id", "private_key", "passphrase", "encrypted").
		Where("id = ?", uint64(*node.SSHKeyID)).
		First(&key).Error; err != nil {
		return "", "", err
	}
	passphrase := strings.TrimSpace(key.Passphrase)
	if !key.Encrypted {
		return strings.TrimSpace(key.PrivateKey), passphrase, nil
	}
	if strings.TrimSpace(config.CFG.Security.EncryptionKey) == "" {
		return "", "", fmt.Errorf("security.encryption_key is required")
	}
	plain, err := utils.DecryptText(strings.TrimSpace(key.PrivateKey), config.CFG.Security.EncryptionKey)
	if err != nil {
		return "", "", err
	}
	return plain, passphrase, nil
}
