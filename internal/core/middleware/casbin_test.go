package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuditAccessDeniedRecordsRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/rbac/roles", nil)
	c.Set("request_id", "req-123")

	auditAccessDenied(c, "42", "rbac.roles.read")

	raw, ok := c.Get("rbac_deny_audit")
	if !ok {
		t.Fatalf("expected rbac_deny_audit to be set")
	}
	auditMap, ok := raw.(gin.H)
	if !ok {
		t.Fatalf("expected gin.H, got %T", raw)
	}
	if got := auditMap["request_id"]; got != "req-123" {
		t.Fatalf("expected request_id req-123, got %v", got)
	}
}
