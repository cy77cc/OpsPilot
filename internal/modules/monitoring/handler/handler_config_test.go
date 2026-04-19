package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type monitoringConfigTestEnv struct {
	db     *gorm.DB
	router *gin.Engine
}

var (
	monitoringConfigEnvOnce sync.Once
	monitoringConfigEnv     *monitoringConfigTestEnv
	monitoringConfigEnvErr  error

	monitoringConfigOldJWTSecret      string
	monitoringConfigOldJWTIssuer      string
	monitoringConfigOldPrometheusAddr string
	monitoringConfigOldRulesFile      string
	monitoringConfigOldJWTExpire      time.Duration
	monitoringConfigTempDir           string
	monitoringConfigConfigured        bool
)

func TestMain(m *testing.M) {
	code := m.Run()
	restoreMonitoringConfigTestGlobals()
	os.Exit(code)
}

func TestChannelTestEndpoint_Returns200ForValidPayload(t *testing.T) {
	env := setupMonitoringConfigTestEnv(t)
	seedMonitoringWriteUser(t, env.db, 1001)

	body := `{"provider":"webhook","target":"https://example.com/hook","config_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	authorizeMonitoringRequest(t, req, 1001)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteRuleEndpoint_ReturnsConflictWithBlockers(t *testing.T) {
	env := setupMonitoringConfigTestEnv(t)
	seedMonitoringWriteUser(t, env.db, 1001)
	if err := env.db.Create(&monitoringmodel.AlertRule{
		ID: 7, Name: "cpu", Metric: "cpu_usage", Operator: "gt", Threshold: 80, Severity: "warning", Enabled: true, State: "enabled",
	}).Error; err != nil {
		t.Fatalf("seed alert rule: %v", err)
	}
	if err := env.db.Create(&monitoringmodel.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/7", nil)
	authorizeMonitoringRequest(t, req, 1001)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
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
	// Task 3 plan/design contract: delete dependency conflicts use the response envelope with business code 409 plus blockers in data.
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected response code field 409, got %d body=%s", resp.Code, w.Body.String())
	}
	if len(resp.Data.Blockers) == 0 {
		t.Fatalf("expected blockers data, got body=%s", w.Body.String())
	}
}

func TestDeleteChannelEndpoint_Returns404WhenMissing(t *testing.T) {
	env := setupMonitoringConfigTestEnv(t)
	seedMonitoringWriteUser(t, env.db, 1001)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-channels/9999", nil)
	authorizeMonitoringRequest(t, req, 1001)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
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
	env := setupMonitoringConfigTestEnv(t)
	seedMonitoringWriteUser(t, env.db, 1001)
	if err := env.db.Create(&monitoringmodel.AlertNotificationChannel{ID: 1001, Name: "webhook", Type: "webhook", Provider: "webhook", Enabled: true}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if err := env.db.Create(&monitoringmodel.AlertSeverityRoute{ID: 31, Scope: "global", Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true}).Error; err != nil {
		t.Fatalf("seed severity route: %v", err)
	}

	t.Run("POST /api/v1/alert-routing/severity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-routing/severity", strings.NewReader(`{"scope":"global","severity":"critical","channel_ids":[1001],"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		authorizeMonitoringRequest(t, req, 1001)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		assertMonitoringSuccessCode(t, w.Body.Bytes(), "create severity route")
	})

	t.Run("PUT /api/v1/alert-routing/severity/:id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/alert-routing/severity/31", strings.NewReader(`{"scope":"global","severity":"warning","channel_ids":[1001],"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		authorizeMonitoringRequest(t, req, 1001)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		assertMonitoringSuccessCode(t, w.Body.Bytes(), "update severity route")
	})

	t.Run("DELETE /api/v1/alert-routing/severity/:id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-routing/severity/31", nil)
		authorizeMonitoringRequest(t, req, 1001)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		assertMonitoringSuccessCode(t, w.Body.Bytes(), "delete severity route")
	})
}

func TestRuleChannelBindingSingleCRUDEndpoints(t *testing.T) {
	env := setupMonitoringConfigTestEnv(t)
	seedMonitoringWriteUser(t, env.db, 1001)
	projectID := uint(42)
	createProjectID := uint(77)
	if err := env.db.Create(&monitoringmodel.AlertRule{
		ID: 7, Name: "cpu", Metric: "cpu_usage", Operator: "gt", Threshold: 80, Severity: "warning", Enabled: true, State: "enabled",
	}).Error; err != nil {
		t.Fatalf("seed alert rule: %v", err)
	}
	if err := env.db.Create(&monitoringmodel.AlertNotificationChannel{ID: 1001, Name: "webhook", Type: "webhook", Provider: "webhook", Enabled: true}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if err := env.db.Create(&monitoringmodel.AlertNotificationChannel{ID: 1002, Name: "slack", Type: "webhook", Provider: "webhook", Enabled: true}).Error; err != nil {
		t.Fatalf("seed create channel: %v", err)
	}
	if err := env.db.Create(&monitoringmodel.AlertRuleChannelBinding{
		RuleID:    7,
		ChannelID: 1001,
		Priority:  1,
		Enabled:   true,
	}).Error; err != nil {
		t.Fatalf("seed global binding: %v", err)
	}
	if err := env.db.Create(&monitoringmodel.AlertRuleChannelBinding{
		RuleID:    7,
		ChannelID: 1001,
		ProjectID: &projectID,
		Priority:  1,
		Enabled:   true,
	}).Error; err != nil {
		t.Fatalf("seed scoped binding: %v", err)
	}

	t.Run("POST /api/v1/alert-rules/:id/channels", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-rules/7/channels", strings.NewReader(`{"channel_id":1002,"project_id":77,"priority":2,"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		authorizeMonitoringRequest(t, req, 1001)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		assertMonitoringSuccessCode(t, w.Body.Bytes(), "create rule-channel binding")

		var created int64
		if err := env.db.Model(&monitoringmodel.AlertRuleChannelBinding{}).
			Where("rule_id = ? AND channel_id = ? AND project_id = ?", 7, 1002, createProjectID).
			Count(&created).Error; err != nil {
			t.Fatalf("count created binding: %v", err)
		}
		if created != 1 {
			t.Fatalf("expected created binding to exist, got count=%d", created)
		}
	})

	t.Run("PUT /api/v1/alert-rules/:id/channels/:channel_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/7/channels/1001", strings.NewReader(`{"project_id":42,"priority":3,"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		authorizeMonitoringRequest(t, req, 1001)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		assertMonitoringSuccessCode(t, w.Body.Bytes(), "update rule-channel binding")
	})

	t.Run("DELETE /api/v1/alert-rules/:id/channels/:channel_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/7/channels/1001?project_id=42", nil)
		authorizeMonitoringRequest(t, req, 1001)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		assertMonitoringSuccessCode(t, w.Body.Bytes(), "delete rule-channel binding")

		var projectScoped int64
		if err := env.db.Model(&monitoringmodel.AlertRuleChannelBinding{}).
			Where("rule_id = ? AND channel_id = ? AND project_id = ?", 7, 1001, projectID).
			Count(&projectScoped).Error; err != nil {
			t.Fatalf("count project-scoped binding after delete: %v", err)
		}
		if projectScoped != 0 {
			t.Fatalf("expected project-scoped binding to be deleted, got count=%d", projectScoped)
		}

		var globalScoped int64
		if err := env.db.Model(&monitoringmodel.AlertRuleChannelBinding{}).
			Where("rule_id = ? AND channel_id = ? AND project_id IS NULL", 7, 1001).
			Count(&globalScoped).Error; err != nil {
			t.Fatalf("count global binding after scoped delete: %v", err)
		}
		if globalScoped != 1 {
			t.Fatalf("expected global binding to remain after scoped delete, got count=%d", globalScoped)
		}
	})
}

func setupMonitoringConfigTestEnv(t *testing.T) *monitoringConfigTestEnv {
	t.Helper()

	monitoringConfigEnvOnce.Do(func() {
		monitoringConfigEnvErr = initMonitoringConfigTestEnv()
	})
	if monitoringConfigEnvErr != nil {
		t.Fatalf("init monitoring config test env: %v", monitoringConfigEnvErr)
	}
	resetMonitoringConfigTables(t, monitoringConfigEnv.db)
	return monitoringConfigEnv
}

func initMonitoringConfigTestEnv() error {
	gin.SetMode(gin.TestMode)

	if err := configureMonitoringConfigTestGlobals(); err != nil {
		return err
	}

	db, err := gorm.Open(sqlite.Open("file:monitoring-handler-config-shared?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		return err
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
		return err
	}

	r := gin.New()
	v1 := r.Group("/api/v1")
	monitoringapi.RegisterMonitoringHandlers(v1, &svc.ServiceContext{DB: db})

	monitoringConfigEnv = &monitoringConfigTestEnv{
		db:     db,
		router: r,
	}
	return nil
}

func configureMonitoringConfigTestGlobals() error {
	monitoringConfigOldJWTSecret = config.CFG.JWT.Secret
	monitoringConfigOldJWTIssuer = config.CFG.JWT.Issuer
	monitoringConfigOldJWTExpire = config.CFG.JWT.Expire
	monitoringConfigOldPrometheusAddr = config.CFG.Prometheus.Address
	monitoringConfigOldRulesFile = os.Getenv("PROMETHEUS_ALERTING_RULES_FILE")

	monitoringConfigTempDir, monitoringConfigEnvErr = os.MkdirTemp("", "monitoring-handler-config-test-*")
	if monitoringConfigEnvErr != nil {
		return monitoringConfigEnvErr
	}

	config.CFG.JWT.Secret = "monitoring-handler-config-test-secret"
	config.CFG.JWT.Issuer = "monitoring-handler-config-test"
	config.CFG.JWT.Expire = time.Hour
	config.CFG.Prometheus.Address = "http://127.0.0.1:1"
	if err := os.Setenv("PROMETHEUS_ALERTING_RULES_FILE", filepath.Join(monitoringConfigTempDir, "alerting_rules.yml")); err != nil {
		return err
	}

	monitoringConfigConfigured = true
	return nil
}

func restoreMonitoringConfigTestGlobals() {
	if !monitoringConfigConfigured {
		return
	}

	config.CFG.JWT.Secret = monitoringConfigOldJWTSecret
	config.CFG.JWT.Issuer = monitoringConfigOldJWTIssuer
	config.CFG.JWT.Expire = monitoringConfigOldJWTExpire
	config.CFG.Prometheus.Address = monitoringConfigOldPrometheusAddr

	if monitoringConfigOldRulesFile == "" {
		_ = os.Unsetenv("PROMETHEUS_ALERTING_RULES_FILE")
	} else {
		_ = os.Setenv("PROMETHEUS_ALERTING_RULES_FILE", monitoringConfigOldRulesFile)
	}
	if monitoringConfigTempDir != "" {
		_ = os.RemoveAll(monitoringConfigTempDir)
	}
}

func resetMonitoringConfigTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	cleanup := db.Session(&gorm.Session{AllowGlobalUpdate: true})
	for _, row := range []any{
		&usermodel.UserRole{},
		&usermodel.RolePermission{},
		&usermodel.Permission{},
		&usermodel.Role{},
		&usermodel.User{},
		&monitoringmodel.AlertRuleChannelBinding{},
		&monitoringmodel.AlertSeverityRoute{},
		&monitoringmodel.AlertNotificationChannel{},
		&monitoringmodel.AlertRule{},
	} {
		if err := cleanup.Delete(row).Error; err != nil {
			t.Fatalf("reset table for %T: %v", row, err)
		}
	}
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
