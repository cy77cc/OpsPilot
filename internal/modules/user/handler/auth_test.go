package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/cy77cc/OpsPilot/api/user/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

type stubAuthLogic struct {
	loginResp   v1.TokenResp
	refreshResp v1.TokenResp
	loginErr    error
	refreshErr  error
	registerErr error
	logoutErr   error
	meResp      map[string]any
	meErr       error

	lastRefreshReq v1.RefreshReq
	lastLogoutReq  v1.LogoutReq
}

func (s *stubAuthLogic) Login(context.Context, v1.LoginReq) (v1.TokenResp, error) {
	return s.loginResp, s.loginErr
}

func (s *stubAuthLogic) Register(context.Context, v1.UserCreateReq) (v1.TokenResp, error) {
	return v1.TokenResp{}, s.registerErr
}

func (s *stubAuthLogic) Refresh(_ context.Context, req v1.RefreshReq) (v1.TokenResp, error) {
	s.lastRefreshReq = req
	return s.refreshResp, s.refreshErr
}

func (s *stubAuthLogic) Logout(_ context.Context, req v1.LogoutReq) error {
	s.lastLogoutReq = req
	return s.logoutErr
}

func (s *stubAuthLogic) GetMe(context.Context, any) (map[string]any, error) {
	return s.meResp, s.meErr
}

func useStubAuthLogic(t *testing.T, stub *stubAuthLogic) {
	t.Helper()
	originalFactory := newAuthLogic
	newAuthLogic = func(*svc.ServiceContext) authLogic {
		return stub
	}
	t.Cleanup(func() {
		newAuthLogic = originalFactory
	})
}

func newAuthTestContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

func setCookieHeader(ctx *gin.Context, cookie *http.Cookie) {
	ctx.Request.Header.Add("Cookie", cookie.String())
}

func findSetCookieHeader(t *testing.T, recorder *httptest.ResponseRecorder, name string) string {
	t.Helper()
	for _, raw := range recorder.Header().Values("Set-Cookie") {
		if strings.HasPrefix(raw, name+"=") {
			return raw
		}
	}
	t.Fatalf("expected Set-Cookie header for %q, got %v", name, recorder.Header().Values("Set-Cookie"))
	return ""
}

func assertCookieSecurityAttrs(t *testing.T, raw string, expectSecure bool) {
	t.Helper()
	if expectSecure && !strings.Contains(raw, "Secure") {
		t.Fatalf("expected Secure cookie attribute, got %q", raw)
	}
	if !expectSecure && strings.Contains(raw, "Secure") {
		t.Fatalf("did not expect Secure cookie attribute, got %q", raw)
	}
	if !strings.Contains(raw, "HttpOnly") {
		t.Fatalf("expected HttpOnly cookie attribute, got %q", raw)
	}
	if !strings.Contains(raw, "SameSite=Strict") {
		t.Fatalf("expected SameSite=Strict cookie attribute, got %q", raw)
	}
}

func decodeResponseData(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}
	if payload["data"] == nil {
		return nil
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected object response data, got %T", payload["data"])
	}
	return data
}

func decodeResponseCode(t *testing.T, recorder *httptest.ResponseRecorder) float64 {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}
	code, ok := payload["code"].(float64)
	if !ok {
		t.Fatalf("expected numeric code in response payload, got %#v", payload["code"])
	}
	return code
}

func assertNoTokenFields(t *testing.T, data map[string]any) {
	t.Helper()
	if _, ok := data["accessToken"]; ok {
		t.Fatalf("response data should not include accessToken: %v", data)
	}
	if _, ok := data["refreshToken"]; ok {
		t.Fatalf("response data should not include refreshToken: %v", data)
	}
}

func TestLogin_SetsAuthCookiesAndRedactsTokenFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		loginResp: v1.TokenResp{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			Roles:        []string{"admin"},
			Permissions:  []string{"user:view"},
			User: &v1.AuthUser{
				Id:       1,
				Username: "alice",
				Name:     "alice",
				Email:    "alice@example.com",
				Status:   "active",
			},
		},
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/login", []byte(`{"username":"alice","password":"secret"}`))

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Login(ctx)

	atCookie := findSetCookieHeader(t, recorder, "opspilot_at")
	rtCookie := findSetCookieHeader(t, recorder, "opspilot_rt")
	assertCookieSecurityAttrs(t, atCookie, false)
	assertCookieSecurityAttrs(t, rtCookie, false)
	if !strings.HasPrefix(atCookie, "opspilot_at=access-token;") {
		t.Fatalf("expected access token cookie value in %q", atCookie)
	}
	if !strings.HasPrefix(rtCookie, "opspilot_rt=refresh-token;") {
		t.Fatalf("expected refresh token cookie value in %q", rtCookie)
	}

	data := decodeResponseData(t, recorder)
	assertNoTokenFields(t, data)
}

func TestRefresh_UsesRefreshCookieOnly_SetsCookiesAndRedactsTokenFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		refreshResp: v1.TokenResp{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			Roles:        []string{"admin"},
			Permissions:  []string{"user:view"},
			User: &v1.AuthUser{
				Id:       1,
				Username: "alice",
				Name:     "alice",
				Email:    "alice@example.com",
				Status:   "active",
			},
		},
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/refresh", []byte(`{"refreshToken":"body-token"}`))
	setCookieHeader(ctx, &http.Cookie{Name: authRefreshCookieName, Value: "cookie-token"})

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Refresh(ctx)

	if got := stub.lastRefreshReq.RefreshToken; got != "cookie-token" {
		t.Fatalf("expected refresh token from cookie transport, got %q", got)
	}

	atCookie := findSetCookieHeader(t, recorder, "opspilot_at")
	rtCookie := findSetCookieHeader(t, recorder, "opspilot_rt")
	assertCookieSecurityAttrs(t, atCookie, false)
	assertCookieSecurityAttrs(t, rtCookie, false)

	data := decodeResponseData(t, recorder)
	assertNoTokenFields(t, data)
}

func TestLogout_ClearsCookiesOnSuccessAndFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		logoutErr error
	}{
		{name: "success"},
		{name: "revocation failure", logoutErr: errors.New("revoke failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubAuthLogic{logoutErr: tt.logoutErr}
			useStubAuthLogic(t, stub)

			ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/logout", []byte(`{"refreshToken":"body-token"}`))
			setCookieHeader(ctx, &http.Cookie{Name: authRefreshCookieName, Value: "cookie-token"})

			h := &UserHandler{svcCtx: &svc.ServiceContext{}}
			h.Logout(ctx)

			if got := stub.lastLogoutReq.RefreshToken; got != "cookie-token" {
				t.Fatalf("expected logout token from cookie transport, got %q", got)
			}

			atCookie := findSetCookieHeader(t, recorder, "opspilot_at")
			rtCookie := findSetCookieHeader(t, recorder, "opspilot_rt")
			assertCookieSecurityAttrs(t, atCookie, false)
			assertCookieSecurityAttrs(t, rtCookie, false)
			if !strings.HasPrefix(atCookie, "opspilot_at=;") || !strings.Contains(atCookie, "Max-Age=0") {
				t.Fatalf("expected cleared access cookie, got %q", atCookie)
			}
			if !strings.HasPrefix(rtCookie, "opspilot_rt=;") || !strings.Contains(rtCookie, "Max-Age=0") {
				t.Fatalf("expected cleared refresh cookie, got %q", rtCookie)
			}
		})
	}
}

func TestRefresh_SetsSecureCookiesWhenForwardedHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		refreshResp: v1.TokenResp{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
		},
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/refresh", nil)
	ctx.Request.Header.Set("X-Forwarded-Proto", "https")
	setCookieHeader(ctx, &http.Cookie{Name: authRefreshCookieName, Value: "cookie-token"})

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Refresh(ctx)

	atCookie := findSetCookieHeader(t, recorder, "opspilot_at")
	rtCookie := findSetCookieHeader(t, recorder, "opspilot_rt")
	assertCookieSecurityAttrs(t, atCookie, true)
	assertCookieSecurityAttrs(t, rtCookie, true)
}

func TestLogin_UsesDomainErrorCodeFromLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		loginErr: xcode.NewErrCode(xcode.UserNotExist),
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/login", []byte(`{"username":"alice","password":"secret"}`))

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Login(ctx)

	if got := decodeResponseCode(t, recorder); got != float64(xcode.UserNotExist) {
		t.Fatalf("expected login response code %d, got %.0f", xcode.UserNotExist, got)
	}
}

func TestRefresh_UsesDomainErrorCodeFromLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		refreshErr: xcode.NewErrCode(xcode.TokenExpired),
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/refresh", nil)
	setCookieHeader(ctx, &http.Cookie{Name: authRefreshCookieName, Value: "cookie-token"})

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Refresh(ctx)

	if got := decodeResponseCode(t, recorder); got != float64(xcode.TokenExpired) {
		t.Fatalf("expected refresh response code %d, got %.0f", xcode.TokenExpired, got)
	}
}

func TestLogin_UnknownLogicErrorUsesStableServerErrorCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		loginErr: errors.New("db connection reset by peer"),
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/login", []byte(`{"username":"alice","password":"secret"}`))

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Login(ctx)

	if got := decodeResponseCode(t, recorder); got != float64(xcode.ServerError) {
		t.Fatalf("expected login response code %d, got %.0f", xcode.ServerError, got)
	}
}

func TestLogout_UsesDomainErrorCodeFromLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		logoutErr: xcode.NewErrCode(xcode.CacheError),
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodPost, "/auth/logout", nil)
	setCookieHeader(ctx, &http.Cookie{Name: authRefreshCookieName, Value: "cookie-token"})

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Logout(ctx)

	if got := decodeResponseCode(t, recorder); got != float64(xcode.CacheError) {
		t.Fatalf("expected logout response code %d, got %.0f", xcode.CacheError, got)
	}
}

func TestMe_UsesDomainErrorCodeFromLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubAuthLogic{
		meErr: xcode.NewErrCode(xcode.Forbidden),
	}
	useStubAuthLogic(t, stub)

	ctx, recorder := newAuthTestContext(http.MethodGet, "/auth/me", nil)
	ctx.Set("uid", uint64(7))

	h := &UserHandler{svcCtx: &svc.ServiceContext{}}
	h.Me(ctx)

	if got := decodeResponseCode(t, recorder); got != float64(xcode.Forbidden) {
		t.Fatalf("expected me response code %d, got %.0f", xcode.Forbidden, got)
	}
}
