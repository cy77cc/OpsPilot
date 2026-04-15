package client

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	golangssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestNewSSHClient_RejectsUnknownHostKey(t *testing.T) {
	host, port, shutdown := startTestPasswordSSHServer(t, "tester", "secret")
	defer shutdown()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(knownHostsPath, nil, 0o600); err != nil {
		t.Fatalf("write empty known_hosts file: %v", err)
	}
	t.Setenv(knownHostsPathEnvKey, knownHostsPath)

	client, err := NewSSHClient("tester", "secret", host, port, "", "")
	if err == nil {
		_ = client.Close()
		t.Fatal("expected unknown host key rejection, got successful connection")
	}

	var keyErr *knownhosts.KeyError
	if !errors.As(err, &keyErr) {
		t.Fatalf("expected known_hosts key error, got: %T: %v", err, err)
	}
	if len(keyErr.Want) != 0 {
		t.Fatalf("expected unknown-host rejection (no trusted keys), got mismatched trusted keys: %v", keyErr.Want)
	}
	if !strings.Contains(err.Error(), "fingerprint ") {
		t.Fatalf("expected actionable fingerprint detail, got: %v", err)
	}
	if !strings.Contains(err.Error(), knownHostsPath) {
		t.Fatalf("expected known_hosts path in error, got: %v", err)
	}
}

func startTestPasswordSSHServer(t *testing.T, username, password string) (string, int, func()) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate ssh host key: %v", err)
	}
	signer, err := golangssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("build ssh signer: %v", err)
	}

	serverConfig := &golangssh.ServerConfig{
		PasswordCallback: func(conn golangssh.ConnMetadata, pass []byte) (*golangssh.Permissions, error) {
			if conn.User() == username && string(pass) == password {
				return nil, nil
			}
			return nil, errors.New("invalid credentials")
		},
	}
	serverConfig.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen test ssh server: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go handleTestSSHConn(conn, serverConfig)
		}
	}()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		t.Fatal("unexpected listener addr type")
	}
	shutdown := func() {
		_ = ln.Close()
		<-done
	}
	return "127.0.0.1", tcpAddr.Port, shutdown
}

func handleTestSSHConn(conn net.Conn, serverConfig *golangssh.ServerConfig) {
	defer conn.Close()

	_, chans, reqs, err := golangssh.NewServerConn(conn, serverConfig)
	if err != nil {
		return
	}
	go golangssh.DiscardRequests(reqs)

	for ch := range chans {
		_ = ch.Reject(golangssh.UnknownChannelType, "unsupported channel type")
	}
}
