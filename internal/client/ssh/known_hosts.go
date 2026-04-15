package client

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const knownHostsPathEnvKey = "OPS_KNOWN_HOSTS_PATH"

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
	fingerprint := ssh.FingerprintSHA256(key)

	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) == 0 {
			return fmt.Errorf("ssh host key for %s is unknown (fingerprint %s); add it to %s or set %s: %w", target, fingerprint, knownHostsPath, knownHostsPathEnvKey, err)
		}
		return fmt.Errorf("ssh host key mismatch for %s (presented fingerprint %s); verify and update %s: %w", target, fingerprint, knownHostsPath, err)
	}
	return err
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
