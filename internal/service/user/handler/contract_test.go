package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/k8s-manage/internal/config"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/testutil"
	"github.com/cy77cc/k8s-manage/internal/utils"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// contractTestSuite provides test infrastructure for API contract tests.
type contractTestSuite struct {
	db        *gorm.DB
	rdb       redis.UniversalClient
	svcCtx    *svc.ServiceContext
	miniRedis *miniredis.Miniredis
}

func newContractTestSuite(t *testing.T) *contractTestSuite {
	t.Helper()

	// Initialize config for JWT
	config.CFG = config.Config{
		JWT: config.JWT{
			Secret:        "test-contract-secret",
			Issuer:        "k8s-manage-test",
			Expire:        time.Hour,
			RefreshExpire: time.Hour * 24,
		},
	}
	utils.MySecret = []byte(config.CFG.JWT.Secret)

	// Setup database
	dbName := "contract_test_" + t.Name()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.UserRole{}, &model.Permission{}, &model.RolePermission{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create default viewer role
	viewerRole := &model.Role{Name: "Viewer", Code: "viewer", Status: 1}
	db.FirstOrCreate(viewerRole, "code = ?", "viewer")

	// Setup Redis
	miniRedis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	return &contractTestSuite{
		db:        db,
		rdb:       rdb,
		svcCtx:    &svc.ServiceContext{DB: db, Rdb: rdb},
		miniRedis: miniRedis,
	}
}

func (s *contractTestSuite) createTestUser(t *testing.T, username, password string) *model.User {
	t.Helper()
	user := testutil.NewUserBuilder().
		WithUsername(username).
		WithPassword(password).
		Build()
	if err := s.db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

// ============================================================================
// Login API Contract Tests
// ============================================================================

func TestLogin_Contract_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	suite.createTestUser(t, "contractuser", "password123")

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)

	body := map[string]string{
		"username": "contractuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Contract assertions
	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess().
		HasCode(xcode.Success).
		HasData()

	// Additional data structure assertions
	data := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)

	// Verify token fields exist (JSON field names are camelCase)
	if _, ok := data["accessToken"]; !ok {
		t.Error("expected accessToken in response data")
	}
	if _, ok := data["refreshToken"]; !ok {
		t.Error("expected refreshToken in response data")
	}
	if _, ok := data["uid"]; !ok {
		t.Error("expected uid in response data")
	}
	if _, ok := data["user"]; !ok {
		t.Error("expected user in response data")
	}

	// Verify user object structure
	user := data["user"].(map[string]any)
	if user["username"] != "contractuser" {
		t.Errorf("expected username 'contractuser', got %v", user["username"])
	}
}

func TestLogin_Contract_UserNotExist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)

	body := map[string]string{
		"username": "nonexistent",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Contract assertions - should be server error wrapping business error
	testutil.AssertContract(t, w.Body.Bytes()).
		IsError().
		ContainsMessage("用户不存在")
}

func TestLogin_Contract_PasswordError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	suite.createTestUser(t, "pwduser", "correctpassword")

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)

	body := map[string]string{
		"username": "pwduser",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		IsError().
		ContainsMessage("密码错误")
}

func TestLogin_Contract_MissingParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)

	// Empty body - binding validation fails
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Empty parameters results in validation error or user not found
	// The handler wraps errors as server errors in current implementation
	testutil.AssertContract(t, w.Body.Bytes()).
		IsError()
}

// ============================================================================
// Register API Contract Tests
// ============================================================================

func TestRegister_Contract_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/register", h.Register)

	body := map[string]string{
		"username": "newuser",
		"password": "password123",
		"email":    "new@example.com",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Contract assertions
	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess().
		HasData()

	data := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)

	// Verify response has token fields (camelCase JSON field names)
	if _, ok := data["accessToken"]; !ok {
		t.Error("expected accessToken in register response")
	}
}

func TestRegister_Contract_DuplicateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	suite.createTestUser(t, "existinguser", "password123")

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/register", h.Register)

	body := map[string]string{
		"username": "existinguser",
		"password": "newpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		IsError().
		ContainsMessage("用户已存在")
}

// ============================================================================
// Refresh API Contract Tests
// ============================================================================

func TestRefresh_Contract_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	// Create user and login
	suite.createTestUser(t, "refreshuser", "password123")

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)

	// First login
	loginBody := map[string]string{
		"username": "refreshuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(loginBody)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	loginData := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)
	refreshToken, ok := loginData["refreshToken"].(string)
	if !ok || refreshToken == "" {
		t.Fatalf("failed to get refreshToken from login response")
	}

	// Then refresh
	refreshBody := map[string]string{
		"refreshToken": refreshToken,
	}
	jsonRefresh, _ := json.Marshal(refreshBody)

	req2 := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(jsonRefresh))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	testutil.AssertContract(t, w2.Body.Bytes()).
		IsSuccess().
		HasData()

	refreshData := testutil.AssertContract(t, w2.Body.Bytes()).GetResponse().Data.(map[string]any)
	if _, ok := refreshData["accessToken"]; !ok {
		t.Error("expected accessToken in refresh response")
	}
}

func TestRefresh_Contract_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	body := map[string]string{
		"refreshToken": "invalid-token",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		IsError()
}

// ============================================================================
// Logout API Contract Tests
// ============================================================================

func TestLogout_Contract_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	suite.createTestUser(t, "logoutuser", "password123")

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)
	r.POST("/logout", h.Logout)

	// Login first
	loginBody := map[string]string{
		"username": "logoutuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(loginBody)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	loginData := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)
	refreshToken, ok := loginData["refreshToken"].(string)
	if !ok || refreshToken == "" {
		t.Fatalf("failed to get refreshToken from login response")
	}

	// Logout
	logoutBody := map[string]string{
		"refreshToken": refreshToken,
	}
	jsonLogout, _ := json.Marshal(logoutBody)

	req2 := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(jsonLogout))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	testutil.AssertContract(t, w2.Body.Bytes()).
		IsSuccess()
}

// ============================================================================
// Me API Contract Tests
// ============================================================================

func TestMe_Contract_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/me", h.Me)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		HasCode(xcode.Unauthorized).
		ContainsMessage("unauthorized")
}

// ============================================================================
// Response Format Verification Tests
// ============================================================================

func TestAPIResponseFormat_StandardFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	suite.createTestUser(t, "formatuser", "password123")

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)

	body := map[string]string{
		"username": "formatuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Parse as generic map to verify structure
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{"code", "msg", "data"}
	for _, field := range requiredFields {
		if _, exists := resp[field]; !exists {
			t.Errorf("required field %q missing in response", field)
		}
	}

	// Verify code is numeric
	if _, ok := resp["code"].(float64); !ok {
		t.Errorf("code field should be numeric, got %T", resp["code"])
	}

	// Verify msg is string
	if _, ok := resp["msg"].(string); !ok {
		t.Errorf("msg field should be string, got %T", resp["msg"])
	}
}

func TestAPIResponseFormat_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newContractTestSuite(t)

	h := NewUserHandler(suite.svcCtx)
	r := gin.New()
	r.POST("/login", h.Login)

	// Invalid request - missing required fields
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Error responses should have code and msg
	if resp["code"] == nil {
		t.Error("error response missing code field")
	}
	if resp["msg"] == nil {
		t.Error("error response missing msg field")
	}

	// Code should be error code (>= 2000)
	code := int(resp["code"].(float64))
	if code < 2000 {
		t.Errorf("expected error code >= 2000, got %d", code)
	}
}
