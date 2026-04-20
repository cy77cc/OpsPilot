package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func TestNotificationWS_UnauthenticatedReturns401(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	ws := r.Group("/ws", middleware.JWTAuth())
	ws.GET("/notifications", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ws/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestNotificationWS_RejectsUserIDQueryImpersonation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	ws := r.Group("/ws", middleware.JWTAuth())
	ws.GET("/notifications", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ws/notifications?user_id=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestNotificationWS_CookieAuthIgnoresQueryUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configureJWTForWebSocketTest(t)

	r := gin.New()
	ws := r.Group("/ws", middleware.JWTAuth())
	ws.GET("/notifications", func(c *gin.Context) {
		uid, ok := c.Get("uid")
		if !ok || uid != uint(7) {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"uid":      uid,
			"user_id":  c.Query("user_id"),
			"authed":   true,
			"boundary": "passed",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/ws/notifications?user_id=999", nil)
	req.AddCookie(&http.Cookie{
		Name:  "opspilot_at",
		Value: issueAccessToken(t, 7),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp struct {
		UID      uint   `json:"uid"`
		UserID   string `json:"user_id"`
		Authed   bool   `json:"authed"`
		Boundary string `json:"boundary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if !resp.Authed || resp.Boundary != "passed" {
		t.Fatalf("expected auth boundary pass marker, got %#v", resp)
	}
	if resp.UID != 7 {
		t.Fatalf("expected authenticated uid 7, got %d", resp.UID)
	}
	if resp.UserID != "999" {
		t.Fatalf("expected query user_id to remain 999, got %q", resp.UserID)
	}
}

func TestHandleWebSocket_RejectsUnexpectedUIDType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ws/notifications", func(c *gin.Context) {
		c.Set("uid", "7")
		HandleWebSocket(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/ws/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty unauthorized error payload")
	}
}

func TestIsOriginAllowed(t *testing.T) {
	t.Run("allowed exact match", func(t *testing.T) {
		if !isOriginAllowed("https://example.com", []string{"https://example.com"}) {
			t.Fatal("expected origin to be allowed")
		}
	})

	t.Run("rejects missing origin", func(t *testing.T) {
		if isOriginAllowed("", []string{"https://example.com"}) {
			t.Fatal("expected empty origin to be rejected")
		}
	})

	t.Run("rejects origin not in allowlist", func(t *testing.T) {
		if isOriginAllowed("https://evil.com", []string{"https://example.com"}) {
			t.Fatal("expected origin to be rejected")
		}
	})
}

func TestWebSocketConfig_AllowlistIncludesViteDevOrigins(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	configPath := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "configs", "config.yaml"))
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config template %q: %v", configPath, err)
	}

	var cfg struct {
		Security struct {
			WebSocketAllowOrigins []string `yaml:"websocket_allow_origins"`
		} `yaml:"security"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("failed to parse config template: %v", err)
	}

	requiredOrigins := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}

	for _, required := range requiredOrigins {
		found := false
		for _, configured := range cfg.Security.WebSocketAllowOrigins {
			if strings.EqualFold(strings.TrimSpace(configured), required) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected websocket_allow_origins to include %q, got %v", required, cfg.Security.WebSocketAllowOrigins)
		}
	}
}

func configureJWTForWebSocketTest(t *testing.T) {
	t.Helper()

	oldSecret := config.CFG.JWT.Secret
	oldIssuer := config.CFG.JWT.Issuer
	oldExpire := config.CFG.JWT.Expire

	config.CFG.JWT.Secret = "websocket-auth-test-secret"
	config.CFG.JWT.Issuer = "websocket-auth-test"
	config.CFG.JWT.Expire = time.Hour

	t.Cleanup(func() {
		config.CFG.JWT.Secret = oldSecret
		config.CFG.JWT.Issuer = oldIssuer
		config.CFG.JWT.Expire = oldExpire
	})
}

func issueAccessToken(t *testing.T, uid uint) string {
	t.Helper()
	token, err := utils.GenToken(uid, false)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}
