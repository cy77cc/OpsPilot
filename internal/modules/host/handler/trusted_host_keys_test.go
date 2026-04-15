package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
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
