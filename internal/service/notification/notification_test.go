package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err = db.AutoMigrate(&model.Notification{}, &model.UserNotification{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Next()
	})
	return r
}

func newTestService(db *gorm.DB) *NotificationService {
	return NewNotificationService(&svc.ServiceContext{DB: db})
}

func TestListNotifications(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	notif := model.Notification{
		Type:     "alert",
		Title:    "Test Alert",
		Content:  "Test content",
		Severity: "critical",
		Source:   "test",
		SourceID: "1",
	}
	db.Create(&notif)

	userNotif := model.UserNotification{
		UserID:         1,
		NotificationID: notif.ID,
	}
	db.Create(&userNotif)

	svc := newTestService(db)

	notifications := router.Group("/notifications")
	notifications.GET("", svc.ListNotifications)

	req, _ := http.NewRequest("GET", "/notifications?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	if len(list) != 1 {
		t.Errorf("expected 1 notification, got %d", len(list))
	}
}

func TestUnreadCount(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	notif1 := model.Notification{Type: "alert", Title: "Test Alert 1", Severity: "critical", Source: "test"}
	notif2 := model.Notification{Type: "task", Title: "Test Task 1", Severity: "info", Source: "test"}
	db.Create(&notif1)
	db.Create(&notif2)

	db.Create(&model.UserNotification{UserID: 1, NotificationID: notif1.ID})
	readAt := time.Now()
	userNotif2 := model.UserNotification{
		UserID:         1,
		NotificationID: notif2.ID,
		ReadAt:         &readAt,
	}
	db.Create(&userNotif2)

	svc := newTestService(db)

	notifications := router.Group("/notifications")
	notifications.GET("/unread-count", svc.UnreadCount)

	req, _ := http.NewRequest("GET", "/notifications/unread-count", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	total := data["total"].(float64)
	if total != 1 {
		t.Errorf("expected 1 unread notification, got %f", total)
	}
}

func TestMarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	notif := model.Notification{Type: "alert", Title: "Test Alert", Severity: "critical", Source: "test"}
	db.Create(&notif)

	userNotif := model.UserNotification{
		UserID:         1,
		NotificationID: notif.ID,
	}
	db.Create(&userNotif)

	svc := newTestService(db)

	notifications := router.Group("/notifications")
	notifications.POST("/:id/read", svc.MarkAsRead)

	req, _ := http.NewRequest("POST", "/notifications/1/read", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var updated model.UserNotification
	db.First(&updated, 1)
	if updated.ReadAt == nil {
		t.Error("expected read_at to be set")
	}
}

func TestDismiss(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	notif := model.Notification{Type: "alert", Title: "Test Alert", Severity: "critical", Source: "test"}
	db.Create(&notif)

	userNotif := model.UserNotification{
		UserID:         1,
		NotificationID: notif.ID,
	}
	db.Create(&userNotif)

	svc := newTestService(db)

	notifications := router.Group("/notifications")
	notifications.POST("/:id/dismiss", svc.Dismiss)

	req, _ := http.NewRequest("POST", "/notifications/1/dismiss", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var updated model.UserNotification
	db.First(&updated, 1)
	if updated.DismissedAt == nil {
		t.Error("expected dismissed_at to be set")
	}
}

func TestConfirm(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	notif := model.Notification{
		Type:       "alert",
		Title:      "Test Alert",
		Severity:   "critical",
		Source:     "test",
		SourceID:   "1",
		ActionType: "confirm",
	}
	db.Create(&notif)

	userNotif := model.UserNotification{
		UserID:         1,
		NotificationID: notif.ID,
	}
	db.Create(&userNotif)

	svc := newTestService(db)

	notifications := router.Group("/notifications")
	notifications.POST("/:id/confirm", svc.Confirm)

	req, _ := http.NewRequest("POST", "/notifications/1/confirm", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var updated model.UserNotification
	db.First(&updated, 1)
	if updated.ConfirmedAt == nil {
		t.Error("expected confirmed_at to be set")
	}
	if updated.ReadAt == nil {
		t.Error("expected read_at to be set after confirm")
	}
}

func TestMarkAllAsRead(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	for i := 0; i < 3; i++ {
		notif := model.Notification{Type: "alert", Title: "Test Alert", Severity: "critical", Source: "test"}
		db.Create(&notif)
		db.Create(&model.UserNotification{UserID: 1, NotificationID: notif.ID})
	}

	svc := newTestService(db)

	notifications := router.Group("/notifications")
	notifications.POST("/read-all", svc.MarkAllAsRead)

	req, _ := http.NewRequest("POST", "/notifications/read-all", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var count int64
	db.Model(&model.UserNotification{}).Where("user_id = ? AND read_at IS NULL", 1).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 unread notifications, got %d", count)
	}
}

func TestUnauthorized(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 不设置 user_id

	svc := newTestService(db)

	notifications := r.Group("/notifications")
	notifications.GET("", svc.ListNotifications)

	req, _ := http.NewRequest("GET", "/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (with error code in body), got %d", w.Code)
	}
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["code"] == float64(1000) {
		t.Error("expected non-success code in response body for unauthorized request")
	}
}

func TestListNotificationsAcceptsUIDContextKey(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("uid", uint64(1))
		c.Next()
	})

	notif := model.Notification{
		Type:     "alert",
		Title:    "Test Alert",
		Content:  "Test content",
		Severity: "critical",
		Source:   "test",
		SourceID: "1",
	}
	db.Create(&notif)
	db.Create(&model.UserNotification{
		UserID:         1,
		NotificationID: notif.ID,
	})

	svc := newTestService(db)
	notifications := router.Group("/notifications")
	notifications.GET("", svc.ListNotifications)

	req, _ := http.NewRequest("GET", "/notifications?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}
