// Package deployment 提供部署管理服务的工具函数。
//
// 本文件包含预览令牌生成/验证和通用工具函数。
package deployment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/config"
)

// previewTokenClaims 是预览令牌的声明信息。
type previewTokenClaims struct {
	ServiceID   uint   `json:"service_id"`   // 服务 ID
	TargetID    uint   `json:"target_id"`    // 目标 ID
	Env         string `json:"env"`          // 环境
	RuntimeType string `json:"runtime_type"` // 运行时类型
	Strategy    string `json:"strategy"`     // 部署策略
	ContextHash string `json:"context_hash"` // 上下文哈希
	ExpUnix     int64  `json:"exp_unix"`     // 过期时间戳
}

// issuePreviewToken 签发预览令牌。
//
// 参数:
//   - req: 发布预览请求
//   - runtimeType: 运行时类型
//   - env: 环境
//   - manifest: 清单内容
//   - expiresAt: 过期时间
//
// 返回: 令牌字符串和上下文哈希
func issuePreviewToken(req ReleasePreviewReq, runtimeType, env, manifest string, expiresAt time.Time) (string, string) {
	contextHash := buildPreviewContextHash(req, runtimeType, env, manifest)
	claims := previewTokenClaims{
		ServiceID:   req.ServiceID,
		TargetID:    req.TargetID,
		Env:         env,
		RuntimeType: runtimeType,
		Strategy:    defaultIfEmpty(req.Strategy, "rolling"),
		ContextHash: contextHash,
		ExpUnix:     expiresAt.Unix(),
	}
	raw, _ := json.Marshal(claims)
	sig := signPreviewPayload(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + hex.EncodeToString(sig), contextHash
}

// validatePreviewToken 验证预览令牌。
//
// 参数:
//   - req: 发布请求
//   - runtimeType: 运行时类型
//   - env: 环境
//   - manifest: 清单内容
//
// 返回: 上下文哈希、令牌哈希、过期时间、原因码、错误
func validatePreviewToken(req ReleasePreviewReq, runtimeType, env, manifest string) (string, string, *time.Time, string, error) {
	token := strings.TrimSpace(req.PreviewToken)
	if token == "" {
		token = strings.TrimSpace(req.ApprovalToken)
	}
	if token == "" {
		return "", "", nil, "preview_required", fmt.Errorf("preview token required before apply")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", "", nil, "preview_invalid", fmt.Errorf("invalid preview token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", nil, "preview_invalid", fmt.Errorf("invalid preview token payload")
	}
	sig, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", "", nil, "preview_invalid", fmt.Errorf("invalid preview token signature")
	}
	if !hmac.Equal(signPreviewPayload(payload), sig) {
		return "", "", nil, "preview_invalid", fmt.Errorf("preview token signature mismatch")
	}
	var claims previewTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", "", nil, "preview_invalid", fmt.Errorf("invalid preview token claims")
	}
	if time.Now().Unix() > claims.ExpUnix {
		return "", "", nil, "preview_expired", fmt.Errorf("preview token expired")
	}
	expectedHash := buildPreviewContextHash(req, runtimeType, env, manifest)
	if claims.ServiceID != req.ServiceID ||
		claims.TargetID != req.TargetID ||
		claims.Env != env ||
		claims.RuntimeType != runtimeType ||
		claims.Strategy != defaultIfEmpty(req.Strategy, "rolling") ||
		claims.ContextHash != expectedHash {
		return "", "", nil, "preview_mismatch", fmt.Errorf("preview token does not match release context")
	}
	expiresAt := time.Unix(claims.ExpUnix, 0).UTC()
	return expectedHash, sha256Hex(token), &expiresAt, "", nil
}

// buildPreviewContextHash 构建预览上下文哈希。
//
// 参数:
//   - req: 发布预览请求
//   - runtimeType: 运行时类型
//   - env: 环境
//   - manifest: 清单内容
//
// 返回: SHA256 哈希字符串
func buildPreviewContextHash(req ReleasePreviewReq, runtimeType, env, manifest string) string {
	keys := make([]string, 0, len(req.Variables))
	for k := range req.Variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{
		fmt.Sprintf("service=%d", req.ServiceID),
		fmt.Sprintf("target=%d", req.TargetID),
		"runtime=" + runtimeType,
		"env=" + env,
		"strategy=" + defaultIfEmpty(req.Strategy, "rolling"),
		"manifest=" + sha256Hex(manifest),
	}
	for _, k := range keys {
		parts = append(parts, "var:"+k+"="+req.Variables[k])
	}
	return sha256Hex(strings.Join(parts, "|"))
}

// signPreviewPayload 对预览载荷进行签名。
//
// 参数:
//   - payload: 载荷字节
//
// 返回: HMAC-SHA256 签名
func signPreviewPayload(payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(previewTokenSecret()))
	mac.Write(payload)
	return mac.Sum(nil)
}

// previewTokenSecret 获取预览令牌签名密钥。
//
// 返回: JWT 密钥或默认值
func previewTokenSecret() string {
	secret := strings.TrimSpace(config.CFG.JWT.Secret)
	if secret == "" {
		return "deploy-preview-token"
	}
	return secret
}

// sha256Hex 计算字符串的 SHA256 哈希并返回十六进制编码。
//
// 参数:
//   - v: 输入字符串
//
// 返回: 十六进制编码的哈希值
func sha256Hex(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

// defaultIfEmpty 如果字符串为空则返回默认值。
//
// 参数:
//   - v: 输入值
//   - d: 默认值
//
// 返回: 非空值
func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

// defaultInt 如果整数为零或负数则返回默认值。
//
// 参数:
//   - v: 输入值
//   - d: 默认值
//
// 返回: 正整数值
func defaultInt(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

// toJSON 将任意值转换为 JSON 字符串。
//
// 参数:
//   - v: 任意值
//
// 返回: JSON 字符串
func toJSON(v any) string {
	if v == nil {
		return "{}"
	}
	raw, _ := json.Marshal(v)
	return string(raw)
}

// truncateText 截断文本到指定长度。
//
// 参数:
//   - v: 输入字符串
//   - max: 最大长度
//
// 返回: 截断后的字符串
func truncateText(v string, max int) string {
	s := strings.TrimSpace(v)
	if len(s) <= max || max <= 0 {
		return s
	}
	return s[:max]
}
