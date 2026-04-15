package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilesHandler_RequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h, uid := newHostSecurityTestHandler(t)
	cases := []struct {
		name        string
		method      string
		target      string
		params      gin.Params
		contentType string
		body        io.Reader
		handler     func(*gin.Context)
	}{
		{name: "list", method: http.MethodGet, target: "/api/v1/hosts/bad/files", params: gin.Params{{Key: "id", Value: "bad"}}, handler: h.ListFiles},
		{name: "read-content", method: http.MethodGet, target: "/api/v1/hosts/bad/files/content", params: gin.Params{{Key: "id", Value: "bad"}}, handler: h.ReadFileContent},
		{name: "write-content", method: http.MethodPut, target: "/api/v1/hosts/bad/files/content", params: gin.Params{{Key: "id", Value: "bad"}}, contentType: "application/json", body: bytes.NewBufferString(`{}`), handler: h.WriteFileContent},
		{name: "upload", method: http.MethodPost, target: "/api/v1/hosts/bad/files/upload", params: gin.Params{{Key: "id", Value: "bad"}}, handler: h.UploadFile},
		{name: "download", method: http.MethodGet, target: "/api/v1/hosts/bad/files/download", params: gin.Params{{Key: "id", Value: "bad"}}, handler: h.DownloadFile},
		{name: "mkdir", method: http.MethodPost, target: "/api/v1/hosts/bad/files/mkdir", params: gin.Params{{Key: "id", Value: "bad"}}, contentType: "application/json", body: bytes.NewBufferString(`{}`), handler: h.MakeDir},
		{name: "rename", method: http.MethodPost, target: "/api/v1/hosts/bad/files/rename", params: gin.Params{{Key: "id", Value: "bad"}}, contentType: "application/json", body: bytes.NewBufferString(`{}`), handler: h.RenamePath},
		{name: "delete", method: http.MethodDelete, target: "/api/v1/hosts/bad/files", params: gin.Params{{Key: "id", Value: "bad"}}, handler: h.DeletePath},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, recorder := newHostSecurityTestContext(tc.method, tc.target, tc.body, tc.params, uid, tc.contentType)
			tc.handler(ctx)
			assertForbiddenResponse(t, recorder)
		})
	}

	t.Run("file-read-permission-allows-list", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:file:read")
		ctx, recorder := newHostSecurityTestContext(http.MethodGet, "/api/v1/hosts/bad/files", nil, gin.Params{{Key: "id", Value: "bad"}}, uid, "")
		h.ListFiles(ctx)
		assertResponseCode(t, recorder, xcode.ParamError)
	})

	t.Run("file-write-permission-allows-write-content", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:file:write")
		ctx, recorder := newHostSecurityTestContext(http.MethodPut, "/api/v1/hosts/bad/files/content", bytes.NewBufferString(`{}`), gin.Params{{Key: "id", Value: "bad"}}, uid, "application/json")
		h.WriteFileContent(ctx)
		assertResponseCode(t, recorder, xcode.ParamError)
	})

	t.Run("legacy-host-read-does-not-allow-files", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:read")
		ctx, recorder := newHostSecurityTestContext(http.MethodGet, "/api/v1/hosts/bad/files", nil, gin.Params{{Key: "id", Value: "bad"}}, uid, "")
		h.ListFiles(ctx)
		assertForbiddenResponse(t, recorder)
	})

	t.Run("file-wildcard-allows-list", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:file:*")
		ctx, recorder := newHostSecurityTestContext(http.MethodGet, "/api/v1/hosts/bad/files", nil, gin.Params{{Key: "id", Value: "bad"}}, uid, "")
		h.ListFiles(ctx)
		assertResponseCode(t, recorder, xcode.ParamError)
	})
}

func TestTerminalHandler_RequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h, uid := newHostSecurityTestHandler(t)
	cases := []struct {
		name    string
		method  string
		target  string
		params  gin.Params
		handler func(*gin.Context)
	}{
		{
			name:    "create-session",
			method:  http.MethodPost,
			target:  "/api/v1/hosts/bad/terminal/sessions",
			params:  gin.Params{{Key: "id", Value: "bad"}},
			handler: h.CreateTerminalSession,
		},
		{
			name:    "get-session",
			method:  http.MethodGet,
			target:  "/api/v1/hosts/bad/terminal/sessions/s1",
			params:  gin.Params{{Key: "id", Value: "bad"}, {Key: "session_id", Value: "s1"}},
			handler: h.GetTerminalSession,
		},
		{
			name:    "delete-session",
			method:  http.MethodDelete,
			target:  "/api/v1/hosts/bad/terminal/sessions/s1",
			params:  gin.Params{{Key: "id", Value: "bad"}, {Key: "session_id", Value: "s1"}},
			handler: h.DeleteTerminalSession,
		},
		{
			name:    "websocket-session",
			method:  http.MethodGet,
			target:  "/api/v1/hosts/bad/terminal/sessions/s1/ws",
			params:  gin.Params{{Key: "id", Value: "bad"}, {Key: "session_id", Value: "s1"}},
			handler: h.TerminalWebsocket,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, recorder := newHostSecurityTestContext(tc.method, tc.target, nil, tc.params, uid, "")
			tc.handler(ctx)
			assertForbiddenResponse(t, recorder)
		})
	}

	t.Run("terminal-write-permission-allows-create", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:terminal:write")
		ctx, recorder := newHostSecurityTestContext(http.MethodPost, "/api/v1/hosts/bad/terminal/sessions", nil, gin.Params{{Key: "id", Value: "bad"}}, uid, "")
		h.CreateTerminalSession(ctx)
		assertResponseCode(t, recorder, xcode.ParamError)
	})

	t.Run("legacy-host-write-does-not-allow-terminal", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:write")
		ctx, recorder := newHostSecurityTestContext(http.MethodPost, "/api/v1/hosts/bad/terminal/sessions", nil, gin.Params{{Key: "id", Value: "bad"}}, uid, "")
		h.CreateTerminalSession(ctx)
		assertForbiddenResponse(t, recorder)
	})

	t.Run("terminal-wildcard-allows-create", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:terminal:*")
		ctx, recorder := newHostSecurityTestContext(http.MethodPost, "/api/v1/hosts/bad/terminal/sessions", nil, gin.Params{{Key: "id", Value: "bad"}}, uid, "")
		h.CreateTerminalSession(ctx)
		assertResponseCode(t, recorder, xcode.ParamError)
	})
}

func TestCredentialsHandler_RequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h, uid := newHostSecurityTestHandler(t)
	cases := []struct {
		name        string
		method      string
		target      string
		params      gin.Params
		contentType string
		body        io.Reader
		handler     func(*gin.Context)
	}{
		{name: "list-ssh-keys", method: http.MethodGet, target: "/api/v1/credentials/ssh_keys", handler: h.ListSSHKeys},
		{name: "create-ssh-key", method: http.MethodPost, target: "/api/v1/credentials/ssh_keys", contentType: "application/json", body: bytes.NewBufferString(`{}`), handler: h.CreateSSHKey},
		{name: "delete-ssh-key", method: http.MethodDelete, target: "/api/v1/credentials/ssh_keys/bad", params: gin.Params{{Key: "id", Value: "bad"}}, handler: h.DeleteSSHKey},
		{name: "verify-ssh-key", method: http.MethodPost, target: "/api/v1/credentials/ssh_keys/bad/verify", params: gin.Params{{Key: "id", Value: "bad"}}, contentType: "application/json", body: bytes.NewBufferString(`{}`), handler: h.VerifySSHKey},
		{name: "list-templates", method: http.MethodGet, target: "/api/v1/credentials/templates", handler: h.ListCredentialTemplates},
		{name: "create-template", method: http.MethodPost, target: "/api/v1/credentials/templates", contentType: "application/json", body: bytes.NewBufferString(`{}`), handler: h.CreateCredentialTemplate},
		{name: "delete-template", method: http.MethodDelete, target: "/api/v1/credentials/templates/bad", params: gin.Params{{Key: "id", Value: "bad"}}, handler: h.DeleteCredentialTemplate},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, recorder := newHostSecurityTestContext(tc.method, tc.target, tc.body, tc.params, uid, tc.contentType)
			tc.handler(ctx)
			assertForbiddenResponse(t, recorder)
		})
	}

	t.Run("credential-read-permission-allows-list", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:credential:read")
		ctx, recorder := newHostSecurityTestContext(http.MethodGet, "/api/v1/credentials/ssh_keys", nil, nil, uid, "")
		h.ListSSHKeys(ctx)
		assertResponseCode(t, recorder, xcode.Success)
	})

	t.Run("credential-write-permission-allows-delete", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:credential:write")
		ctx, recorder := newHostSecurityTestContext(http.MethodDelete, "/api/v1/credentials/ssh_keys/bad", nil, gin.Params{{Key: "id", Value: "bad"}}, uid, "")
		h.DeleteSSHKey(ctx)
		assertResponseCode(t, recorder, xcode.ParamError)
	})

	t.Run("legacy-host-read-does-not-allow-credentials", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:read")
		ctx, recorder := newHostSecurityTestContext(http.MethodGet, "/api/v1/credentials/ssh_keys", nil, nil, uid, "")
		h.ListSSHKeys(ctx)
		assertForbiddenResponse(t, recorder)
	})

	t.Run("credential-wildcard-allows-list", func(t *testing.T) {
		h, uid := newHostSecurityTestHandler(t)
		grantPermission(t, h, uid, "host:credential:*")
		ctx, recorder := newHostSecurityTestContext(http.MethodGet, "/api/v1/credentials/ssh_keys", nil, nil, uid, "")
		h.ListSSHKeys(ctx)
		assertResponseCode(t, recorder, xcode.Success)
	})
}

func newHostSecurityTestHandler(t *testing.T) (*Handler, uint64) {
	t.Helper()

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
		&hostmodel.SSHKey{},
		&hostmodel.SSHCredentialTemplate{},
	); err != nil {
		t.Fatalf("auto migrate rbac tables: %v", err)
	}

	user := &usermodel.User{
		Username:     "noperm01",
		PasswordHash: "hash",
		Status:       1,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return NewHandler(&svc.ServiceContext{DB: db}), uint64(user.ID)
}

func grantPermission(t *testing.T, h *Handler, uid uint64, permissionCode string) {
	t.Helper()

	roleCode := "role_" + strings.ReplaceAll(permissionCode, ":", "_")
	role := &usermodel.Role{
		Name:   roleCode,
		Code:   roleCode,
		Status: 1,
	}
	if err := h.svcCtx.DB.Create(role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	permission := &usermodel.Permission{
		Name:   permissionCode,
		Code:   permissionCode,
		Type:   1,
		Status: 1,
	}
	if err := h.svcCtx.DB.Create(permission).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}

	if err := h.svcCtx.DB.Create(&usermodel.UserRole{
		UserID: int64(uid),
		RoleID: int64(role.ID),
	}).Error; err != nil {
		t.Fatalf("create user role: %v", err)
	}
	if err := h.svcCtx.DB.Create(&usermodel.RolePermission{
		RoleID:       int64(role.ID),
		PermissionID: int64(permission.ID),
	}).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}
}

func newHostSecurityTestContext(method, target string, body io.Reader, params gin.Params, uid uint64, contentType string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, body)
	ctx.Params = params
	ctx.Set("uid", uid)
	if contentType != "" {
		ctx.Request.Header.Set("Content-Type", contentType)
	}
	return ctx, recorder
}

func assertForbiddenResponse(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	assertResponseCode(t, recorder, xcode.Forbidden)
}

func assertResponseCode(t *testing.T, recorder *httptest.ResponseRecorder, want xcode.Xcode) {
	t.Helper()
	var resp struct {
		Code uint32 `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, recorder.Body.String())
	}
	if resp.Code != uint32(want) {
		t.Fatalf("expected response code %d, got %d body=%s", want, resp.Code, recorder.Body.String())
	}
}
