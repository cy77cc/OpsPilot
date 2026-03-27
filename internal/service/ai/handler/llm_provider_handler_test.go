package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/config"
	aiService "github.com/cy77cc/OpsPilot/internal/service/ai"
	aimodel "github.com/cy77cc/OpsPilot/internal/service/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminLLMProviderRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	aiService.RegisterAdminAIHandlers(v1, &svc.ServiceContext{})

	routes := r.Routes()
	seen := make(map[string]bool, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = true
	}

	expected := []string{
		http.MethodGet + " /api/v1/admin/ai/models",
		http.MethodGet + " /api/v1/admin/ai/models/:id",
		http.MethodPost + " /api/v1/admin/ai/models",
		http.MethodPut + " /api/v1/admin/ai/models/:id",
		http.MethodPut + " /api/v1/admin/ai/models/:id/default",
		http.MethodDelete + " /api/v1/admin/ai/models/:id",
		http.MethodPost + " /api/v1/admin/ai/models/import/preview",
		http.MethodPost + " /api/v1/admin/ai/models/import",
	}
	for _, route := range expected {
		if !seen[route] {
			t.Fatalf("expected route %q to be registered", route)
		}
	}
}

func TestLLMProviderHandler_CreateAndListModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := aimodel.NewHTTPHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/models", bytes.NewBufferString(`{"name":"Qwen Plus","provider":"qwen","model":"qwen-plus","base_url":"https://example.com/v1","api_key":"sk-test-secret","is_default":true}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateModel(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http 200, got %d", recorder.Code)
	}

	var createResp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Code != int(xcode.Success) {
		t.Fatalf("expected success code %d, got %d", xcode.Success, createResp.Code)
	}
	if createResp.Data["api_key_masked"] == "" {
		t.Fatal("expected masked api key in create response")
	}

	var stored aimodel.LLMProviderRecord
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("reload stored provider: %v", err)
	}
	if stored.APIKey == "sk-test-secret" {
		t.Fatal("expected api key to be encrypted in the database")
	}

	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai/models", nil)

	h.ListModels(c)

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			List  []map[string]any `json:"list"`
			Total int              `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResp.Code != int(xcode.Success) {
		t.Fatalf("expected success code %d, got %d", xcode.Success, listResp.Code)
	}
	if listResp.Data.Total != 1 || len(listResp.Data.List) != 1 {
		t.Fatalf("expected one provider in list, got %#v", listResp.Data)
	}
	if listResp.Data.List[0]["api_key_masked"] == "" {
		t.Fatal("expected masked api key in list response")
	}
}

func TestLLMProviderHandler_PreviewImportInvalidJSONReturnsLLMImportInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := aimodel.NewHTTPHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/models/import/preview", bytes.NewBufferString(`{"providers":[`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.PreviewImport(c)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != int(xcode.LLMImportInvalidJSON) {
		t.Fatalf("expected code %d, got %d", xcode.LLMImportInvalidJSON, resp.Code)
	}
}

func TestLLMProviderHandler_PreviewImportValidationFailureReturnsLLMImportValidationFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := aimodel.NewHTTPHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/models/import/preview", bytes.NewBufferString(`{"providers":[{"name":"OnlyName"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.PreviewImport(c)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != int(xcode.LLMImportValidationFail) {
		t.Fatalf("expected code %d, got %d", xcode.LLMImportValidationFail, resp.Code)
	}
}

func newLLMProviderHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&aimodel.LLMProviderRecord{}); err != nil {
		t.Fatalf("auto migrate provider record: %v", err)
	}
	return db
}

func setHandlerTestEncryptionKey(t *testing.T) {
	t.Helper()

	original := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = "llm-provider-test-key"
	t.Cleanup(func() { config.CFG.Security.EncryptionKey = original })
}
