package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHostList_DoesNotExposeSSHPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&hostmodel.Node{}); err != nil {
		t.Fatalf("auto migrate node table: %v", err)
	}

	node := &hostmodel.Node{
		Name:        "task6-host",
		IP:          "10.0.0.8",
		Port:        22,
		SSHUser:     "root",
		SSHPassword: "Task6-Sensitive-Password",
		Status:      "online",
		Source:      "manual_ssh",
	}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	h := NewHandler(&svc.ServiceContext{DB: db})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/hosts", nil)

	h.List(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http 200, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "ssh_password") {
		t.Fatalf("expected host list response to hide ssh_password, got body: %s", recorder.Body.String())
	}
}

func TestHostGet_DoesNotExposeSSHPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&hostmodel.Node{}); err != nil {
		t.Fatalf("auto migrate node table: %v", err)
	}

	node := &hostmodel.Node{
		Name:        "task6-host-detail",
		IP:          "10.0.0.9",
		Port:        22,
		SSHUser:     "root",
		SSHPassword: "Task6-Detail-Sensitive-Password",
		Status:      "online",
		Source:      "manual_ssh",
	}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	h := NewHandler(&svc.ServiceContext{DB: db})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", node.ID)}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/hosts/"+fmt.Sprintf("%d", node.ID), nil)

	h.Get(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http 200, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "ssh_password") {
		t.Fatalf("expected host detail response to hide ssh_password, got body: %s", recorder.Body.String())
	}
}
