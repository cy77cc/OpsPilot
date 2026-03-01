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

	"github.com/cy77cc/k8s-manage/internal/config"
)

type previewTokenClaims struct {
	ServiceID   uint   `json:"service_id"`
	TargetID    uint   `json:"target_id"`
	Env         string `json:"env"`
	RuntimeType string `json:"runtime_type"`
	Strategy    string `json:"strategy"`
	ContextHash string `json:"context_hash"`
	ExpUnix     int64  `json:"exp_unix"`
}

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

func signPreviewPayload(payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(previewTokenSecret()))
	mac.Write(payload)
	return mac.Sum(nil)
}

func previewTokenSecret() string {
	secret := strings.TrimSpace(config.CFG.JWT.Secret)
	if secret == "" {
		return "deploy-preview-token"
	}
	return secret
}

func sha256Hex(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func defaultInt(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

func toJSON(v any) string {
	if v == nil {
		return "{}"
	}
	raw, _ := json.Marshal(v)
	return string(raw)
}

func truncateText(v string, max int) string {
	s := strings.TrimSpace(v)
	if len(s) <= max || max <= 0 {
		return s
	}
	return s[:max]
}
