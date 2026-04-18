package alerthealhandler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	alerthealhandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/alertheal"
	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAlertHealJobs_ListByAlertReturnsJobsForAlertFingerprint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAlertHealHandlerTestDB(t)
	adminID := seedAlertHealAdminUser(t, db)
	seedAlertHealAlert(t, db, monitoringmodel.AlertEvent{
		ID:          42,
		Title:       "CPU High",
		Source:      "alertmanager/fp-list",
		Status:      "firing",
		TriggeredAt: time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC),
	})
	seedAlertHealIngestEvent(t, db, aimodel.AIAlertIngestEvent{
		ID:          "evt-list",
		Source:      "receiver-a",
		Protocol:    "alertmanager",
		Fingerprint: "fp-list",
		Status:      "firing",
		DedupeKey:   "receiver-a:fp-list:firing",
		Title:       "CPU High",
		ReceivedAt:  time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC),
	})
	seedAlertHealJob(t, db, aimodel.AIAlertHealJob{
		ID:         "job-list",
		EventID:    "evt-list",
		Scene:      "alert_self_heal",
		Status:     "failed_manual",
		RetryCount: 3,
		MaxRetry:   3,
		LastError:  "boom",
	})

	router := newAlertHealHandlerRouter(db, adminID)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/alert-heal/jobs?alert_id=42", nil)
	router.ServeHTTP(recorder, req)

	resp := decodeAlertHealHandlerResponse(t, recorder)
	if resp.Code != uint32(xcode.Success) {
		t.Fatalf("expected success code %d, got %d body=%s", xcode.Success, resp.Code, recorder.Body.String())
	}

	var payload struct {
		List  []map[string]any `json:"list"`
		Total int64            `json:"total"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		t.Fatalf("decode list payload: %v body=%s", err, recorder.Body.String())
	}
	if payload.Total != 1 || len(payload.List) != 1 {
		t.Fatalf("expected one linked job, total=%d len=%d", payload.Total, len(payload.List))
	}
	if payload.List[0]["id"] != "job-list" {
		t.Fatalf("expected job-list, got %v", payload.List[0]["id"])
	}
	if payload.List[0]["event_fingerprint"] != "fp-list" {
		t.Fatalf("expected fingerprint fp-list, got %v", payload.List[0]["event_fingerprint"])
	}
}

func TestAlertHealJobs_GetJobReturnsJoinedEventFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAlertHealHandlerTestDB(t)
	adminID := seedAlertHealAdminUser(t, db)
	seedAlertHealIngestEvent(t, db, aimodel.AIAlertIngestEvent{
		ID:          "evt-get",
		Source:      "receiver-a",
		Protocol:    "alertmanager",
		Fingerprint: "fp-get",
		Status:      "firing",
		DedupeKey:   "receiver-a:fp-get:firing",
		Title:       "Disk Full",
		Target:      "node-a",
		ReceivedAt:  time.Date(2026, 4, 18, 12, 10, 0, 0, time.UTC),
	})
	seedAlertHealJob(t, db, aimodel.AIAlertHealJob{
		ID:          "job-get",
		EventID:     "evt-get",
		Scene:       "alert_self_heal",
		Status:      "waiting_approval",
		LatestRunID: "run-1",
	})

	router := newAlertHealHandlerRouter(db, adminID)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/alert-heal/jobs/job-get", nil)
	router.ServeHTTP(recorder, req)

	resp := decodeAlertHealHandlerResponse(t, recorder)
	if resp.Code != uint32(xcode.Success) {
		t.Fatalf("expected success code %d, got %d body=%s", xcode.Success, resp.Code, recorder.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		t.Fatalf("decode job payload: %v body=%s", err, recorder.Body.String())
	}
	if payload["event_title"] != "Disk Full" {
		t.Fatalf("expected event_title Disk Full, got %v", payload["event_title"])
	}
	if payload["status"] != "waiting_approval" {
		t.Fatalf("expected waiting_approval status, got %v", payload["status"])
	}
}

func TestAlertHealJobs_RetryResetsTerminalJobToPending(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAlertHealHandlerTestDB(t)
	adminID := seedAlertHealAdminUser(t, db)
	seedAlertHealIngestEvent(t, db, aimodel.AIAlertIngestEvent{
		ID:          "evt-retry",
		Source:      "receiver-a",
		Protocol:    "alertmanager",
		Fingerprint: "fp-retry",
		Status:      "firing",
		DedupeKey:   "receiver-a:fp-retry:firing",
		Title:       "CPU High",
		ReceivedAt:  time.Date(2026, 4, 18, 12, 20, 0, 0, time.UTC),
	})
	seedAlertHealJob(t, db, aimodel.AIAlertHealJob{
		ID:         "job-retry",
		EventID:    "evt-retry",
		Scene:      "alert_self_heal",
		Status:     "failed_manual",
		RetryCount: 3,
		LastError:  "needs retry",
	})

	router := newAlertHealHandlerRouter(db, adminID)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/alert-heal/jobs/job-retry/retry", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	resp := decodeAlertHealHandlerResponse(t, recorder)
	if resp.Code != uint32(xcode.Success) {
		t.Fatalf("expected success code %d, got %d body=%s", xcode.Success, resp.Code, recorder.Body.String())
	}

	var saved aimodel.AIAlertHealJob
	if err := db.Where("id = ?", "job-retry").Take(&saved).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if saved.Status != "pending" {
		t.Fatalf("expected pending status after retry, got %q", saved.Status)
	}
	if saved.RetryCount != 0 {
		t.Fatalf("expected retry_count reset to 0, got %d", saved.RetryCount)
	}
	if saved.LastError != "" {
		t.Fatalf("expected last_error cleared, got %q", saved.LastError)
	}
}

func TestAlertHealJobs_RetryRejectsResolvedAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAlertHealHandlerTestDB(t)
	adminID := seedAlertHealAdminUser(t, db)
	seedAlertHealIngestEvent(t, db, aimodel.AIAlertIngestEvent{
		ID:          "evt-resolved-firing",
		Source:      "receiver-a",
		Protocol:    "alertmanager",
		Fingerprint: "fp-resolved",
		Status:      "firing",
		DedupeKey:   "receiver-a:fp-resolved:firing",
		Title:       "Memory High",
		ReceivedAt:  time.Date(2026, 4, 18, 12, 30, 0, 0, time.UTC),
	})
	seedAlertHealIngestEvent(t, db, aimodel.AIAlertIngestEvent{
		ID:          "evt-resolved",
		Source:      "receiver-a",
		Protocol:    "alertmanager",
		Fingerprint: "fp-resolved",
		Status:      "resolved",
		DedupeKey:   "receiver-a:fp-resolved:resolved",
		Title:       "Memory Recovered",
		ReceivedAt:  time.Date(2026, 4, 18, 12, 31, 0, 0, time.UTC),
	})
	seedAlertHealJob(t, db, aimodel.AIAlertHealJob{
		ID:      "job-resolved",
		EventID: "evt-resolved-firing",
		Scene:   "alert_self_heal",
		Status:  "failed_manual",
	})

	router := newAlertHealHandlerRouter(db, adminID)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/alert-heal/jobs/job-resolved/retry", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	resp := decodeAlertHealHandlerResponse(t, recorder)
	if resp.Code != uint32(xcode.ParamError) {
		t.Fatalf("expected param error code %d, got %d body=%s", xcode.ParamError, resp.Code, recorder.Body.String())
	}
}

type alertHealHandlerResponse struct {
	Code uint32          `json:"code"`
	Data json.RawMessage `json:"data"`
}

func decodeAlertHealHandlerResponse(t *testing.T, recorder *httptest.ResponseRecorder) alertHealHandlerResponse {
	t.Helper()

	var resp alertHealHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return resp
}

func newAlertHealHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&aimodel.AIAlertIngestEvent{},
		&aimodel.AIAlertHealJob{},
		&monitoringmodel.AlertEvent{},
		&usermodel.User{},
		&usermodel.Role{},
		&usermodel.UserRole{},
	); err != nil {
		t.Fatalf("auto migrate alert-heal handler tables: %v", err)
	}
	return db
}

func newAlertHealHandlerRouter(db *gorm.DB, userID uint64) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("uid", userID)
		c.Next()
	})

	h := alerthealhandler.NewHTTPHandler(alerthealhandler.NewService(&svc.ServiceContext{DB: db}))
	router.GET("/ai/alert-heal/jobs", h.ListJobsByAlert)
	router.GET("/ai/alert-heal/jobs/:id", h.GetJob)
	router.POST("/ai/alert-heal/jobs/:id/retry", h.RetryJob)
	return router
}

func seedAlertHealAdminUser(t *testing.T, db *gorm.DB) uint64 {
	t.Helper()

	user := usermodel.User{
		Username:     "opsadmin1",
		PasswordHash: "hash",
		Email:        "ops@example.com",
		Phone:        "13800000000",
		Status:       1,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&user).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	role := usermodel.Role{
		Name:   "Admin",
		Code:   "admin",
		Status: 1,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Create(&usermodel.UserRole{
		UserID: int64(user.ID),
		RoleID: int64(role.ID),
	}).Error; err != nil {
		t.Fatalf("attach admin role: %v", err)
	}
	return uint64(user.ID)
}

func seedAlertHealAlert(t *testing.T, db *gorm.DB, row monitoringmodel.AlertEvent) {
	t.Helper()
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed monitoring alert: %v", err)
	}
}

func seedAlertHealIngestEvent(t *testing.T, db *gorm.DB, row aimodel.AIAlertIngestEvent) {
	t.Helper()
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed ingest event: %v", err)
	}
}

func seedAlertHealJob(t *testing.T, db *gorm.DB, row aimodel.AIAlertHealJob) {
	t.Helper()
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed alert-heal job: %v", err)
	}
}
