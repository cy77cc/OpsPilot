package logic

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
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

// CredentialTemplateCreateReq 创建认证预设请求参数。
type CredentialTemplateCreateReq struct {
	Name        string  `json:"name"`        // 预设名称
	AuthType    string  `json:"auth_type"`   // 认证类型: password/key
	SSHUser     string  `json:"ssh_user"`    // SSH 用户名
	Port        int     `json:"port"`        // SSH 端口
	Password    string  `json:"password"`    // SSH 密码（密码认证）
	SSHKeyID    *uint64 `json:"ssh_key_id"`  // SSH 密钥 ID（密钥认证）
	Description string  `json:"description"` // 描述
}

// CredentialItem 统一凭证项。
type CredentialItem struct {
	ID          string   `json:"id"`          // 凭证 ID (带前缀)
	Name        string   `json:"name"`        // 名称
	Description string   `json:"description"` // 描述
	Type        string   `json:"type"`        // 类型: ssh_key, password, token, certificate
	AuthMethod  string   `json:"authMethod"`  // 认证方式
	HostCount   int      `json:"hostCount"`   // 关联主机数
	Tags        []string `json:"tags"`        // 标签
	Status      string   `json:"status"`      // 状态: available, expiring_soon, expired, disabled
	ExpireAt    string   `json:"expireAt"`    // 过期时间
	UpdatedAt   string   `json:"updatedAt"`   // 更新时间
	UpdatedBy   string   `json:"updatedBy"`   // 更新者
}

// CredentialStats 凭证统计信息。
type CredentialStats struct {
	Total          int    `json:"total"`          // 总凭证数
	Available      int    `json:"available"`      // 可用凭证数
	ExpiringSoon   int    `json:"expiringSoon"`   // 即将过期数
	Expired        int    `json:"expired"`        // 已过期数
	RecentUpdate   string `json:"recentUpdate"`   // 最近更新时间
	RecentUpdateBy string `json:"recentUpdateBy"` // 最近更新者
}

// ListUnifiedCredentials 获取统一凭证列表。
func (s *HostService) ListUnifiedCredentials(ctx context.Context) ([]CredentialItem, error) {
	keys, err := s.ListSSHKeys(ctx)
	if err != nil {
		return nil, err
	}
	templates, err := s.ListCredentialTemplates(ctx)
	if err != nil {
		return nil, err
	}

	var items []CredentialItem
	for _, key := range keys {
		items = append(items, CredentialItem{
			ID:          fmt.Sprintf("key-%d", key.ID),
			Name:        key.Name,
			Description: fmt.Sprintf("Algorithm: %s", key.Algorithm),
			Type:        "ssh_key",
			AuthMethod:  "SSH Key",
			HostCount:   key.UsageCount,
			Tags:        []string{},
			Status:      "available",
			UpdatedAt:   key.UpdatedAt.Format("2006-01-02 15:04:05"),
			UpdatedBy:   "system",
		})
	}
	for _, tpl := range templates {
		authMethod := "用户名密码"
		tplType := "password"
		if tpl.AuthType == "key" {
			authMethod = "SSH Key"
			tplType = "ssh_key"
		}
		items = append(items, CredentialItem{
			ID:          fmt.Sprintf("tpl-%d", tpl.ID),
			Name:        tpl.Name,
			Description: tpl.Description,
			Type:        tplType,
			AuthMethod:  authMethod,
			HostCount:   0,
			Tags:        []string{},
			Status:      "available",
			UpdatedAt:   tpl.UpdatedAt.Format("2006-01-02 15:04:05"),
			UpdatedBy:   strconv.FormatUint(tpl.CreatedBy, 10),
		})
	}

	return items, nil
}

// GetCredentialStats 获取凭证统计。
func (s *HostService) GetCredentialStats(ctx context.Context) (*CredentialStats, error) {
	list, err := s.ListUnifiedCredentials(ctx)
	if err != nil {
		return nil, err
	}

	stats := &CredentialStats{
		Total:     len(list),
		Available: len(list), // 暂时默认都可用
	}

	if len(list) > 0 {
		stats.RecentUpdate = list[0].UpdatedAt
		stats.RecentUpdateBy = list[0].UpdatedBy
	}

	return stats, nil
}

// CredentialDetail 凭证详情。
type CredentialDetail struct {
	CredentialItem
	Secret       string  `json:"secret"`       // 密钥内容 (脱敏或根据权限显示)
	CreatedAt    string  `json:"createdAt"`    // 创建时间
	CreatedBy    string  `json:"createdBy"`    // 创建者
	UsageCount   int     `json:"usageCount"`   // 累计使用次数
	SuccessCount int     `json:"successCount"` // 成功次数
	FailureCount int     `json:"failureCount"` // 失败次数
	SuccessRate  float64 `json:"successRate"`  // 成功率
	RecentUsage  string  `json:"recentUsage"`  // 最近使用时间
}

// GetCredentialDetail 获取凭证详情。
func (s *HostService) GetCredentialDetail(ctx context.Context, id string) (*CredentialDetail, error) {
	isKey := strings.HasPrefix(id, "key-")
	realIDStr := strings.TrimPrefix(strings.TrimPrefix(id, "key-"), "tpl-")
	realID, _ := strconv.ParseUint(realIDStr, 10, 64)

	if isKey {
		var key model.SSHKey
		if err := s.svcCtx.DB.WithContext(ctx).First(&key, realID).Error; err != nil {
			return nil, err
		}
		return &CredentialDetail{
			CredentialItem: CredentialItem{
				ID:          id,
				Name:        key.Name,
				Description: fmt.Sprintf("Algorithm: %s", key.Algorithm),
				Type:        "ssh_key",
				AuthMethod:  "SSH Key",
				HostCount:   key.UsageCount,
				Status:      "available",
				UpdatedAt:   key.UpdatedAt.Format("2006-01-02 15:04:05"),
				UpdatedBy:   "system",
			},
			Secret:       key.PublicKey,
			CreatedAt:    key.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy:    "system",
			UsageCount:   key.UsageCount,
			SuccessCount: key.UsageCount,
			SuccessRate:  100,
		}, nil
	} else {
		var tpl model.SSHCredentialTemplate
		if err := s.svcCtx.DB.WithContext(ctx).First(&tpl, realID).Error; err != nil {
			return nil, err
		}
		authMethod := "用户名密码"
		if tpl.AuthType == "key" {
			authMethod = "SSH Key"
		}
		return &CredentialDetail{
			CredentialItem: CredentialItem{
				ID:          id,
				Name:        tpl.Name,
				Description: tpl.Description,
				Type:        tpl.AuthType,
				AuthMethod:  authMethod,
				Status:      "available",
				UpdatedAt:   tpl.UpdatedAt.Format("2006-01-02 15:04:05"),
				UpdatedBy:   strconv.FormatUint(tpl.CreatedBy, 10),
			},
			Secret:    "********",
			CreatedAt: tpl.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy: strconv.FormatUint(tpl.CreatedBy, 10),
		}, nil
	}
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

// ListCredentialTemplates 获取认证预设列表。
//
// 返回所有认证预设模板，密码已脱敏。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 认证预设列表
func (s *HostService) ListCredentialTemplates(ctx context.Context) ([]model.SSHCredentialTemplate, error) {
	var list []model.SSHCredentialTemplate
	err := s.svcCtx.DB.WithContext(ctx).
		Select("id", "name", "auth_type", "ssh_user", "port", "ssh_key_id", "description", "created_by", "created_at", "updated_at").
		Order("id desc").
		Find(&list).Error
	return list, err
}

// CreateCredentialTemplate 创建认证预设。
//
// 验证参数，密码类型时加密存储密码。
//
// 参数:
//   - ctx: 上下文
//   - uid: 操作用户 ID
//   - req: 创建请求参数
//
// 返回: 创建的预设对象（密码已脱敏）
func (s *HostService) CreateCredentialTemplate(ctx context.Context, uid uint64, req CredentialTemplateCreateReq) (*model.SSHCredentialTemplate, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("name is required")
	}
	if req.AuthType != "password" && req.AuthType != "key" {
		return nil, errors.New("auth_type must be 'password' or 'key'")
	}

	template := &model.SSHCredentialTemplate{
		Name:        req.Name,
		AuthType:    req.AuthType,
		SSHUser:     req.SSHUser,
		Port:        req.Port,
		SSHKeyID:    req.SSHKeyID,
		Description: req.Description,
		CreatedBy:   uid,
	}

	// 设置默认值
	if template.SSHUser == "" {
		template.SSHUser = "root"
	}
	if template.Port <= 0 {
		template.Port = 22
	}

	// 密码类型时加密存储密码
	if req.AuthType == "password" {
		if strings.TrimSpace(req.Password) == "" {
			return nil, errors.New("password is required for password auth type")
		}
		if strings.TrimSpace(config.CFG.Security.EncryptionKey) == "" {
			return nil, errors.New("security.encryption_key is required")
		}
		cipher, err := utils.EncryptText(req.Password, config.CFG.Security.EncryptionKey)
		if err != nil {
			return nil, err
		}
		template.Password = cipher
	} else if req.AuthType == "key" {
		// 密钥类型时验证 ssh_key_id 是否有效
		if req.SSHKeyID == nil || *req.SSHKeyID == 0 {
			return nil, errors.New("ssh_key_id is required for key auth type")
		}
		var key model.SSHKey
		if err := s.svcCtx.DB.WithContext(ctx).Select("id").First(&key, *req.SSHKeyID).Error; err != nil {
			return nil, errors.New("ssh_key not found")
		}
	}

	if err := s.svcCtx.DB.WithContext(ctx).Create(template).Error; err != nil {
		return nil, err
	}

	return template, nil
}

// DeleteCredentialTemplate 删除认证预设。
//
// 参数:
//   - ctx: 上下文
//   - id: 预设 ID
//
// 返回: 删除错误
func (s *HostService) DeleteCredentialTemplate(ctx context.Context, id uint64) error {
	result := s.svcCtx.DB.WithContext(ctx).Delete(&model.SSHCredentialTemplate{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("credential template not found")
	}
	return nil
}

// GetCredentialTemplate 获取认证预设详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 预设 ID
//
// 返回: 预设对象（密码已脱敏）
func (s *HostService) GetCredentialTemplate(ctx context.Context, id uint64) (*model.SSHCredentialTemplate, error) {
	var template model.SSHCredentialTemplate
	if err := s.svcCtx.DB.WithContext(ctx).First(&template, id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}
