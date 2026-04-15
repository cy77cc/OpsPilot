package client

import (
	"fmt"
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
	return callback, nil
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
