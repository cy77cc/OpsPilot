package automation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutomationRunLifecycleAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:automationapi?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.AutomationInventory{},
		&model.AutomationPlaybook{},
		&model.AutomationRun{},
		&model.AutomationRunLog{},
		&model.AutomationExecutionAudit{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Exec("INSERT INTO users (id, username, password_hash, email, phone, status) VALUES (1, 'admin', 'x', 'admin@example.com', '', 1)").Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	h := NewHandler(&svc.ServiceContext{DB: db})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uid", uint(1))
		c.Next()
	})
	g := r.Group("/api/v1/automation")
	g.POST("/runs/preview", h.PreviewRun)
	g.POST("/runs/execute", h.ExecuteRun)
	g.GET("/runs/:id", h.GetRun)
	g.GET("/runs/:id/logs", h.GetRunLogs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/automation/runs/preview", strings.NewReader(`{"action":"collect.inventory","params":{"scope":"all"}}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/automation/runs/execute", strings.NewReader(`{"approval_token":"ok","action":"collect.inventory","params":{"scope":"all"}}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("execute status=%d body=%s", w.Code, w.Body.String())
	}
	var executeResp struct {
		Data model.AutomationRun `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &executeResp); err != nil {
		t.Fatalf("decode execute: %v", err)
	}
	if strings.TrimSpace(executeResp.Data.ID) == "" {
		t.Fatalf("expected run id")
	}
	if executeResp.Data.Status != "succeeded" {
		t.Fatalf("expected succeeded, got %s", executeResp.Data.Status)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/automation/runs/"+executeResp.Data.ID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get run status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/automation/runs/"+executeResp.Data.ID+"/logs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get logs status=%d body=%s", w.Code, w.Body.String())
	}
	var logsResp struct {
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &logsResp); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if logsResp.Data.Total < 1 {
		t.Fatalf("expected logs, got total=%d, body=%s", logsResp.Data.Total, w.Body.String())
	}
}
