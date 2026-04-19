package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
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
	r := setupMonitoringConfigRouter(db, 1001)

	body := `{"provider":"webhook","target":"https://example.com/hook","config_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
	r := setupMonitoringConfigRouter(db, 1001)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/7", nil)
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
	if resp.Code != 409 {
		t.Fatalf("expected response code field 409, got %d body=%s", resp.Code, w.Body.String())
	}
	if len(resp.Data.Blockers) == 0 {
		t.Fatalf("expected blockers data, got body=%s", w.Body.String())
	}
}

func TestDeleteChannelEndpoint_Returns404WhenMissing(t *testing.T) {
	db := setupMonitoringConfigDB(t, "delete-channel-endpoint-missing")
	seedMonitoringWriteUser(t, db, 1001)
	r := setupMonitoringConfigRouter(db, 1001)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-channels/9999", nil)
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
	if resp.Code != 2005 {
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
	r := setupMonitoringConfigRouter(db, 1001)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/alert-routing/severity", strings.NewReader(`{"scope":"global","severity":"critical","channel_ids":[1001],"enabled":true}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("create expected 200, got %d body=%s", createW.Code, createW.Body.String())
	}
	assertMonitoringSuccessCode(t, createW.Body.Bytes(), "create severity route")

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/alert-routing/severity/31", strings.NewReader(`{"scope":"global","severity":"warning","channel_ids":[1001],"enabled":true}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d body=%s", updateW.Code, updateW.Body.String())
	}
	assertMonitoringSuccessCode(t, updateW.Body.Bytes(), "update severity route")

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-routing/severity/31", nil)
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
	r := setupMonitoringConfigRouter(db, 1001)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/alert-rules/7/channels", strings.NewReader(`{"channel_id":1001,"project_id":42,"priority":2,"enabled":true}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("create expected 200, got %d body=%s", createW.Code, createW.Body.String())
	}
	assertMonitoringSuccessCode(t, createW.Body.Bytes(), "create rule-channel binding")

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/7/channels/1001", strings.NewReader(`{"project_id":42,"priority":3,"enabled":true}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d body=%s", updateW.Code, updateW.Body.String())
	}
	assertMonitoringSuccessCode(t, updateW.Body.Bytes(), "update rule-channel binding")

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/7/channels/1001?project_id=42", nil)
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

func setupMonitoringConfigRouter(db *gorm.DB, uid uint) *gin.Engine {
	h := NewHandler(&svc.ServiceContext{DB: db})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uid", uid)
		c.Next()
	})

	// Existing endpoint in this test suite.
	r.POST("/api/v1/alert-channels/test", h.TestChannel)

	// New CRUD endpoints expected by this change set.
	registerHandlerMethodIfExists(r, http.MethodDelete, "/api/v1/alert-rules/:id", h, "DeleteRule")
	registerHandlerMethodIfExists(r, http.MethodDelete, "/api/v1/alert-channels/:id", h, "DeleteChannel")
	registerHandlerMethodIfExists(r, http.MethodPost, "/api/v1/alert-routing/severity", h, "CreateSeverityRoute")
	registerHandlerMethodIfExists(r, http.MethodPut, "/api/v1/alert-routing/severity/:id", h, "UpdateSeverityRouteByID")
	registerHandlerMethodIfExists(r, http.MethodDelete, "/api/v1/alert-routing/severity/:id", h, "DeleteSeverityRoute")
	registerHandlerMethodIfExists(r, http.MethodPost, "/api/v1/alert-rules/:id/channels", h, "CreateRuleChannelBinding")
	registerHandlerMethodIfExists(r, http.MethodPut, "/api/v1/alert-rules/:id/channels/:channel_id", h, "UpdateRuleChannelBinding")
	registerHandlerMethodIfExists(r, http.MethodDelete, "/api/v1/alert-rules/:id/channels/:channel_id", h, "DeleteRuleChannelBinding")

	return r
}

func registerHandlerMethodIfExists(r *gin.Engine, method, route string, h *Handler, methodName string) {
	value := reflect.ValueOf(h).MethodByName(methodName)
	if !value.IsValid() {
		return
	}
	if value.Type().NumIn() != 1 || value.Type().In(0) != reflect.TypeOf(&gin.Context{}) {
		return
	}
	if value.Type().NumOut() != 0 {
		return
	}

	r.Handle(method, route, func(c *gin.Context) {
		value.Call([]reflect.Value{reflect.ValueOf(c)})
	})
}

func assertMonitoringSuccessCode(t *testing.T, body []byte, action string) {
	t.Helper()

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("%s unmarshal response: %v body=%s", action, err, string(body))
	}
	if resp.Code != 1000 {
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
