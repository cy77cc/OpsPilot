package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	llmapi "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/api"
	llmmodel "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminLLMProviderRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	h := llmapi.NewHTTPHandler(&svc.ServiceContext{})
	models := v1.Group("/admin/ai/models")
	{
		models.GET("", h.ListModels)
		models.GET("/:id", h.GetModel)
		models.POST("", h.CreateModel)
		models.PUT("/:id", h.UpdateModel)
		models.PUT("/:id/default", h.SetDefaultModel)
		models.DELETE("/:id", h.DeleteModel)
		models.POST("/import/preview", h.PreviewImport)
		models.POST("/import", h.ImportModels)
	}

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
	h := llmapi.NewHTTPHandlerWithDB(db)

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

	var stored llmmodel.AILLMProvider
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
	h := llmapi.NewHTTPHandlerWithDB(db)

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
	h := llmapi.NewHTTPHandlerWithDB(db)

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

func TestLLMProviderHandler_PreviewImportSuccessMatchesFrontendContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := llmapi.NewHTTPHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/models/import/preview", bytes.NewBufferString(`{"replace_all":true,"providers":[{"name":"Qwen Prod","provider":"qwen","model":"qwen-max","base_url":"https://example.com","api_key":"sk-test"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.PreviewImport(c)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ReplaceAll bool             `json:"replace_all"`
			Total      int              `json:"total"`
			Providers  []map[string]any `json:"providers"`
			Count      int              `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != int(xcode.Success) {
		t.Fatalf("expected success code %d, got %d", xcode.Success, resp.Code)
	}
	if !resp.Data.ReplaceAll {
		t.Fatal("expected replace_all=true")
	}
	if resp.Data.Total != 1 || resp.Data.Count != 1 || len(resp.Data.Providers) != 1 {
		t.Fatalf("unexpected preview payload: %#v", resp.Data)
	}
	if resp.Data.Providers[0]["name"] != "Qwen Prod" {
		t.Fatalf("expected provider name Qwen Prod, got %#v", resp.Data.Providers[0]["name"])
	}
}

func TestLLMProviderHandler_ImportSuccessMatchesFrontendContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := llmapi.NewHTTPHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/models/import", bytes.NewBufferString(`{"replace_all":false,"providers":[{"name":"Qwen Prod","provider":"qwen","model":"qwen-max","base_url":"https://example.com","api_key":"sk-test"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ImportModels(c)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ReplaceAll bool             `json:"replace_all"`
			Created    int              `json:"created"`
			Updated    int              `json:"updated"`
			Providers  []map[string]any `json:"providers"`
			Count      int              `json:"count"`
			List       []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != int(xcode.Success) {
		t.Fatalf("expected success code %d, got %d", xcode.Success, resp.Code)
	}
	if resp.Data.ReplaceAll {
		t.Fatal("expected replace_all=false")
	}
	if resp.Data.Created != 1 || resp.Data.Updated != 0 || resp.Data.Count != 1 {
		t.Fatalf("unexpected import counters: %#v", resp.Data)
	}
	if len(resp.Data.Providers) != 1 || len(resp.Data.List) != 1 {
		t.Fatalf("unexpected import rows: %#v", resp.Data)
	}
}

func TestLLMProviderHandler_UpdateModel_PartialPayloadKeepsExistingFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := llmapi.NewHTTPHandlerWithDB(db)

	if err := db.Create(&llmmodel.AILLMProvider{
		ID:          1,
		Name:        "Qwen Prod",
		Provider:    "qwen",
		Model:       "qwen-max",
		BaseURL:     "https://example.com",
		APIKey:      "cipher",
		Temperature: 0.8,
		Thinking:    true,
		IsDefault:   true,
		IsEnabled:   true,
		SortOrder:   10,
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/ai/models/1", bytes.NewBufferString(`{"name":"Qwen Prod v2"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateModel(c)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if resp.Code != int(xcode.Success) {
		t.Fatalf("expected success code %d, got %d", xcode.Success, resp.Code)
	}

	var row llmmodel.AILLMProvider
	if err := db.First(&row, 1).Error; err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if row.Name != "Qwen Prod v2" {
		t.Fatalf("expected updated name, got %q", row.Name)
	}
	if !row.IsDefault || !row.IsEnabled || !row.Thinking {
		t.Fatalf("expected existing boolean flags unchanged, got default=%v enabled=%v thinking=%v", row.IsDefault, row.IsEnabled, row.Thinking)
	}
	if row.Temperature != 0.8 {
		t.Fatalf("expected temperature unchanged to 0.8, got %v", row.Temperature)
	}
}

func TestLLMProviderHandler_DeleteModel_RejectsDefaultModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := llmapi.NewHTTPHandlerWithDB(db)

	if err := db.Create(&llmmodel.AILLMProvider{
		ID:        1,
		Name:      "Default",
		Provider:  "qwen",
		Model:     "qwen-max",
		BaseURL:   "https://example.com",
		APIKey:    "cipher",
		IsDefault: true,
		IsEnabled: true,
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/ai/models/1", nil)

	h.DeleteModel(c)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if resp.Code != int(xcode.LLMProviderInUse) {
		t.Fatalf("expected code %d, got %d", xcode.LLMProviderInUse, resp.Code)
	}

	var count int64
	if err := db.Model(&llmmodel.AILLMProvider{}).Where("id = ? AND deleted_at IS NULL", 1).Count(&count).Error; err != nil {
		t.Fatalf("count provider: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected default model to remain, got count=%d", count)
	}
}

func TestLLMProviderHandler_ImportReplaceAll_ReplacesExistingRows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := llmapi.NewHTTPHandlerWithDB(db)

	if err := db.Create(&llmmodel.AILLMProvider{
		ID:       1,
		Name:     "Old",
		Provider: "ark",
		Model:    "doubao",
		BaseURL:  "https://old.example.com",
		APIKey:   "cipher",
	}).Error; err != nil {
		t.Fatalf("seed old provider: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/models/import", bytes.NewBufferString(`{"replace_all":true,"providers":[{"name":"New","provider":"qwen","model":"qwen-max","base_url":"https://new.example.com","api_key":"sk-test"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ImportModels(c)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if resp.Code != int(xcode.Success) {
		t.Fatalf("expected success code %d, got %d", xcode.Success, resp.Code)
	}

	var rows []llmmodel.AILLMProvider
	if err := db.Where("deleted_at IS NULL").Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "New" {
		t.Fatalf("expected only imported provider to remain, got %#v", rows)
	}
}

func TestLLMProviderHandler_ImportModels_TransactionalRollbackOnError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setHandlerTestEncryptionKey(t)
	db := newLLMProviderHandlerTestDB(t)
	h := llmapi.NewHTTPHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/models/import", bytes.NewBufferString(`{"replace_all":false,"providers":[{"name":"Qwen-A","provider":"qwen","model":"qwen-max","base_url":"https://example.com","api_key":"sk-1"},{"name":"Qwen-B","provider":"qwen","model":"qwen-max","base_url":"https://example.com","api_key":"sk-2"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ImportModels(c)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if resp.Code != int(xcode.ServerError) {
		t.Fatalf("expected server error code %d, got %d", xcode.ServerError, resp.Code)
	}

	var count int64
	if err := db.Model(&llmmodel.AILLMProvider{}).Where("deleted_at IS NULL").Count(&count).Error; err != nil {
		t.Fatalf("count providers: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to keep provider table empty, got %d rows", count)
	}
}

func newLLMProviderHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&llmmodel.AILLMProvider{}); err != nil {
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
