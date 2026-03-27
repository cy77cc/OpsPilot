package logic

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/utils"
	golangssh "golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// SSHKeyCreateReq 创建 SSH 密钥请求参数。
type SSHKeyCreateReq struct {
	Name       string `json:"name"`        // 密钥名称
	PrivateKey string `json:"private_key"` // 私钥内容
	Passphrase string `json:"passphrase"`  // 私钥密码（可选）
}

// SSHKeyVerifyReq 验证 SSH 密钥请求参数。
type SSHKeyVerifyReq struct {
	IP       string `json:"ip"`       // 目标主机 IP
	Port     int    `json:"port"`     // SSH 端口
	Username string `json:"username"` // SSH 用户名
}

// ListSSHKeys 获取 SSH 密钥列表。
//
// 返回所有 SSH 密钥的基本信息，私钥已脱敏。
//
// 参数:
//   - ctx: 上下文
//
// 返回: SSH 密钥列表
func (s *HostService) ListSSHKeys(ctx context.Context) ([]model.SSHKey, error) {
	var list []model.SSHKey
	err := s.svcCtx.DB.WithContext(ctx).Select("id", "name", "public_key", "fingerprint", "algorithm", "encrypted", "usage_count", "created_at", "updated_at").
		Order("id desc").
		Find(&list).Error
	return list, err
}

// CreateSSHKey 创建 SSH 密钥。
//
// 验证私钥格式，提取公钥和指纹，加密存储私钥。
//
// 参数:
//   - ctx: 上下文
//   - req: 创建请求参数
//
// 返回: 创建的 SSH 密钥对象（私钥已脱敏）
func (s *HostService) CreateSSHKey(ctx context.Context, req SSHKeyCreateReq) (*model.SSHKey, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("name is required")
	}
	if strings.TrimSpace(req.PrivateKey) == "" {
		return nil, errors.New("private_key is required")
	}
	if strings.TrimSpace(config.CFG.Security.EncryptionKey) == "" {
		return nil, errors.New("security.encryption_key is required")
	}
	pub, alg, fp, err := parsePrivateKeyMeta(req.PrivateKey, req.Passphrase)
	if err != nil {
		return nil, err
	}
	cipher, err := utils.EncryptText(req.PrivateKey, config.CFG.Security.EncryptionKey)
	if err != nil {
		return nil, err
	}
	key := &model.SSHKey{
		Name:        req.Name,
		PublicKey:   pub,
		PrivateKey:  cipher,
		Passphrase:  req.Passphrase,
		Fingerprint: fp,
		Algorithm:   alg,
		Encrypted:   true,
	}
	if err := s.svcCtx.DB.WithContext(ctx).Create(key).Error; err != nil {
		return nil, err
	}
	key.PrivateKey = ""
	key.Passphrase = ""
	return key, nil
}

// DeleteSSHKey 删除 SSH 密钥。
//
// 检查密钥是否被主机引用，被引用的密钥无法删除。
//
// 参数:
//   - ctx: 上下文
//   - id: 密钥 ID
//
// 返回: 删除错误
func (s *HostService) DeleteSSHKey(ctx context.Context, id uint64) error {
	var count int64
	if err := s.svcCtx.DB.WithContext(ctx).Model(&model.Node{}).Where("ssh_key_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("ssh key is in use by hosts")
	}
	return s.svcCtx.DB.WithContext(ctx).Delete(&model.SSHKey{}, id).Error
}

// VerifySSHKey 验证 SSH 密钥。
//
// 使用指定密钥尝试连接目标主机，验证密钥是否可用。
// 连接成功后更新密钥的使用次数。
//
// 参数:
//   - ctx: 上下文
//   - id: 密钥 ID
//   - req: 验证请求参数
//
// 返回: 验证结果 {reachable: bool, hostname: string, message: string}
func (s *HostService) VerifySSHKey(ctx context.Context, id uint64, req SSHKeyVerifyReq) (map[string]any, error) {
	if req.Port <= 0 {
		req.Port = DefaultSSHPort
	}
	if req.Username == "" {
		req.Username = "root"
	}
	if strings.TrimSpace(req.IP) == "" {
		return nil, errors.New("ip is required")
	}
	privateKey, passphrase, err := s.loadPrivateKey(ctx, &id)
	if err != nil {
		return nil, err
	}
	cli, err := sshclient.NewSSHClient(req.Username, "", req.IP, req.Port, privateKey, passphrase)
	if err != nil {
		return map[string]any{"reachable": false, "message": err.Error()}, nil
	}
	defer cli.Close()
	out, err := sshclient.RunCommand(cli, "hostname")
	if err != nil {
		return map[string]any{"reachable": false, "message": err.Error()}, nil
	}
	_ = s.svcCtx.DB.WithContext(ctx).Model(&model.SSHKey{}).Where("id = ?", id).UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error
	return map[string]any{"reachable": true, "hostname": out}, nil
}

// parsePrivateKeyMeta 解析私钥元数据。
//
// 从私钥中提取公钥、算法类型和指纹。
//
// 参数:
//   - privateKey: 私钥内容
//   - passphrase: 私钥密码（可选）
//
// 返回:
//   - publicKey: 公钥内容（OpenSSH 格式）
//   - algorithm: 算法类型（如 ssh-rsa, ssh-ed25519）
//   - fingerprint: SHA256 指纹
//   - err: 解析错误
func parsePrivateKeyMeta(privateKey string, passphrase string) (publicKey string, algorithm string, fingerprint string, err error) {
	var signer golangssh.Signer
	if passphrase != "" {
		signer, err = golangssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
	} else {
		signer, err = golangssh.ParsePrivateKey([]byte(privateKey))
	}
	if err != nil {
		return "", "", "", fmt.Errorf("invalid private key: %w", err)
	}
	pub := signer.PublicKey()
	pubBytes := pub.Marshal()
	hash := sha256.Sum256(pubBytes)
	fp := "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
	return strings.TrimSpace(string(golangssh.MarshalAuthorizedKey(pub))), pub.Type(), fp, nil
}
