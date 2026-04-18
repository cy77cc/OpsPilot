package approvalhandler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	approvalhandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/approval"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPendingApprovalsGlobal_RequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newPendingApprovalsGlobalTestDB(t)
	viewer := seedPendingApprovalsGlobalUser(t, db, "viewer01")
	router := newPendingApprovalsGlobalRouter(t, db, uint64(viewer.ID))

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/approvals/pending/global", nil)
	router.ServeHTTP(recorder, req)

	resp := decodeApprovalSubmitResponse(t, recorder)
	if resp.Code != uint32(xcode.Forbidden) {
		t.Fatalf("expected forbidden code %d, got %d body=%s", xcode.Forbidden, resp.Code, recorder.Body.String())
	}
}

func TestPendingApprovalsGlobal_AllowsAdminFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newPendingApprovalsGlobalTestDB(t)
	adminUser := seedPendingApprovalsGlobalUser(t, db, "opsadmin")
	adminRole := seedPendingApprovalsGlobalRole(t, db, "admin")
	attachPendingApprovalsGlobalRole(t, db, adminUser.ID, adminRole.ID)
	router := newPendingApprovalsGlobalRouter(t, db, uint64(adminUser.ID))

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/approvals/pending/global", nil)
	router.ServeHTTP(recorder, req)

	resp := decodeApprovalSubmitResponse(t, recorder)
	if resp.Code != uint32(xcode.Success) {
		t.Fatalf("expected admin fallback success code %d, got %d body=%s", xcode.Success, resp.Code, recorder.Body.String())
	}
}

func TestPendingApprovalsGlobal_ReturnsPendingAcrossUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newPendingApprovalsGlobalTestDB(t)
	viewer := seedPendingApprovalsGlobalUser(t, db, "viewer02")
	grantPendingApprovalsGlobalPermission(t, db, viewer.ID, "ai:approval:read")

	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	seedPendingApprovalTask(t, db, pendingApprovalTaskSeed{
		approvalID: "approval-u1-pending",
		userID:     101,
		status:     "pending",
		createdAt:  now.Add(2 * time.Minute),
	})
	seedPendingApprovalTask(t, db, pendingApprovalTaskSeed{
		approvalID: "approval-u2-pending",
		userID:     202,
		status:     "pending",
		createdAt:  now.Add(1 * time.Minute),
	})
	seedPendingApprovalTask(t, db, pendingApprovalTaskSeed{
		approvalID: "approval-u3-approved",
		userID:     303,
		status:     "approved",
		createdAt:  now,
	})

	router := newPendingApprovalsGlobalRouter(t, db, uint64(viewer.ID))
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/approvals/pending/global?page=1&page_size=2", nil)
	router.ServeHTTP(recorder, req)

	resp := decodeApprovalSubmitResponse(t, recorder)
	if resp.Code != uint32(xcode.Success) {
		t.Fatalf("expected success code %d, got %d body=%s", xcode.Success, resp.Code, recorder.Body.String())
	}

	var tasks []ai.AIApprovalTask
	if err := json.Unmarshal(resp.Data, &tasks); err != nil {
		t.Fatalf("decode data as approval tasks: %v payload=%s", err, string(resp.Data))
	}
	if len(tasks) != 2 {
		t.Fatalf("expected two pending approvals, got %d", len(tasks))
	}
	if tasks[0].ApprovalID != "approval-u1-pending" {
		t.Fatalf("expected newest pending task first, got %q", tasks[0].ApprovalID)
	}
	if tasks[1].ApprovalID != "approval-u2-pending" {
		t.Fatalf("expected second pending task, got %q", tasks[1].ApprovalID)
	}
}

type pendingApprovalTaskSeed struct {
	approvalID string
	userID     uint64
	status     string
	createdAt  time.Time
}

func newPendingApprovalsGlobalTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&ai.AIApprovalTask{},
		&usermodel.User{},
		&usermodel.Role{},
		&usermodel.Permission{},
		&usermodel.UserRole{},
		&usermodel.RolePermission{},
	); err != nil {
		t.Fatalf("auto migrate pending global tables: %v", err)
	}
	return db
}

func seedPendingApprovalsGlobalUser(t *testing.T, db *gorm.DB, username string) *usermodel.User {
	t.Helper()
	user := &usermodel.User{
		Username:     username,
		PasswordHash: "hash",
		Status:       1,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user %s: %v", username, err)
	}
	return user
}

func seedPendingApprovalsGlobalRole(t *testing.T, db *gorm.DB, code string) *usermodel.Role {
	t.Helper()
	role := &usermodel.Role{Name: code, Code: code, Status: 1}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role %s: %v", code, err)
	}
	return role
}

func attachPendingApprovalsGlobalRole(t *testing.T, db *gorm.DB, userID usermodel.UserID, roleID usermodel.UserID) {
	t.Helper()
	if err := db.Create(&usermodel.UserRole{
		UserID: int64(userID),
		RoleID: int64(roleID),
	}).Error; err != nil {
		t.Fatalf("attach role to user: %v", err)
	}
}

func grantPendingApprovalsGlobalPermission(t *testing.T, db *gorm.DB, userID usermodel.UserID, code string) {
	t.Helper()

	role := seedPendingApprovalsGlobalRole(t, db, "role-"+code)
	permission := &usermodel.Permission{
		Name:   code,
		Code:   code,
		Type:   1,
		Status: 1,
	}
	if err := db.Create(permission).Error; err != nil {
		t.Fatalf("seed permission %s: %v", code, err)
	}
	attachPendingApprovalsGlobalRole(t, db, userID, role.ID)
	if err := db.Create(&usermodel.RolePermission{
		RoleID:       int64(role.ID),
		PermissionID: int64(permission.ID),
	}).Error; err != nil {
		t.Fatalf("attach role permission: %v", err)
	}
}

func seedPendingApprovalTask(t *testing.T, db *gorm.DB, seed pendingApprovalTaskSeed) {
	t.Helper()
	task := &ai.AIApprovalTask{
		ApprovalID:     seed.approvalID,
		CheckpointID:   seed.approvalID + "-checkpoint",
		SessionID:      seed.approvalID + "-session",
		RunID:          seed.approvalID + "-run",
		UserID:         seed.userID,
		ToolName:       "exec_command",
		ToolCallID:     seed.approvalID + "-call",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         seed.status,
		TimeoutSeconds: 300,
		CreatedAt:      seed.createdAt,
		UpdatedAt:      seed.createdAt,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed approval task %s: %v", seed.approvalID, err)
	}
}

func newPendingApprovalsGlobalRouter(t *testing.T, db *gorm.DB, userID uint64) *gin.Engine {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("uid", userID)
		c.Next()
	})
	h := approvalhandler.NewHTTPHandler(approvalhandler.NewService(&svc.ServiceContext{DB: db}))
	router.GET("/ai/approvals/pending/global", h.ListPendingApprovalsGlobal)
	return router
}
