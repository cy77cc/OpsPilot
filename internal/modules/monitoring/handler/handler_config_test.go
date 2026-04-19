package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	monitoringapi "github.com/cy77cc/OpsPilot/internal/modules/monitoring/api"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestChannelTestEndpoint_Returns200ForValidPayload(t *testing.T) {
	db := setupMonitoringConfigDB(t, "channel-test-endpoint")
	seedMonitoringWriteUser(t, db, 1001)
	r := setupMonitoringConfigRouter(t, db)

	body := `{"provider":"webhook","target":"https://example.com/hook","config_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	authorizeMonitoringRequest(t, req, 1001)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteRuleEndpoint_ReturnsConflictWithBlockers(t *testing.T) {
	db := setupMonitoringConfigDB(t, "delete-rule-endpoint-conflict")
	seedMonitoringWriteUser(t, db, 1001)
	if err := db.Create(&monitoringmodel.AlertRule{
		ID: 7, Name: "cpu", Metric: "cpu_usage", Operator: "gt", Threshold: 80, Severity: "warning", Enabled: true, State: "enabled",
	}).Error; err != nil {
		t.Fatalf("seed alert rule: %v", err)
	}
	if err := db.Create(&monitoringmodel.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}
	r := setupMonitoringConfigRouter(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/7", nil)
	authorizeMonitoringRequest(t, req, 1001)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Blockers []struct {
				Type  string `json:"type"`
				Count int    `json:"count"`
			} `json:"blockers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v body=%s", err, w.Body.String())
	}
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected response code field 409, got %d body=%s", resp.Code, w.Body.String())
	}
	if len(resp.Data.Blockers) == 0 {
		t.Fatalf("expected blockers data, got body=%s", w.Body.String())
	}
}

func TestDeleteChannelEndpoint_Returns404WhenMissing(t *testing.T) {
	db := setupMonitoringConfigDB(t, "delete-channel-endpoint-missing")
	seedMonitoringWriteUser(t, db, 1001)
	r := setupMonitoringConfigRouter(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-channels/9999", nil)
	authorizeMonitoringRequest(t, req, 1001)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v body=%s", err, w.Body.String())
	}
	if resp.Code != int(xcode.NotFound) {
		t.Fatalf("expected not-found code field 2005, got %d body=%s", resp.Code, w.Body.String())
	}
}

func TestSeverityRouteSingleCRUDEndpoints(t *testing.T) {
	db := setupMonitoringConfigDB(t, "severity-route-single-crud")
	seedMonitoringWriteUser(t, db, 1001)
	if err := db.Create(&monitoringmodel.AlertNotificationChannel{ID: 1001, Name: "webhook", Type: "webhook", Provider: "webhook", Enabled: true}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if err := db.Create(&monitoringmodel.AlertSeverityRoute{ID: 31, Scope: "global", Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true}).Error; err != nil {
		t.Fatalf("seed severity route: %v", err)
	}
	r := setupMonitoringConfigRouter(t, db)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/alert-routing/severity", strings.NewReader(`{"scope":"global","severity":"critical","channel_ids":[1001],"enabled":true}`))
	createReq.Header.Set("Content-Type", "application/json")
	authorizeMonitoringRequest(t, createReq, 1001)
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("create expected 200, got %d body=%s", createW.Code, createW.Body.String())
	}
	assertMonitoringSuccessCode(t, createW.Body.Bytes(), "create severity route")

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/alert-routing/severity/31", strings.NewReader(`{"scope":"global","severity":"warning","channel_ids":[1001],"enabled":true}`))
	updateReq.Header.Set("Content-Type", "application/json")
	authorizeMonitoringRequest(t, updateReq, 1001)
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d body=%s", updateW.Code, updateW.Body.String())
	}
	assertMonitoringSuccessCode(t, updateW.Body.Bytes(), "update severity route")

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-routing/severity/31", nil)
	authorizeMonitoringRequest(t, deleteReq, 1001)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Fatalf("delete expected 200, got %d body=%s", deleteW.Code, deleteW.Body.String())
	}
	assertMonitoringSuccessCode(t, deleteW.Body.Bytes(), "delete severity route")
}

func TestRuleChannelBindingSingleCRUDEndpoints(t *testing.T) {
	db := setupMonitoringConfigDB(t, "rule-channel-binding-single-crud")
	seedMonitoringWriteUser(t, db, 1001)
	projectID := uint(42)
	if err := db.Create(&monitoringmodel.AlertRule{
		ID: 7, Name: "cpu", Metric: "cpu_usage", Operator: "gt", Threshold: 80, Severity: "warning", Enabled: true, State: "enabled",
	}).Error; err != nil {
		t.Fatalf("seed alert rule: %v", err)
	}
	if err := db.Create(&monitoringmodel.AlertNotificationChannel{ID: 1001, Name: "webhook", Type: "webhook", Provider: "webhook", Enabled: true}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if err := db.Create(&monitoringmodel.AlertRuleChannelBinding{
		RuleID:    7,
		ChannelID: 1001,
		ProjectID: &projectID,
		Priority:  1,
		Enabled:   true,
	}).Error; err != nil {
		t.Fatalf("seed scoped binding: %v", err)
	}
	r := setupMonitoringConfigRouter(t, db)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/alert-rules/7/channels", strings.NewReader(`{"channel_id":1001,"project_id":42,"priority":2,"enabled":true}`))
	createReq.Header.Set("Content-Type", "application/json")
	authorizeMonitoringRequest(t, createReq, 1001)
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("create expected 200, got %d body=%s", createW.Code, createW.Body.String())
	}
	assertMonitoringSuccessCode(t, createW.Body.Bytes(), "create rule-channel binding")

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/7/channels/1001", strings.NewReader(`{"project_id":42,"priority":3,"enabled":true}`))
	updateReq.Header.Set("Content-Type", "application/json")
	authorizeMonitoringRequest(t, updateReq, 1001)
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d body=%s", updateW.Code, updateW.Body.String())
	}
	assertMonitoringSuccessCode(t, updateW.Body.Bytes(), "update rule-channel binding")

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/7/channels/1001?project_id=42", nil)
	authorizeMonitoringRequest(t, deleteReq, 1001)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Fatalf("delete expected 200, got %d body=%s", deleteW.Code, deleteW.Body.String())
	}
	assertMonitoringSuccessCode(t, deleteW.Body.Bytes(), "delete rule-channel binding")
}

func setupMonitoringConfigDB(t *testing.T, dbName string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&usermodel.User{},
		&usermodel.Role{},
		&usermodel.Permission{},
		&usermodel.UserRole{},
		&usermodel.RolePermission{},
		&monitoringmodel.AlertRule{},
		&monitoringmodel.AlertNotificationChannel{},
		&monitoringmodel.AlertRuleChannelBinding{},
		&monitoringmodel.AlertSeverityRoute{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func setupMonitoringConfigRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	configureJWTForMonitoringHandlerTests(t)
	configurePrometheusForMonitoringHandlerTests(t)

	r := gin.New()
	v1 := r.Group("/api/v1")
	monitoringapi.RegisterMonitoringHandlers(v1, &svc.ServiceContext{DB: db})
	return r
}

func configurePrometheusForMonitoringHandlerTests(t *testing.T) {
	t.Helper()

	reloadStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(reloadStub.Close)

	oldAddress := config.CFG.Prometheus.Address
	config.CFG.Prometheus.Address = reloadStub.URL
	t.Cleanup(func() {
		config.CFG.Prometheus.Address = oldAddress
	})

	t.Setenv("PROMETHEUS_ALERTING_RULES_FILE", filepath.Join(t.TempDir(), "alerting_rules.yml"))
}

func configureJWTForMonitoringHandlerTests(t *testing.T) {
	t.Helper()

	oldSecret := config.CFG.JWT.Secret
	oldIssuer := config.CFG.JWT.Issuer
	oldExpire := config.CFG.JWT.Expire

	config.CFG.JWT.Secret = "monitoring-handler-config-test-secret"
	config.CFG.JWT.Issuer = "monitoring-handler-config-test"
	config.CFG.JWT.Expire = time.Hour

	t.Cleanup(func() {
		config.CFG.JWT.Secret = oldSecret
		config.CFG.JWT.Issuer = oldIssuer
		config.CFG.JWT.Expire = oldExpire
	})
}

func authorizeMonitoringRequest(t *testing.T, req *http.Request, uid uint) {
	t.Helper()

	token, err := utils.GenToken(uid, false)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

func assertMonitoringSuccessCode(t *testing.T, body []byte, action string) {
	t.Helper()

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("%s unmarshal response: %v body=%s", action, err, string(body))
	}
	if resp.Code != int(xcode.Success) {
		t.Fatalf("%s expected response code field 1000, got %d body=%s", action, resp.Code, string(body))
	}
}

func seedMonitoringWriteUser(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()

	user := usermodel.User{
		ID:           usermodel.UserID(userID),
		Username:     "opsuser1",
		PasswordHash: "hash",
		Status:       1,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	role := usermodel.Role{Name: "Ops", Code: "ops", Status: 1}
	perm := usermodel.Permission{Name: "MonitoringWrite", Code: "monitoring:write", Status: 1}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&perm).Error; err != nil {
		t.Fatalf("seed permission: %v", err)
	}
	if err := db.Create(&usermodel.RolePermission{RoleID: int64(role.ID), PermissionID: int64(perm.ID)}).Error; err != nil {
		t.Fatalf("seed role permission: %v", err)
	}
	if err := db.Create(&usermodel.UserRole{UserID: int64(userID), RoleID: int64(role.ID)}).Error; err != nil {
		t.Fatalf("seed user role: %v", err)
	}

	// Ensure UID conversion path in auth helpers remains covered by this test setup.
	if got := httpx.ToUint64(uint(userID)); got == 0 {
		t.Fatalf("unexpected uid conversion result: %d", got)
	}
}
