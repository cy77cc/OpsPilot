package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAdminAIModelRoutes_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	registerAdminAIModelRoutesForTest(v1, newAdminAIModelTestEnforcer(t))

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/v1/admin/ai/models", nil)
		r.ServeHTTP(recorder, req)

		assertAuthFailure(t, recorder, http.StatusUnauthorized, 2003)
	}
}

func TestAdminAIModelRoutes_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	registerAdminAIModelRoutesForTest(v1, newAdminAIModelTestEnforcer(t))

	token := mustAdminAITestToken(t, 101)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/v1/admin/ai/models", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(recorder, req)

		assertAuthFailure(t, recorder, http.StatusForbidden, 2004)
	}
}

func registerAdminAIModelRoutesForTest(v1 *gin.RouterGroup, enforcer *casbin.Enforcer) {
	admin := v1.Group("/admin/ai", middleware.JWTAuth())
	admin.GET("/models", middleware.CasbinAuth(enforcer, "ai:model:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	admin.POST("/models", middleware.CasbinAuth(enforcer, "ai:model:write"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func newAdminAIModelTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()

	m, err := casbinmodel.NewModelFromString(`
[request_definition]
r = sub, obj

[policy_definition]
p = sub, obj

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) || r.sub == p.sub
`)
	if err != nil {
		t.Fatalf("build casbin model: %v", err)
	}

	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("build casbin enforcer: %v", err)
	}
	return enforcer
}

func mustAdminAITestToken(t *testing.T, uid uint) string {
	t.Helper()

	claims := utils.MyClaims{
		Uid: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "ops-pilot-test",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(utils.MySecret)
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return signed
}

func assertAuthFailure(t *testing.T, recorder *httptest.ResponseRecorder, wantHTTPStatus int, wantCode int) {
	t.Helper()

	if recorder.Code != wantHTTPStatus {
		t.Fatalf("expected http status %d, got %d", wantHTTPStatus, recorder.Code)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != wantCode {
		t.Fatalf("expected business code %d, got %d", wantCode, resp.Code)
	}
}
