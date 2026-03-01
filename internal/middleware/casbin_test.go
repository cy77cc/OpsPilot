package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
)

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	m, err := model.NewModelFromString(`
[request_definition]
r = sub, obj

[role_definition]
g = _, _

[policy_definition]
p = sub, obj

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (r.sub == p.sub || g(r.sub, p.sub)) && r.obj == p.obj
`)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	return e
}

func TestCasbinAuth_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)

	r := gin.New()
	r.GET("/check", func(c *gin.Context) {
		c.Set("uid", uint64(1001))
		c.Next()
	}, CasbinAuth(enforcer, "rbac:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (%s)", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got == "" {
		t.Fatal("expected response body for forbidden")
	}
	if !containsAll(w.Body.String(), []string{"\"code\":2004", "无权限"}) {
		t.Fatalf("unexpected forbidden body: %s", w.Body.String())
	}
}

func TestCasbinAuth_Allow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)
	_, _ = enforcer.AddPolicy("1001", "rbac:read")

	r := gin.New()
	r.GET("/check", func(c *gin.Context) {
		c.Set("uid", uint64(1001))
		c.Next()
	}, CasbinAuth(enforcer, "rbac:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if !containsAll(w.Body.String(), []string{"\"ok\":true"}) {
		t.Fatalf("unexpected allow body: %s", w.Body.String())
	}
}

func TestCasbinAuth_BypassForSuperAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)
	_, _ = enforcer.AddGroupingPolicy("1001", "super-admin")

	r := gin.New()
	r.GET("/check", func(c *gin.Context) {
		c.Set("uid", uint64(1001))
		c.Next()
	}, CasbinAuth(enforcer, "rbac:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if !containsAll(w.Body.String(), []string{"\"ok\":true"}) {
		t.Fatalf("unexpected allow body: %s", w.Body.String())
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})())
}

func TestCasbinAuth_UnauthorizedWhenUIDMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)

	r := gin.New()
	r.GET("/check", CasbinAuth(enforcer, "rbac:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (%s)", w.Code, w.Body.String())
	}
	if !containsAll(w.Body.String(), []string{"\"code\":2003", "未认证"}) {
		t.Fatalf("unexpected unauthorized body: %s", w.Body.String())
	}
}
