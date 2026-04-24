package logic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	golangssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newHostLogicTestService(t *testing.T) (*HostService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.HostProbeSession{}, &model.Node{}, &model.TrustedHostKey{}, &model.SSHCredentialTemplate{}); err != nil {
		t.Fatalf("auto migrate host tables: %v", err)
	}

	oldKey := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = "task6-test-encryption-key"
	t.Cleanup(func() {
		config.CFG.Security.EncryptionKey = oldKey
	})

	return NewHostService(&svc.ServiceContext{DB: db}), db
}

func assertCipherRoundTrip(t *testing.T, cipher, plain string) {
	t.Helper()

	if cipher == plain {
		t.Fatalf("expected ciphertext != plaintext, both were %q", plain)
	}
	got, err := utils.DecryptText(cipher, config.CFG.Security.EncryptionKey)
	if err != nil {
		t.Fatalf("decrypt ciphertext: %v", err)
	}
	if got != plain {
		t.Fatalf("decrypted password mismatch, got %q want %q", got, plain)
	}
}

func TestTrustedHostKeyModel_AutoMigrates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&model.TrustedHostKey{}); err != nil {
		t.Fatalf("auto migrate trusted host key: %v", err)
	}

	item := &model.TrustedHostKey{
		HostID:            10,
		Host:              "118.193.38.89",
		Port:              13012,
		Algorithm:         "ssh-ed25519",
		FingerprintSHA256: "SHA256:test-fingerprint",
		PublicKey:         "ssh-ed25519 AAAATEST",
		Status:            model.TrustedHostKeyStatusTrusted,
		CreatedBy:         1,
		ConfirmedAt:       time.Now(),
		LastSeenAt:        time.Now(),
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("create trusted host key: %v", err)
	}
}

func startTestPasswordSSHServer(t *testing.T, username, password string) (string, int, golangssh.PublicKey, func()) {
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
	return "127.0.0.1", tcpAddr.Port, signer.PublicKey(), shutdown
}

func handleTestSSHConn(conn net.Conn, serverConfig *golangssh.ServerConfig) {
	defer conn.Close()

	_, chans, reqs, err := golangssh.NewServerConn(conn, serverConfig)
	if err != nil {
		return
	}
	go golangssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(golangssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go func(ch golangssh.Channel, in <-chan *golangssh.Request) {
			defer ch.Close()
			for req := range in {
				switch req.Type {
				case "exec":
					if req.WantReply {
						_ = req.Reply(true, nil)
					}
					_, _ = ch.Write([]byte("hostname=test-node\nos=TestOS\narch=x86_64\nkernel=6.1\ncpu=2\nmem=2048\ndisk=20\n"))
					_, _ = ch.SendRequest("exit-status", false, golangssh.Marshal(struct{ Status uint32 }{Status: 0}))
					return
				default:
					if req.WantReply {
						_ = req.Reply(false, nil)
					}
				}
			}
		}(channel, requests)
	}
}

func TestProbe_PersistsEncryptedPassword(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const plainPassword = "PlainText-Task6-Password"
	resp, err := hostSvc.Probe(context.Background(), 1001, ProbeReq{
		Name:     "task6-node",
		IP:       "127.0.0.1",
		Port:     1,
		AuthType: "password",
		Username: "root",
		Password: plainPassword,
	})
	if err != nil {
		t.Fatalf("probe returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected probe response")
	}

	var probe model.HostProbeSession
	if err := db.WithContext(context.Background()).First(&probe).Error; err != nil {
		t.Fatalf("load persisted probe session: %v", err)
	}
	assertCipherRoundTrip(t, probe.PasswordCipher, plainPassword)
}

func TestProbe_UsesCredentialTemplatePassword(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const (
		sshUser   = "ops"
		sshPass   = "Template-Password-Task"
		hostName  = "template-probe-node"
		wrongUser = "root"
	)
	host, port, hostKey, shutdown := startTestPasswordSSHServer(t, sshUser, sshPass)
	defer shutdown()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	knownHostsLine := knownhosts.Line([]string{net.JoinHostPort(host, strconv.Itoa(port))}, hostKey)
	if err := os.WriteFile(knownHostsPath, []byte(knownHostsLine+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts file: %v", err)
	}
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	cipher, err := utils.EncryptText(sshPass, config.CFG.Security.EncryptionKey)
	if err != nil {
		t.Fatalf("encrypt template password: %v", err)
	}
	template := &model.SSHCredentialTemplate{
		Name:      "task-template-password",
		AuthType:  "password",
		SSHUser:   sshUser,
		Port:      port,
		Password:  cipher,
		CreatedBy: 1001,
	}
	if err := db.WithContext(context.Background()).Create(template).Error; err != nil {
		t.Fatalf("seed credential template: %v", err)
	}

	resp, err := hostSvc.Probe(context.Background(), 1001, ProbeReq{
		Name:                 hostName,
		IP:                   host,
		Port:                 22,
		AuthType:             "password",
		Username:             wrongUser,
		Password:             "wrong-password",
		CredentialTemplateID: &template.ID,
	})
	if err != nil {
		t.Fatalf("probe returned error: %v", err)
	}
	if resp == nil || !resp.Reachable {
		t.Fatalf("expected probe reachable, got %#v", resp)
	}

	var probe model.HostProbeSession
	if err := db.WithContext(context.Background()).Where("name = ?", hostName).First(&probe).Error; err != nil {
		t.Fatalf("load persisted probe session: %v", err)
	}
	if probe.Username != sshUser {
		t.Fatalf("expected username from template %q, got %q", sshUser, probe.Username)
	}
	if probe.Port != port {
		t.Fatalf("expected port from template %d, got %d", port, probe.Port)
	}
	assertCipherRoundTrip(t, probe.PasswordCipher, sshPass)
}

func TestCreateWithProbe_LegacyRequestEncryptsPassword(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const plainPassword = "Legacy-Create-Password"
	created, err := hostSvc.CreateWithProbe(context.Background(), 1001, false, CreateReq{
		Name:     "legacy-create-node",
		IP:       "10.10.0.2",
		Port:     22,
		Username: "root",
		Password: plainPassword,
	})
	if err != nil {
		t.Fatalf("create from legacy request: %v", err)
	}

	var node model.Node
	if err := db.WithContext(context.Background()).First(&node, created.ID).Error; err != nil {
		t.Fatalf("load created node: %v", err)
	}
	assertCipherRoundTrip(t, node.SSHPassword, plainPassword)
}

func TestCreateWithProbe_ProbeFlowEncryptsNodePassword(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const probeToken = "task6-probe-token"
	const plainPassword = "Probe-Create-Password"
	if err := db.WithContext(context.Background()).Create(&model.HostProbeSession{
		TokenHash:      hashToken(probeToken),
		Name:           "probe-create-node",
		IP:             "10.10.0.3",
		Port:           22,
		AuthType:       "password",
		Username:       "root",
		PasswordCipher: plainPassword,
		Reachable:      true,
		FactsJSON:      `{"hostname":"task6-host"}`,
		WarningsJSON:   `[]`,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
		CreatedBy:      1002,
	}).Error; err != nil {
		t.Fatalf("seed probe session: %v", err)
	}

	created, err := hostSvc.CreateWithProbe(context.Background(), 1002, false, CreateReq{
		ProbeToken: probeToken,
	})
	if err != nil {
		t.Fatalf("create with probe token: %v", err)
	}

	var node model.Node
	if err := db.WithContext(context.Background()).First(&node, created.ID).Error; err != nil {
		t.Fatalf("load created node: %v", err)
	}
	assertCipherRoundTrip(t, node.SSHPassword, plainPassword)
}

func TestCreateWithProbe_ReassignsOnboardingTrustedHostKeys(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const (
		probeToken = "task7-probe-token"
		userID     = uint64(1002)
		host       = "10.10.0.30"
		port       = 22
	)
	if err := db.WithContext(context.Background()).Create(&model.HostProbeSession{
		TokenHash:      hashToken(probeToken),
		Name:           "probe-trust-node",
		IP:             host,
		Port:           port,
		AuthType:       "password",
		Username:       "root",
		PasswordCipher: "Probe-Create-Password",
		Reachable:      true,
		FactsJSON:      `{"hostname":"task7-host"}`,
		WarningsJSON:   `[]`,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
		CreatedBy:      userID,
	}).Error; err != nil {
		t.Fatalf("seed probe session: %v", err)
	}
	seedTrust := &model.TrustedHostKey{
		HostID:            0,
		Host:              host,
		Port:              port,
		Algorithm:         "ssh-ed25519",
		FingerprintSHA256: "SHA256:test-fingerprint",
		PublicKey:         "ssh-ed25519 AAAATEST",
		Status:            model.TrustedHostKeyStatusTrusted,
		CreatedBy:         userID,
		ConfirmedAt:       time.Now(),
		LastSeenAt:        time.Now(),
	}
	if err := db.WithContext(context.Background()).Create(seedTrust).Error; err != nil {
		t.Fatalf("seed onboarding trusted host key: %v", err)
	}

	created, err := hostSvc.CreateWithProbe(context.Background(), userID, false, CreateReq{
		ProbeToken: probeToken,
	})
	if err != nil {
		t.Fatalf("create with probe token: %v", err)
	}

	var trusted model.TrustedHostKey
	if err := db.WithContext(context.Background()).First(&trusted, seedTrust.ID).Error; err != nil {
		t.Fatalf("reload trusted host key: %v", err)
	}
	if trusted.HostID != uint64(created.ID) {
		t.Fatalf("expected trusted host key host_id=%d, got %d", created.ID, trusted.HostID)
	}
}

func TestUpdate_EncryptsSSHPasswordPatch(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	node := &model.Node{
		Name:    "update-node",
		IP:      "10.10.0.4",
		Port:    22,
		SSHUser: "root",
		Status:  "online",
		Source:  "manual_ssh",
	}
	if err := db.WithContext(context.Background()).Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	const newPassword = "Updated-Password-Task6"
	if _, err := hostSvc.Update(context.Background(), uint64(node.ID), map[string]any{
		"ssh_password": newPassword,
	}); err != nil {
		t.Fatalf("update node ssh_password: %v", err)
	}

	var persisted model.Node
	if err := db.WithContext(context.Background()).First(&persisted, node.ID).Error; err != nil {
		t.Fatalf("reload updated node: %v", err)
	}
	assertCipherRoundTrip(t, persisted.SSHPassword, newPassword)
}

func TestUpdateCredentials_EncryptsSSHPassword(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const (
		sshUser      = "ops"
		plainNewPass = "Updated-Credential-Password"
	)
	host, port, hostKey, shutdown := startTestPasswordSSHServer(t, sshUser, plainNewPass)
	defer shutdown()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	knownHostsLine := knownhosts.Line([]string{net.JoinHostPort(host, strconv.Itoa(port))}, hostKey)
	if err := os.WriteFile(knownHostsPath, []byte(knownHostsLine+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts file: %v", err)
	}
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	node := &model.Node{
		Name:        "update-credentials-node",
		IP:          host,
		Port:        port,
		SSHUser:     sshUser,
		SSHPassword: "old-password",
		Status:      "online",
		Source:      "manual_ssh",
	}
	if err := db.WithContext(context.Background()).Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	updated, resp, err := hostSvc.UpdateCredentials(context.Background(), uint64(node.ID), UpdateCredentialsReq{
		AuthType: "password",
		Username: sshUser,
		Password: plainNewPass,
		Port:     port,
	})
	if err != nil {
		t.Fatalf("update credentials: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated node")
	}
	if resp == nil || !resp.Reachable {
		t.Fatalf("expected successful credential probe, got %#v", resp)
	}

	var persisted model.Node
	if err := db.WithContext(context.Background()).First(&persisted, node.ID).Error; err != nil {
		t.Fatalf("reload updated node: %v", err)
	}
	assertCipherRoundTrip(t, persisted.SSHPassword, plainNewPass)
}

func TestUpdateCredentials_ReturnsProbeFailureDetail(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const (
		sshUser         = "ops"
		actualPassword  = "Correct-Password"
		invalidPassword = "Wrong-Password"
	)
	host, port, hostKey, shutdown := startTestPasswordSSHServer(t, sshUser, actualPassword)
	defer shutdown()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	knownHostsLine := knownhosts.Line([]string{net.JoinHostPort(host, strconv.Itoa(port))}, hostKey)
	if err := os.WriteFile(knownHostsPath, []byte(knownHostsLine+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts file: %v", err)
	}
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	node := &model.Node{
		Name:        "probe-failure-detail-node",
		IP:          host,
		Port:        port,
		SSHUser:     sshUser,
		SSHPassword: "old-password",
		Status:      "online",
		Source:      "manual_ssh",
	}
	if err := db.WithContext(context.Background()).Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	updated, resp, err := hostSvc.UpdateCredentials(context.Background(), uint64(node.ID), UpdateCredentialsReq{
		AuthType: "password",
		Username: sshUser,
		Password: invalidPassword,
		Port:     port,
	})
	if err == nil {
		t.Fatal("expected update credentials error")
	}
	if !strings.Contains(err.Error(), "credential probe failed:") {
		t.Fatalf("expected credential probe failure prefix, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected probe response")
	}
	if resp.Reachable {
		t.Fatalf("expected failed probe response, got %#v", resp)
	}
	if resp.ErrorCode != "auth_error" {
		t.Fatalf("expected auth_error, got %q", resp.ErrorCode)
	}
	if strings.TrimSpace(resp.Message) == "" {
		t.Fatal("expected non-empty probe failure message")
	}
	if !strings.Contains(err.Error(), resp.Message) {
		t.Fatalf("expected returned error to include probe detail %q, got %v", resp.Message, err)
	}
	if updated == nil {
		t.Fatal("expected backup node to be returned")
	}
	if updated.SSHPassword != "old-password" {
		t.Fatalf("expected backup node credentials unchanged, got %q", updated.SSHPassword)
	}

	var persisted model.Node
	if err := db.WithContext(context.Background()).First(&persisted, node.ID).Error; err != nil {
		t.Fatalf("reload node: %v", err)
	}
	if persisted.SSHPassword != "old-password" {
		t.Fatalf("expected persisted credentials unchanged, got %q", persisted.SSHPassword)
	}
}

func TestUpdateCredentials_ReturnsStructuredHostKeyFailure(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	const (
		sshUser     = "ops"
		sshPassword = "Updated-Credential-Password"
	)
	host, port, hostKey, shutdown := startTestPasswordSSHServer(t, sshUser, sshPassword)
	defer shutdown()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(knownHostsPath, nil, 0o600); err != nil {
		t.Fatalf("write empty known_hosts file: %v", err)
	}
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	node := &model.Node{
		Name:        "host-key-failure-node",
		IP:          host,
		Port:        port,
		SSHUser:     sshUser,
		SSHPassword: "old-password",
		Status:      "online",
		Source:      "manual_ssh",
	}
	if err := db.WithContext(context.Background()).Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	updated, resp, err := hostSvc.UpdateCredentials(context.Background(), uint64(node.ID), UpdateCredentialsReq{
		AuthType: "password",
		Username: sshUser,
		Password: sshPassword,
		Port:     port,
	})
	if err == nil {
		t.Fatal("expected update credentials error")
	}
	if !strings.Contains(err.Error(), "credential probe failed:") {
		t.Fatalf("expected credential probe failure prefix, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected probe response")
	}
	if resp.Reachable {
		t.Fatalf("expected failed probe response, got %#v", resp)
	}
	if resp.ErrorCode != "ssh_host_key_unknown" {
		t.Fatalf("expected ssh_host_key_unknown, got %q", resp.ErrorCode)
	}
	if resp.HostKey == nil {
		t.Fatal("expected structured host key hint")
	}
	expectedFingerprint := golangssh.FingerprintSHA256(hostKey)
	expectedPublicKey := strings.TrimSpace(string(golangssh.MarshalAuthorizedKey(hostKey)))

	if resp.HostKey.Host != host {
		t.Fatalf("host key hint host mismatch, got %q want %q", resp.HostKey.Host, host)
	}
	if resp.HostKey.Port != port {
		t.Fatalf("host key hint port mismatch, got %d want %d", resp.HostKey.Port, port)
	}
	if resp.HostKey.Algorithm != hostKey.Type() {
		t.Fatalf("host key hint algorithm mismatch, got %q want %q", resp.HostKey.Algorithm, hostKey.Type())
	}
	if resp.HostKey.FingerprintSHA256 != expectedFingerprint {
		t.Fatalf("host key hint fingerprint mismatch, got %q want %q", resp.HostKey.FingerprintSHA256, expectedFingerprint)
	}
	if resp.HostKey.PublicKey != expectedPublicKey {
		t.Fatalf("host key hint public key mismatch, got %q want %q", resp.HostKey.PublicKey, expectedPublicKey)
	}
	if resp.HostKey.KnownHostsPath != knownHostsPath {
		t.Fatalf("host key hint known_hosts mismatch, got %q want %q", resp.HostKey.KnownHostsPath, knownHostsPath)
	}
	if len(resp.HostKey.TrustedFingerprints) != 0 {
		t.Fatalf("expected no trusted fingerprints for unknown host key, got %v", resp.HostKey.TrustedFingerprints)
	}
	if strings.TrimSpace(resp.Message) == "" {
		t.Fatal("expected non-empty probe message")
	}
	if !strings.Contains(err.Error(), expectedFingerprint) {
		t.Fatalf("expected returned error to include fingerprint %q, got %v", expectedFingerprint, err)
	}

	if updated == nil {
		t.Fatal("expected backup node to be returned")
	}
	if updated.SSHPassword != "old-password" {
		t.Fatalf("expected backup node credentials unchanged, got %q", updated.SSHPassword)
	}

	var persisted model.Node
	if err := db.WithContext(context.Background()).First(&persisted, node.ID).Error; err != nil {
		t.Fatalf("reload node: %v", err)
	}
	if persisted.SSHPassword != "old-password" {
		t.Fatalf("expected persisted credentials unchanged, got %q", persisted.SSHPassword)
	}
}
