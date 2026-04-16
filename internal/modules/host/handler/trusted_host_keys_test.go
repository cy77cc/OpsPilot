package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	hostlogic "github.com/cy77cc/OpsPilot/internal/modules/host/logic"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	golangssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTrustHostKey_CreatesTrustedEntry(t *testing.T) {
	db, hostSvc := newTrustedHostKeyHandlerTestDeps(t)
	const (
		hostID = uint64(10)
		uid    = uint64(1)
	)
	seedHost(t, db, hostID, "118.193.38.89", 13012)
	grantHostPermission(t, db, uid, "host:trust_host_key")

	algorithm, fingerprint, publicKey := generateAuthorizedKeyMeta(t)
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	body := map[string]any{
		"host":               "118.193.38.89",
		"port":               13012,
		"algorithm":          algorithm,
		"fingerprint_sha256": fingerprint,
		"public_key":         publicKey,
		"replace_existing":   false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	ctx, recorder := newHostMutationTestContext(http.MethodPost, "/api/v1/hosts/10/trust-host-key", bytes.NewReader(payload), gin.Params{{Key: "id", Value: "10"}}, uid)

	h := &Handler{svcCtx: &svc.ServiceContext{DB: db}, hostService: hostSvc}
	h.TrustHostKey(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertHandlerSuccess(t, recorder)

	var item hostmodel.TrustedHostKey
	if err := db.WithContext(context.Background()).
		Where("host_id = ? AND host = ? AND port = ?", hostID, "118.193.38.89", 13012).
		First(&item).Error; err != nil {
		t.Fatalf("query trusted host key: %v", err)
	}
	if item.Status != hostmodel.TrustedHostKeyStatusTrusted {
		t.Fatalf("expected trusted status, got %q", item.Status)
	}
	if item.FingerprintSHA256 != fingerprint {
		t.Fatalf("expected fingerprint %q, got %q", fingerprint, item.FingerprintSHA256)
	}

	rawKnownHosts, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if !strings.Contains(string(rawKnownHosts), publicKey) {
		t.Fatalf("expected known_hosts to include trusted public key, got %s", string(rawKnownHosts))
	}
}

func TestTrustHostKey_RotatesExistingEntry(t *testing.T) {
	db, hostSvc := newTrustedHostKeyHandlerTestDeps(t)
	const (
		hostID = uint64(10)
		uid    = uint64(1)
	)
	seedHost(t, db, hostID, "118.193.38.89", 13012)
	grantHostPermission(t, db, uid, "host:trust_host_key")

	oldAlgorithm, oldFingerprint, oldPublicKey := generateAuthorizedKeyMeta(t)
	newAlgorithm, newFingerprint, newPublicKey := generateAuthorizedKeyMeta(t)
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	knownHostsOldLine, err := knownHostsLine("118.193.38.89", 13012, oldPublicKey)
	if err != nil {
		t.Fatalf("build old known_hosts line: %v", err)
	}
	if err := os.WriteFile(knownHostsPath, []byte(knownHostsOldLine+"\n"), 0o600); err != nil {
		t.Fatalf("write seed known_hosts: %v", err)
	}

	now := time.Now().Add(-time.Hour)
	oldItem := &hostmodel.TrustedHostKey{
		HostID:            hostID,
		Host:              "118.193.38.89",
		Port:              13012,
		Algorithm:         oldAlgorithm,
		FingerprintSHA256: oldFingerprint,
		PublicKey:         oldPublicKey,
		Status:            hostmodel.TrustedHostKeyStatusTrusted,
		CreatedBy:         uid,
		ConfirmedAt:       now,
		LastSeenAt:        now,
	}
	if err := db.WithContext(context.Background()).Create(oldItem).Error; err != nil {
		t.Fatalf("seed trusted key: %v", err)
	}

	body := map[string]any{
		"host":               "118.193.38.89",
		"port":               13012,
		"algorithm":          newAlgorithm,
		"fingerprint_sha256": newFingerprint,
		"public_key":         newPublicKey,
		"replace_existing":   true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	ctx, recorder := newHostMutationTestContext(http.MethodPost, "/api/v1/hosts/10/trust-host-key", bytes.NewReader(payload), gin.Params{{Key: "id", Value: "10"}}, uid)
	h := &Handler{svcCtx: &svc.ServiceContext{DB: db}, hostService: hostSvc}
	h.TrustHostKey(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertHandlerSuccess(t, recorder)

	var reloadedOld hostmodel.TrustedHostKey
	if err := db.WithContext(context.Background()).First(&reloadedOld, oldItem.ID).Error; err != nil {
		t.Fatalf("reload old trusted key: %v", err)
	}
	if reloadedOld.Status != hostmodel.TrustedHostKeyStatusRotated {
		t.Fatalf("expected old entry to be rotated, got %q", reloadedOld.Status)
	}

	var trustedCount int64
	if err := db.WithContext(context.Background()).
		Model(&hostmodel.TrustedHostKey{}).
		Where("host_id = ? AND host = ? AND port = ? AND status = ?", hostID, "118.193.38.89", 13012, hostmodel.TrustedHostKeyStatusTrusted).
		Count(&trustedCount).Error; err != nil {
		t.Fatalf("count trusted entries: %v", err)
	}
	if trustedCount != 1 {
		t.Fatalf("expected exactly 1 trusted entry after rotation, got %d", trustedCount)
	}

	rawKnownHosts, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if strings.Contains(string(rawKnownHosts), oldPublicKey) {
		t.Fatalf("expected old key to be removed from known_hosts, got %s", string(rawKnownHosts))
	}
	if !strings.Contains(string(rawKnownHosts), newPublicKey) {
		t.Fatalf("expected new key to be written into known_hosts, got %s", string(rawKnownHosts))
	}
}

func TestTrustHostKey_AllowsOnboardingWithoutHostRecord(t *testing.T) {
	db, hostSvc := newTrustedHostKeyHandlerTestDeps(t)
	const uid = uint64(1)
	grantHostPermission(t, db, uid, "host:trust_host_key")

	algorithm, fingerprint, publicKey := generateAuthorizedKeyMeta(t)
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	body := map[string]any{
		"host":               "118.193.38.89",
		"port":               13012,
		"algorithm":          algorithm,
		"fingerprint_sha256": fingerprint,
		"public_key":         publicKey,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	ctx, recorder := newHostMutationTestContext(http.MethodPost, "/api/v1/hosts/0/trust-host-key", bytes.NewReader(payload), gin.Params{{Key: "id", Value: "0"}}, uid)

	h := &Handler{svcCtx: &svc.ServiceContext{DB: db}, hostService: hostSvc}
	h.TrustHostKey(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertHandlerSuccess(t, recorder)

	var item hostmodel.TrustedHostKey
	if err := db.WithContext(context.Background()).
		Where("host_id = ? AND host = ? AND port = ?", 0, "118.193.38.89", 13012).
		First(&item).Error; err != nil {
		t.Fatalf("query trusted host key: %v", err)
	}
	if item.HostID != 0 {
		t.Fatalf("expected onboarding trust entry with host_id 0, got %d", item.HostID)
	}
}

func TestHealthCheck_ReturnsHostKeyTrustPayload(t *testing.T) {
	db, hostSvc := newTrustedHostKeyHandlerTestDeps(t)
	const (
		hostID = uint64(21)
		uid    = uint64(1)
		user   = "ops"
		pass   = "secret"
	)
	host, port, _, shutdown := startTestPasswordSSHServer(t, user, pass)
	defer shutdown()

	seedHost(t, db, hostID, host, port)
	if err := db.WithContext(context.Background()).
		Model(&hostmodel.Node{}).
		Where("id = ?", hostmodel.NodeID(hostID)).
		Updates(map[string]any{
			"ssh_user":     user,
			"ssh_password": pass,
		}).Error; err != nil {
		t.Fatalf("seed host credentials: %v", err)
	}
	grantHostPermission(t, db, uid, "host:read")

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(knownHostsPath, nil, 0o600); err != nil {
		t.Fatalf("write empty known_hosts file: %v", err)
	}
	t.Setenv("OPS_KNOWN_HOSTS_PATH", knownHostsPath)

	ctx, recorder := newHostMutationTestContext(http.MethodPost, "/api/v1/hosts/21/ssh/check", nil, gin.Params{{Key: "id", Value: "21"}}, uid)
	h := &Handler{svcCtx: &svc.ServiceContext{DB: db}, hostService: hostSvc}
	h.SSHCheck(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertHandlerSuccess(t, recorder)

	var resp struct {
		Data struct {
			Reachable bool `json:"reachable"`
			HostKey   struct {
				Host              string `json:"host"`
				Port              int    `json:"port"`
				Algorithm         string `json:"algorithm"`
				FingerprintSHA256 string `json:"fingerprint_sha256"`
				PublicKey         string `json:"public_key"`
				KnownHostsPath    string `json:"known_hosts_path"`
			} `json:"host_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if resp.Data.Reachable {
		t.Fatalf("expected unreachable payload, got %+v", resp.Data)
	}
	if strings.TrimSpace(resp.Data.HostKey.FingerprintSHA256) == "" {
		t.Fatalf("expected host_key fingerprint, got %+v", resp.Data.HostKey)
	}
	if strings.TrimSpace(resp.Data.HostKey.PublicKey) == "" {
		t.Fatalf("expected host_key public_key, got %+v", resp.Data.HostKey)
	}
}

func newTrustedHostKeyHandlerTestDeps(t *testing.T) (*gorm.DB, *hostlogic.HostService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&usermodel.User{},
		&usermodel.Role{},
		&usermodel.Permission{},
		&usermodel.UserRole{},
		&usermodel.RolePermission{},
		&hostmodel.Node{},
		&hostmodel.TrustedHostKey{},
	); err != nil {
		t.Fatalf("auto migrate tables: %v", err)
	}

	user := &usermodel.User{
		Username:     "trusted01",
		PasswordHash: "hash",
		Status:       1,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svcCtx := &svc.ServiceContext{DB: db}
	return db, hostlogic.NewHostService(svcCtx)
}

func seedHost(t *testing.T, db *gorm.DB, hostID uint64, ip string, port int) {
	t.Helper()
	host := &hostmodel.Node{
		ID:      hostmodel.NodeID(hostID),
		Name:    "host-" + strconv.FormatUint(hostID, 10),
		IP:      ip,
		Port:    port,
		SSHUser: "root",
		Status:  "offline",
		Source:  "manual_ssh",
	}
	if err := db.WithContext(context.Background()).Create(host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
}

func grantHostPermission(t *testing.T, db *gorm.DB, uid uint64, permissionCode string) {
	t.Helper()
	roleCode := "role_" + strings.ReplaceAll(permissionCode, ":", "_")
	role := &usermodel.Role{
		Name:   roleCode,
		Code:   roleCode,
		Status: 1,
	}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	permission := &usermodel.Permission{
		Name:   permissionCode,
		Code:   permissionCode,
		Type:   1,
		Status: 1,
	}
	if err := db.Create(permission).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if err := db.Create(&usermodel.UserRole{UserID: int64(uid), RoleID: int64(role.ID)}).Error; err != nil {
		t.Fatalf("create user role: %v", err)
	}
	if err := db.Create(&usermodel.RolePermission{RoleID: int64(role.ID), PermissionID: int64(permission.ID)}).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}
}

func newHostMutationTestContext(method, target string, body io.Reader, params gin.Params, uid uint64) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = params
	ctx.Set("uid", uid)
	return ctx, recorder
}

func assertHandlerSuccess(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	var resp struct {
		Code uint32 `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if resp.Code != uint32(xcode.Success) {
		t.Fatalf("expected success code %d, got %d body=%s", xcode.Success, resp.Code, recorder.Body.String())
	}
}

func generateAuthorizedKeyMeta(t *testing.T) (algorithm string, fingerprint string, publicKey string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}
	signer, err := golangssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("build signer: %v", err)
	}
	pub := signer.PublicKey()
	return pub.Type(), golangssh.FingerprintSHA256(pub), strings.TrimSpace(string(golangssh.MarshalAuthorizedKey(pub)))
}

func knownHostsLine(host string, port int, authorizedKey string) (string, error) {
	key, _, _, _, err := golangssh.ParseAuthorizedKey([]byte(authorizedKey))
	if err != nil {
		return "", err
	}
	return knownhosts.Line([]string{net.JoinHostPort(host, strconv.Itoa(port))}, key), nil
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
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
			}
		}(channel, requests)
	}
}
