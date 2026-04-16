package client

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const knownHostsPathEnvKey = "OPS_KNOWN_HOSTS_PATH"

// HostKeyTrustFailure 描述主机密钥校验失败类型。
type HostKeyTrustFailure string

const (
	HostKeyTrustFailureUnknown  HostKeyTrustFailure = "unknown"
	HostKeyTrustFailureMismatch HostKeyTrustFailure = "mismatch"
	HostKeyTrustFailureRevoked  HostKeyTrustFailure = "revoked"
)

// HostKeyTrustError 携带主机密钥信任失败的结构化上下文。
type HostKeyTrustError struct {
	Failure             HostKeyTrustFailure
	Host                string
	Port                int
	Algorithm           string
	FingerprintSHA256   string
	PublicKey           string
	KnownHostsPath      string
	TrustedFingerprints []string

	message string
	cause   error
}

func (e *HostKeyTrustError) Error() string {
	return e.message
}

func (e *HostKeyTrustError) Unwrap() error {
	return e.cause
}

// AsHostKeyTrustError 尝试从任意错误中提取 HostKeyTrustError。
func AsHostKeyTrustError(err error) (*HostKeyTrustError, bool) {
	var trustErr *HostKeyTrustError
	if !errors.As(err, &trustErr) {
		return nil, false
	}
	return trustErr, true
}

func loadKnownHostsVerifier() (ssh.HostKeyCallback, error) {
	knownHostsPath, err := knownHostsPath()
	if err != nil {
		return nil, err
	}

	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts verifier from %q: %w", knownHostsPath, err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := callback(hostname, remote, key); err != nil {
			return formatHostKeyVerificationError(err, hostname, remote, key, knownHostsPath)
		}
		return nil
	}, nil
}

func formatHostKeyVerificationError(err error, hostname string, remote net.Addr, key ssh.PublicKey, knownHostsPath string) error {
	target := strings.TrimSpace(hostname)
	if target == "" && remote != nil {
		target = remote.String()
	}
	resolvedHost, resolvedPort := resolveTargetAddress(hostname, remote)
	fingerprint := ssh.FingerprintSHA256(key)
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
	algorithm := ""
	if key != nil {
		algorithm = key.Type()
	}

	var revokedErr *knownhosts.RevokedError
	if errors.As(err, &revokedErr) {
		trustedFingerprints := make([]string, 0, 1)
		if revokedErr.Revoked.Key != nil {
			trustedFingerprints = append(trustedFingerprints, ssh.FingerprintSHA256(revokedErr.Revoked.Key))
		}
		resolvedPath := strings.TrimSpace(revokedErr.Revoked.Filename)
		if resolvedPath == "" {
			resolvedPath = knownHostsPath
		}
		message := fmt.Sprintf("ssh host key for %s is revoked (fingerprint %s); remove revoked key from %s", target, fingerprint, resolvedPath)
		return &HostKeyTrustError{
			Failure:             HostKeyTrustFailureRevoked,
			Host:                resolvedHost,
			Port:                resolvedPort,
			Algorithm:           algorithm,
			FingerprintSHA256:   fingerprint,
			PublicKey:           publicKey,
			KnownHostsPath:      resolvedPath,
			TrustedFingerprints: trustedFingerprints,
			message:             message,
			cause:               err,
		}
	}

	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) == 0 {
			message := fmt.Sprintf("ssh host key for %s is unknown (fingerprint %s); add it to %s or set %s", target, fingerprint, knownHostsPath, knownHostsPathEnvKey)
			return &HostKeyTrustError{
				Failure:           HostKeyTrustFailureUnknown,
				Host:              resolvedHost,
				Port:              resolvedPort,
				Algorithm:         algorithm,
				FingerprintSHA256: fingerprint,
				PublicKey:         publicKey,
				KnownHostsPath:    knownHostsPath,
				message:           message,
				cause:             err,
			}
		}

		trustedFingerprints := make([]string, 0, len(keyErr.Want))
		resolvedPath := knownHostsPath
		for _, want := range keyErr.Want {
			if want.Key != nil {
				trustedFingerprints = append(trustedFingerprints, ssh.FingerprintSHA256(want.Key))
			}
			if strings.TrimSpace(resolvedPath) == "" && strings.TrimSpace(want.Filename) != "" {
				resolvedPath = want.Filename
			}
		}
		message := fmt.Sprintf("ssh host key mismatch for %s (presented fingerprint %s); verify and update %s", target, fingerprint, resolvedPath)
		return &HostKeyTrustError{
			Failure:             HostKeyTrustFailureMismatch,
			Host:                resolvedHost,
			Port:                resolvedPort,
			Algorithm:           algorithm,
			FingerprintSHA256:   fingerprint,
			PublicKey:           publicKey,
			KnownHostsPath:      resolvedPath,
			TrustedFingerprints: trustedFingerprints,
			message:             message,
			cause:               err,
		}
	}
	return err
}

func resolveTargetAddress(hostname string, remote net.Addr) (string, int) {
	parse := func(raw string) (string, int, bool) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return "", 0, false
		}
		host, portText, err := net.SplitHostPort(trimmed)
		if err != nil {
			return trimmed, 0, true
		}
		port, err := strconv.Atoi(portText)
		if err != nil {
			return host, 0, true
		}
		return host, port, true
	}

	if host, port, ok := parse(hostname); ok {
		return host, port
	}
	if remote != nil {
		if host, port, ok := parse(remote.String()); ok {
			return host, port
		}
	}
	return "", 0
}

func knownHostsPath() (string, error) {
	if customPath := strings.TrimSpace(os.Getenv(knownHostsPathEnvKey)); customPath != "" {
		return customPath, nil
	}

	homePath, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir for known_hosts: %w", err)
	}
	return filepath.Join(homePath, ".ssh", "known_hosts"), nil
}
