package logic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/google/uuid"
)

// Probe 执行 SSH 连接探测。
//
// 尝试通过 SSH 连接目标主机，收集系统信息（主机名、操作系统、架构、内核、
// CPU、内存、磁盘），生成一次性探测令牌用于后续创建主机。
//
// 探测结果会存储到 host_probe_sessions 表，令牌有效期为 10 分钟。
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户 ID
//   - req: 探测请求参数
//
// 返回: 探测响应，包含系统信息和探测令牌
func (s *HostService) Probe(ctx context.Context, userID uint64, req ProbeReq) (*ProbeResp, error) {
	normalizeProbeReq(&req)
	if err := validateProbeReq(req); err != nil {
		return &ProbeResp{Reachable: false, ErrorCode: "validation_error", Message: err.Error()}, nil
	}

	start := time.Now()
	facts, warnings, privateKey, err := s.probeFacts(ctx, req)
	latency := time.Since(start).Milliseconds()
	resp := &ProbeResp{
		Reachable: err == nil,
		LatencyMS: latency,
		Facts:     facts,
		Warnings:  warnings,
		ExpiresAt: time.Now().Add(ProbeTokenTTL),
	}
	if err != nil {
		resp.ErrorCode, resp.Message = mapProbeError(err)
	}

	token := uuid.NewString()
	hash := hashToken(token)
	factsJSON, _ := json.Marshal(resp.Facts)
	warningsJSON, _ := json.Marshal(resp.Warnings)
	probe := model.HostProbeSession{
		TokenHash:      hash,
		Name:           req.Name,
		IP:             req.IP,
		Port:           req.Port,
		AuthType:       req.AuthType,
		Username:       req.Username,
		PasswordCipher: req.Password,
		Reachable:      resp.Reachable,
		LatencyMS:      resp.LatencyMS,
		FactsJSON:      string(factsJSON),
		WarningsJSON:   string(warningsJSON),
		ExpiresAt:      resp.ExpiresAt,
		CreatedBy:      userID,
	}
	if req.SSHKeyID != nil {
		probe.SSHKeyID = req.SSHKeyID
	}
	if strings.TrimSpace(privateKey) == "" && req.AuthType == "key" {
		resp.Warnings = append(resp.Warnings, "ssh key not found, key auth may fail")
	}
	if err := s.svcCtx.DB.WithContext(ctx).Create(&probe).Error; err != nil {
		return nil, err
	}

	resp.ProbeToken = token
	return resp, nil
}

// hashToken 对探测令牌进行 SHA256 哈希。
//
// 用于安全存储令牌，避免明文存储。
//
// 参数:
//   - token: 原始令牌
//
// 返回: 哈希后的令牌字符串
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// mapProbeError 将 SSH 错误映射为标准错误码和消息。
//
// 参数:
//   - err: 原始错误
//
// 返回:
//   - errorCode: 标准错误码 (timeout_error/auth_error/validation_error/connect_error)
//   - message: 错误消息
func mapProbeError(err error) (string, string) {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return "timeout_error", err.Error()
	case strings.Contains(msg, "authentication") || strings.Contains(msg, "unable to authenticate"):
		return "auth_error", err.Error()
	case strings.Contains(msg, "no ssh auth method"):
		return "validation_error", err.Error()
	default:
		return "connect_error", err.Error()
	}
}

// loadPrivateKey 加载 SSH 密钥的私钥内容。
//
// 从数据库加载指定 SSH 密钥，处理加密存储的私钥解密。
//
// 参数:
//   - ctx: 上下文
//   - sshKeyID: SSH 密钥 ID
//
// 返回:
//   - privateKey: 私钥内容
//   - passphrase: 私钥密码
//   - error: 错误信息
func (s *HostService) loadPrivateKey(ctx context.Context, sshKeyID *uint64) (string, string, error) {
	if sshKeyID == nil {
		return "", "", nil
	}
	var key model.SSHKey
	if err := s.svcCtx.DB.WithContext(ctx).Select("id", "private_key", "passphrase", "encrypted").Where("id = ?", *sshKeyID).First(&key).Error; err != nil {
		return "", "", err
	}
	passphrase := strings.TrimSpace(key.Passphrase)
	if key.Encrypted {
		privateKey, err := utils.DecryptText(key.PrivateKey, config.CFG.Security.EncryptionKey)
		if err != nil {
			return "", "", err
		}
		return privateKey, passphrase, nil
	}
	return key.PrivateKey, passphrase, nil
}

// probeFacts 通过 SSH 连接收集主机系统信息。
//
// 执行远程命令收集主机名、操作系统、架构、内核版本、CPU 核心数、
// 内存大小、磁盘大小等信息。
//
// 参数:
//   - ctx: 上下文
//   - req: 探测请求参数
//
// 返回:
//   - ProbeFacts: 系统信息
//   - []string: 警告信息列表
//   - string: 使用的私钥（用于后续创建主机）
//   - error: 错误信息
func (s *HostService) probeFacts(ctx context.Context, req ProbeReq) (ProbeFacts, []string, string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()

	privateKey, passphrase, err := s.loadPrivateKey(probeCtx, req.SSHKeyID)
	if err != nil {
		return ProbeFacts{}, nil, "", err
	}
	password := req.Password
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(req.Username, password, req.IP, req.Port, privateKey, passphrase)
	if err != nil {
		return ProbeFacts{}, nil, privateKey, err
	}
	defer cli.Close()

	cmd := `
echo "hostname=$(hostname)"
echo "os=$(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2 | tr -d '"')"
echo "arch=$(uname -m)"
echo "kernel=$(uname -r)"
echo "cpu=$(nproc)"
echo "mem=$(free -m | awk '/Mem:/ {print $2}')"
echo "disk=$(df -BG / | tail -1 | awk '{print $2}' | tr -d G)"
`
	out, err := sshclient.RunCommand(cli, cmd)
	if err != nil {
		return ProbeFacts{}, nil, privateKey, err
	}

	facts := ProbeFacts{}
	warnings := make([]string, 0)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "hostname":
			facts.Hostname = val
		case "os":
			facts.OS = val
		case "arch":
			facts.Arch = val
		case "kernel":
			facts.Kernel = val
		case "cpu":
			_, _ = fmt.Sscanf(val, "%d", &facts.CPUCores)
		case "mem":
			_, _ = fmt.Sscanf(val, "%d", &facts.MemoryMB)
		case "disk":
			_, _ = fmt.Sscanf(val, "%d", &facts.DiskGB)
		}
	}
	if facts.Hostname == "" {
		warnings = append(warnings, "hostname not detected")
	}
	return facts, warnings, privateKey, nil
}

// validateProbeReq 验证探测请求参数。
//
// 检查必需字段是否填写，认证类型是否有效。
//
// 参数:
//   - req: 探测请求参数
//
// 返回: 验证错误
func validateProbeReq(req ProbeReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.IP) == "" {
		return errors.New("ip is required")
	}
	if strings.TrimSpace(req.Username) == "" {
		return errors.New("username is required")
	}
	if req.AuthType != "password" && req.AuthType != "key" {
		return errors.New("auth_type must be password or key")
	}
	if req.AuthType == "password" && strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required when auth_type=password")
	}
	if req.AuthType == "key" && req.SSHKeyID == nil {
		return errors.New("ssh_key_id is required when auth_type=key")
	}
	return nil
}

// normalizeProbeReq 规范化探测请求参数。
//
// 设置默认端口和用户名。
//
// 参数:
//   - req: 探测请求参数指针
func normalizeProbeReq(req *ProbeReq) {
	if req.Port <= 0 {
		req.Port = DefaultSSHPort
	}
	if req.Username == "" {
		req.Username = "root"
	}
}

// normalizeCredentialReq 规范化凭证更新请求参数。
//
// 设置默认端口和认证类型。
//
// 参数:
//   - req: 凭证更新请求参数指针
func normalizeCredentialReq(req *UpdateCredentialsReq) {
	if req.Port <= 0 {
		req.Port = DefaultSSHPort
	}
	if req.AuthType == "" {
		req.AuthType = "password"
	}
}
