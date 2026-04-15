package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	"github.com/gin-gonic/gin"
)

func TestJWTAuth_RejectsQueryToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", JWTAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x?token=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp struct {
		Code xcode.Xcode `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Code != xcode.Unauthorized {
		t.Fatalf("expected error code %d, got %d", xcode.Unauthorized, resp.Code)
	}
}

func TestJWTAuth_RejectsValidQueryToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configureJWTForTest(t)

	r := gin.New()
	r.GET("/x", JWTAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	validJWT := issueAccessToken(t, 7)

	req := httptest.NewRequest(http.MethodGet, "/x?token="+validJWT, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp struct {
		Code xcode.Xcode `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.Code != xcode.Unauthorized {
		t.Fatalf("expected error code %d, got %d", xcode.Unauthorized, resp.Code)
	}
}

func TestJWTAuth_AcceptsBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configureJWTForTest(t)

	r := gin.New()
	r.GET("/x", JWTAuth(), func(c *gin.Context) {
		if uid, ok := c.Get("uid"); !ok || uid != uint(42) {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+issueAccessToken(t, 42))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestJWTAuth_AcceptsAccessCookieWhenAuthorizationMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configureJWTForTest(t)

	r := gin.New()
	r.GET("/x", JWTAuth(), func(c *gin.Context) {
		if uid, ok := c.Get("uid"); !ok || uid != uint(88) {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{
		Name:  "opspilot_at",
		Value: issueAccessToken(t, 88),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func configureJWTForTest(t *testing.T) {
	t.Helper()

	oldSecret := config.CFG.JWT.Secret
	oldIssuer := config.CFG.JWT.Issuer
	oldExpire := config.CFG.JWT.Expire

	config.CFG.JWT.Secret = "jwt-auth-test-secret"
	config.CFG.JWT.Issuer = "jwt-auth-test"
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
