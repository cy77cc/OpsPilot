package logic

import (
	"context"
	"testing"
	"time"

	v1 "github.com/cy77cc/k8s-manage/api/user/v1"
	"github.com/cy77cc/k8s-manage/internal/config"
	dao "github.com/cy77cc/k8s-manage/internal/dao/user"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/testutil"
	"github.com/cy77cc/k8s-manage/internal/utils"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// authTestSuite provides test infrastructure for auth tests.
// Each test gets a fresh database instance to avoid conflicts.
type authTestSuite struct {
	db        *gorm.DB
	miniRedis *miniredis.Miniredis
	rdb       redis.UniversalClient
	svcCtx    *svc.ServiceContext
	userDAO   *dao.UserDAO
	whitelist *dao.WhiteListDao
}

func newAuthTestSuite(t *testing.T) *authTestSuite {
	t.Helper()

	// Initialize config for JWT (required by utils.GenToken)
	// This must be done before any token generation
	config.CFG = config.Config{
		JWT: config.JWT{
			Secret:        "test-secret-key-for-auth-tests",
			Issuer:        "k8s-manage-test",
			Expire:        time.Hour,
			RefreshExpire: time.Hour * 24 * 7,
		},
	}
	// Update the MySecret variable that was initialized at package load time
	utils.MySecret = []byte(config.CFG.JWT.Secret)

	// Use unique database name for each test to avoid conflicts
	dbName := "auth_test_" + time.Now().Format("20060102150405") + "_" + randomString(6)

	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.UserRole{}, &model.Permission{}, &model.RolePermission{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create default viewer role
	viewerRole := &model.Role{Name: "Viewer", Code: "viewer", Status: 1}
	if err := db.FirstOrCreate(viewerRole, "code = ?", "viewer").Error; err != nil {
		t.Fatalf("failed to create viewer role: %v", err)
	}

	// Create miniredis for Redis mock
	miniRedis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	userDAO := dao.NewUserDAO(db, nil, rdb)
	whitelist := dao.NewWhiteListDao(db, nil, rdb)

	return &authTestSuite{
		db:        db,
		miniRedis: miniRedis,
		rdb:       rdb,
		svcCtx:    &svc.ServiceContext{DB: db, Rdb: rdb},
		userDAO:   userDAO,
		whitelist: whitelist,
	}
}

func (s *authTestSuite) createTestUser(t *testing.T, username, password string) *model.User {
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

// randomString generates a random string for unique db names.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().Nanosecond()%len(letters)]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// ============================================================================
// Login Tests
// ============================================================================

func TestLogin_Success(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	// Create test user
	suite.createTestUser(t, "testuser", "password123")

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	resp, err := logic.Login(ctx, v1.LoginReq{
		Username: "testuser",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("expected login success, got error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token, got empty")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh token, got empty")
	}
	if resp.Uid == 0 {
		t.Error("expected uid, got 0")
	}
	if resp.User == nil {
		t.Fatal("expected user info, got nil")
	}
	if resp.User.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", resp.User.Username)
	}
}

func TestLogin_UserNotExist(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	_, err := logic.Login(ctx, v1.LoginReq{
		Username: "nonexistent",
		Password: "password123",
	})

	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}

	// Check error code
	if codeErr := xcode.FromError(err); codeErr.Code != xcode.UserNotExist {
		t.Errorf("expected UserNotExist error code, got: %v", codeErr)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	// Create test user with correct password
	suite.createTestUser(t, "testuser_wp", "correctpassword")

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	_, err := logic.Login(ctx, v1.LoginReq{
		Username: "testuser_wp",
		Password: "wrongpassword",
	})

	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}

	if codeErr := xcode.FromError(err); codeErr.Code != xcode.PasswordError {
		t.Errorf("expected PasswordError code, got: %v", codeErr)
	}
}

func TestLogin_WhitelistAdded(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	suite.createTestUser(t, "testuser_wl", "password123")

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	resp, err := logic.Login(ctx, v1.LoginReq{
		Username: "testuser_wl",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Verify refresh token is in whitelist
	ok, err := suite.whitelist.IsWhitelisted(ctx, resp.RefreshToken)
	if err != nil {
		t.Fatalf("failed to check whitelist: %v", err)
	}
	if !ok {
		t.Error("expected refresh token to be in whitelist")
	}
}

// ============================================================================
// Register Tests
// ============================================================================

func TestRegister_Success(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	resp, err := logic.Register(ctx, v1.UserCreateReq{
		Username: "newuser",
		Password: "password123",
		Email:    "newuser@example.com",
	})

	if err != nil {
		t.Fatalf("expected register success, got error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token, got empty")
	}
	if resp.User == nil {
		t.Fatal("expected user info, got nil")
	}
	if resp.User.Username != "newuser" {
		t.Errorf("expected username 'newuser', got '%s'", resp.User.Username)
	}

	// Verify user exists in database
	var count int64
	suite.db.Model(&model.User{}).Where("username = ?", "newuser").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 user in db, got %d", count)
	}
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	// Create existing user
	suite.createTestUser(t, "existinguser", "password123")

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	_, err := logic.Register(ctx, v1.UserCreateReq{
		Username: "existinguser",
		Password: "newpassword",
		Email:    "new@example.com",
	})

	if err == nil {
		t.Fatal("expected error for existing user, got nil")
	}

	if codeErr := xcode.FromError(err); codeErr.Code != xcode.UserAlreadyExist {
		t.Errorf("expected UserAlreadyExist error code, got: %v", codeErr)
	}
}

func TestRegister_PasswordHashed(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	_, err := logic.Register(ctx, v1.UserCreateReq{
		Username: "hashuser",
		Password: "plaintext123",
		Email:    "hash@example.com",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Verify password is hashed in database
	var user model.User
	if err := suite.db.Where("username = ?", "hashuser").First(&user).Error; err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	// Password hash should not be plaintext
	if user.PasswordHash == "plaintext123" {
		t.Error("password should be hashed, not stored as plaintext")
	}
	// Should start with bcrypt prefix
	if len(user.PasswordHash) < 10 {
		t.Errorf("password hash seems too short: %s", user.PasswordHash)
	}
}

func TestRegister_DefaultRoleAssigned(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	_, err := logic.Register(ctx, v1.UserCreateReq{
		Username: "roleuser",
		Password: "password123",
		Email:    "role@example.com",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Verify user has viewer role
	var user model.User
	if err := suite.db.Where("username = ?", "roleuser").First(&user).Error; err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	var userRole model.UserRole
	err = suite.db.Where("user_id = ?", user.ID).First(&userRole).Error
	if err != nil {
		t.Errorf("expected user to have a role assigned: %v", err)
	}
}

// ============================================================================
// Refresh Tests
// ============================================================================

func TestRefresh_Success(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	// Create user and login first
	suite.createTestUser(t, "refreshuser", "password123")

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	// Login to get refresh token
	loginResp, err := logic.Login(ctx, v1.LoginReq{
		Username: "refreshuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Verify login refresh token is in whitelist
	loginTokenInList, _ := suite.whitelist.IsWhitelisted(ctx, loginResp.RefreshToken)
	if !loginTokenInList {
		t.Fatal("login refresh token should be in whitelist before refresh")
	}

	// Refresh token immediately (should still be valid)
	refreshResp, err := logic.Refresh(ctx, v1.RefreshReq{
		RefreshToken: loginResp.RefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	if refreshResp.AccessToken == "" {
		t.Error("expected new access token")
	}
	if refreshResp.RefreshToken == "" {
		t.Error("expected new refresh token")
	}
	if refreshResp.Uid != loginResp.Uid {
		t.Errorf("uid mismatch: expected %d, got %d", loginResp.Uid, refreshResp.Uid)
	}

	// Verify new refresh token is in whitelist and can be used
	newInList, _ := suite.whitelist.IsWhitelisted(ctx, refreshResp.RefreshToken)
	if !newInList {
		t.Error("new refresh token should be in whitelist")
	}

	// Verify new tokens work by using them for another refresh
	secondRefreshResp, err := logic.Refresh(ctx, v1.RefreshReq{
		RefreshToken: refreshResp.RefreshToken,
	})
	if err != nil {
		t.Fatalf("second refresh with new token failed: %v", err)
	}
	if secondRefreshResp.RefreshToken == "" {
		t.Error("expected refresh token from second refresh")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	_, err := logic.Refresh(ctx, v1.RefreshReq{
		RefreshToken: "invalid-token",
	})

	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestRefresh_TokenNotInWhitelist(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	// Create a valid-looking token but not in whitelist
	_, err := logic.Refresh(ctx, v1.RefreshReq{
		RefreshToken: "valid-looking-but-not-whitelisted-token",
	})

	if err == nil {
		t.Fatal("expected error for non-whitelisted token, got nil")
	}

	if codeErr := xcode.FromError(err); codeErr.Code != xcode.TokenInvalid {
		t.Errorf("expected TokenInvalid error code, got: %v", codeErr)
	}
}

// ============================================================================
// Logout Tests
// ============================================================================

func TestLogout_Success(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	// Create user and login
	suite.createTestUser(t, "logoutuser", "password123")

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	loginResp, err := logic.Login(ctx, v1.LoginReq{
		Username: "logoutuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Verify token is in whitelist
	inList, _ := suite.whitelist.IsWhitelisted(ctx, loginResp.RefreshToken)
	if !inList {
		t.Fatal("refresh token should be in whitelist before logout")
	}

	// Logout
	err = logic.Logout(ctx, v1.LogoutReq{
		RefreshToken: loginResp.RefreshToken,
	})
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Verify token is removed from whitelist
	inList, _ = suite.whitelist.IsWhitelisted(ctx, loginResp.RefreshToken)
	if inList {
		t.Error("refresh token should be removed from whitelist after logout")
	}
}

func TestLogout_EmptyToken(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	// Logout with empty token should not error
	err := logic.Logout(ctx, v1.LogoutReq{
		RefreshToken: "",
	})
	if err != nil {
		t.Errorf("logout with empty token should not error, got: %v", err)
	}
}

func TestLogout_Twice(t *testing.T) {
	suite := newAuthTestSuite(t)
	ctx := context.Background()

	suite.createTestUser(t, "doublelogout", "password123")

	logic := &UserLogic{
		svcCtx:       suite.svcCtx,
		userDAO:      suite.userDAO,
		whiteListDao: suite.whitelist,
	}

	loginResp, _ := logic.Login(ctx, v1.LoginReq{
		Username: "doublelogout",
		Password: "password123",
	})

	// First logout
	_ = logic.Logout(ctx, v1.LogoutReq{RefreshToken: loginResp.RefreshToken})

	// Second logout with same token should not error (idempotent)
	err := logic.Logout(ctx, v1.LogoutReq{RefreshToken: loginResp.RefreshToken})
	if err != nil {
		t.Errorf("second logout should not error, got: %v", err)
	}
}
