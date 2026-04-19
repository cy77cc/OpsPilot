package handler

import (
	"net/http"
	"net/http/httptest"
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
	db, err := gorm.Open(sqlite.Open("file:channel-test-endpoint?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&usermodel.User{},
		&usermodel.Role{},
		&usermodel.Permission{},
		&usermodel.UserRole{},
		&usermodel.RolePermission{},
		&monitoringmodel.AlertNotificationChannel{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	seedMonitoringWriteUser(t, db, 1001)
	h := NewHandler(&svc.ServiceContext{DB: db})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uid", uint(1001))
		c.Next()
	})
	r.POST("/api/v1/alert-channels/test", h.TestChannel)

	body := `{"provider":"webhook","target":"https://example.com/hook","config_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
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
