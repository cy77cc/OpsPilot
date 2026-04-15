package logic

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	golangssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"gorm.io/gorm"

	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
)

const trustKnownHostsPathEnvKey = "OPS_KNOWN_HOSTS_PATH"

// TrustHostKeyReq 描述显式信任主机密钥请求。
type TrustHostKeyReq struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Algorithm         string `json:"algorithm"`
	FingerprintSHA256 string `json:"fingerprint_sha256"`
	PublicKey         string `json:"public_key"`
	ReplaceExisting   bool   `json:"replace_existing"`
}

// TrustHostKey 将指定主机密钥标记为 trusted 并同步 known_hosts。
func (s *HostService) TrustHostKey(ctx context.Context, hostID, operator uint64, req TrustHostKeyReq) (*model.TrustedHostKey, error) {
	normalized, err := normalizeTrustHostKeyReq(req)
	if err != nil {
		return nil, err
	}

	var node model.Node
	if err := s.svcCtx.DB.WithContext(ctx).First(&node, hostID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("host not found")
		}
		return nil, err
	}

	path, err := trustKnownHostsPath()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	item := &model.TrustedHostKey{
		HostID:            hostID,
		Host:              normalized.Host,
		Port:              normalized.Port,
		Algorithm:         normalized.Algorithm,
		FingerprintSHA256: normalized.FingerprintSHA256,
		PublicKey:         normalized.PublicKey,
		Status:            model.TrustedHostKeyStatusTrusted,
		CreatedBy:         operator,
		ConfirmedAt:       now,
		LastSeenAt:        now,
	}

	err = s.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing []model.TrustedHostKey
		if err := tx.
			Where("host_id = ? AND host = ? AND port = ? AND status = ?", hostID, normalized.Host, normalized.Port, model.TrustedHostKeyStatusTrusted).
			Order("id DESC").
			Find(&existing).Error; err != nil {
			return err
		}

		if len(existing) > 0 && !normalized.ReplaceExisting {
			return errors.New("trusted host key already exists; set replace_existing=true to rotate")
		}

		if len(existing) > 0 {
			if err := tx.Model(&model.TrustedHostKey{}).
				Where("host_id = ? AND host = ? AND port = ? AND status = ?", hostID, normalized.Host, normalized.Port, model.TrustedHostKeyStatusTrusted).
				Updates(map[string]any{
					"status":       model.TrustedHostKeyStatusRotated,
					"last_seen_at": now,
					"updated_at":   now,
				}).Error; err != nil {
				return err
			}
		}

		if err := syncKnownHostsEntry(path, normalized.Host, normalized.Port, normalized.PublicKey); err != nil {
			return err
		}

		return tx.Create(item).Error
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// ListTrustedHostKeys 列出指定主机的所有信任密钥记录。
func (s *HostService) ListTrustedHostKeys(ctx context.Context, hostID uint64) ([]model.TrustedHostKey, error) {
	var list []model.TrustedHostKey
	if err := s.svcCtx.DB.WithContext(ctx).
		Where("host_id = ?", hostID).
		Order("confirmed_at DESC, id DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func normalizeTrustHostKeyReq(req TrustHostKeyReq) (*TrustHostKeyReq, error) {
	req.Host = strings.TrimSpace(req.Host)
	req.Algorithm = strings.TrimSpace(req.Algorithm)
	req.FingerprintSHA256 = strings.TrimSpace(req.FingerprintSHA256)
	req.PublicKey = strings.TrimSpace(req.PublicKey)
	if req.Host == "" {
		return nil, errors.New("host is required")
	}
	if req.Port <= 0 {
		return nil, errors.New("port must be greater than 0")
	}
	if req.Algorithm == "" {
		return nil, errors.New("algorithm is required")
	}
	if req.FingerprintSHA256 == "" {
		return nil, errors.New("fingerprint_sha256 is required")
	}
	if req.PublicKey == "" {
		return nil, errors.New("public_key is required")
	}

	pub, _, _, _, err := golangssh.ParseAuthorizedKey([]byte(req.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid public_key: %w", err)
	}
	if pub.Type() != req.Algorithm {
		return nil, errors.New("algorithm does not match public_key")
	}
	if got := golangssh.FingerprintSHA256(pub); got != req.FingerprintSHA256 {
		return nil, errors.New("fingerprint_sha256 does not match public_key")
	}
	return &req, nil
}

func trustKnownHostsPath() (string, error) {
	if customPath := strings.TrimSpace(os.Getenv(trustKnownHostsPathEnvKey)); customPath != "" {
		return customPath, nil
	}
	homePath, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir for known_hosts: %w", err)
	}
	return filepath.Join(homePath, ".ssh", "known_hosts"), nil
}

func syncKnownHostsEntry(path string, host string, port int, publicKey string) error {
	host = strings.TrimSpace(host)
	publicKey = strings.TrimSpace(publicKey)
	if host == "" {
		return errors.New("host is required")
	}
	if port <= 0 {
		return errors.New("port must be greater than 0")
	}
	if publicKey == "" {
		return errors.New("public_key is required")
	}

	key, _, _, _, err := golangssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	bracketedAddress := "[" + host + "]:" + strconv.Itoa(port)
	colonAddress := host + ":" + strconv.Itoa(port)
	newLine := knownhosts.Line([]string{address}, key)

	existing := []byte{}
	if raw, readErr := os.ReadFile(path); readErr == nil {
		existing = raw
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("read known_hosts: %w", readErr)
	}

	filtered := make([]string, 0)
	for _, line := range strings.Split(string(existing), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			filtered = append(filtered, line)
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		hostField := fields[0]
		matchesTarget := false
		for _, pattern := range strings.Split(hostField, ",") {
			candidate := strings.TrimPrefix(strings.TrimSpace(pattern), "!")
			if candidate == address || candidate == bracketedAddress || candidate == colonAddress {
				matchesTarget = true
				break
			}
		}
		if matchesTarget {
			continue
		}
		filtered = append(filtered, line)
	}
	filtered = append(filtered, newLine)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create known_hosts dir: %w", err)
	}
	tmpFile, err := os.CreateTemp(dir, ".known_hosts-*")
	if err != nil {
		return fmt.Errorf("create known_hosts temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		_ = tmpFile.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmpFile.Chmod(0o600); err != nil {
		return fmt.Errorf("chmod known_hosts temp file: %w", err)
	}
	content := strings.Join(filtered, "\n") + "\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		return fmt.Errorf("write known_hosts temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync known_hosts temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close known_hosts temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("atomic replace known_hosts: %w", err)
	}
	cleanup = false
	return nil
}
