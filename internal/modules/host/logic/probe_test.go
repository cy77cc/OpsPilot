package logic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	golangssh "golang.org/x/crypto/ssh"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newHostLogicTestService(t *testing.T) (*HostService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.HostProbeSession{}, &model.Node{}); err != nil {
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
	host, port, shutdown := startTestPasswordSSHServer(t, sshUser, plainNewPass)
	defer shutdown()

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
