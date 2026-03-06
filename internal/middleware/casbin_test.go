package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
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

// ============================================================================
// Extended Tests (T1.2.2)
// ============================================================================

// TestCasbinAuth_BypassForAdminRole tests that admin role also bypasses.
func TestCasbinAuth_BypassForAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)
	_, _ = enforcer.AddGroupingPolicy("2001", "admin")

	r := gin.New()
	r.GET("/check", func(c *gin.Context) {
		c.Set("uid", uint64(2001))
		c.Next()
	}, CasbinAuth(enforcer, "any:permission"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin bypass, got %d", w.Code)
	}
}

// TestCasbinAuth_BypassForChineseSuperAdmin tests Chinese role name.
func TestCasbinAuth_BypassForChineseSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)
	_, _ = enforcer.AddGroupingPolicy("3001", "超级管理员")

	r := gin.New()
	r.GET("/check", func(c *gin.Context) {
		c.Set("uid", uint64(3001))
		c.Next()
	}, CasbinAuth(enforcer, "any:permission"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for 超级管理员 bypass, got %d", w.Code)
	}
}

// TestCasbinAuth_MultiRolePermissionMerge tests user with multiple roles.
func TestCasbinAuth_MultiRolePermissionMerge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)

	// User 4001 has two roles: developer and viewer
	_, _ = enforcer.AddGroupingPolicy("4001", "developer")
	_, _ = enforcer.AddGroupingPolicy("4001", "viewer")

	// developer role has write permission
	_, _ = enforcer.AddPolicy("developer", "resource:write")
	// viewer role has read permission
	_, _ = enforcer.AddPolicy("viewer", "resource:read")

	// Test read permission (from viewer role)
	r := gin.New()
	r.GET("/read", func(c *gin.Context) {
		c.Set("uid", uint64(4001))
		c.Next()
	}, CasbinAuth(enforcer, "resource:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for resource:read via viewer role, got %d", w.Code)
	}

	// Test write permission (from developer role)
	r2 := gin.New()
	r2.GET("/write", func(c *gin.Context) {
		c.Set("uid", uint64(4001))
		c.Next()
	}, CasbinAuth(enforcer, "resource:write"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req2 := httptest.NewRequest(http.MethodGet, "/write", nil)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for resource:write via developer role, got %d", w2.Code)
	}
}

// TestCasbinAuth_RoleInheritance tests role inheritance through grouping.
func TestCasbinAuth_RoleInheritance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)

	// Setup role hierarchy: admin inherits from developer, developer inherits from viewer
	_, _ = enforcer.AddGroupingPolicy("admin", "developer")
	_, _ = enforcer.AddGroupingPolicy("developer", "viewer")

	// viewer has read permission
	_, _ = enforcer.AddPolicy("viewer", "resource:read")

	// User 5001 is admin
	_, _ = enforcer.AddGroupingPolicy("5001", "admin")

	// Test: admin should inherit viewer's read permission through developer
	r := gin.New()
	r.GET("/read", func(c *gin.Context) {
		c.Set("uid", uint64(5001))
		c.Next()
	}, CasbinAuth(enforcer, "resource:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for inherited permission, got %d", w.Code)
	}
}

// TestCasbinAuth_AuditLogOnDeny tests that audit log is set on permission denial.
func TestCasbinAuth_AuditLogOnDeny(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)

	var capturedAudit gin.H
	r := gin.New()
	r.GET("/check",
		func(c *gin.Context) {
			c.Set("uid", uint64(6001))
			c.Next()
			// After middleware chain completes, check audit log
			if v, exists := c.Get("rbac_deny_audit"); exists {
				capturedAudit = v.(gin.H)
			}
		},
		CasbinAuth(enforcer, "sensitive:action"),
		func(c *gin.Context) {
			// This should not be reached
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// Verify audit log was captured after middleware chain
	if capturedAudit == nil {
		t.Fatal("expected audit log to be set on deny")
	}
	if capturedAudit["actor"] != "6001" {
		t.Errorf("expected actor '6001', got '%v'", capturedAudit["actor"])
	}
	if capturedAudit["action"] != "sensitive:action" {
		t.Errorf("expected action 'sensitive:action', got '%v'", capturedAudit["action"])
	}
}

// TestCasbinAuth_ConcurrentPermissionCheck tests thread safety.
func TestCasbinAuth_ConcurrentPermissionCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)
	_, _ = enforcer.AddPolicy("7001", "concurrent:test")

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := gin.New()
			r.GET("/check", func(c *gin.Context) {
				c.Set("uid", uint64(7001))
				c.Next()
			}, CasbinAuth(enforcer, "concurrent:test"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest(http.MethodGet, "/check", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- nil
			}
		}()
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for range errors {
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("concurrent permission checks had %d failures", errorCount)
	}
}

// TestCasbinAuth_NilEnforcer tests handling of nil enforcer.
func TestCasbinAuth_NilEnforcer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/check", func(c *gin.Context) {
		c.Set("uid", uint64(8001))
		c.Next()
	}, CasbinAuth(nil, "any:permission"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for nil enforcer, got %d", w.Code)
	}
}

// TestCasbinAuth_PermissionDeniedForWrongPermission tests explicit denial.
func TestCasbinAuth_PermissionDeniedForWrongPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newTestEnforcer(t)

	// User has read permission but not write
	_, _ = enforcer.AddPolicy("9001", "resource:read")

	r := gin.New()
	r.GET("/write", func(c *gin.Context) {
		c.Set("uid", uint64(9001))
		c.Next()
	}, CasbinAuth(enforcer, "resource:write"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/write", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong permission, got %d", w.Code)
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
