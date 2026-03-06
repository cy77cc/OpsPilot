package testutil

import (
	"encoding/json"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/xcode"
)

// APIResponse represents the standard API response format.
type APIResponse struct {
	Code xcode.Xcode `json:"code"`
	Msg  string      `json:"msg"`
	Data any         `json:"data,omitempty"`
}

// ContractAssertion defines assertions for API contract testing.
type ContractAssertion struct {
	t       *testing.T
	resp    APIResponse
	rawBody []byte
}

// AssertContract parses response body and returns a ContractAssertion.
func AssertContract(t *testing.T, body []byte) *ContractAssertion {
	t.Helper()
	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse response as JSON: %v\nbody: %s", err, string(body))
	}
	return &ContractAssertion{
		t:       t,
		resp:    resp,
		rawBody: body,
	}
}

// IsSuccess asserts the response indicates success (code 1000-1999).
func (a *ContractAssertion) IsSuccess() *ContractAssertion {
	a.t.Helper()
	if a.resp.Code < 1000 || a.resp.Code >= 2000 {
		a.t.Errorf("expected success code (1000-1999), got %d: %s", a.resp.Code, a.resp.Msg)
	}
	return a
}

// IsError asserts the response indicates an error (code >= 2000).
func (a *ContractAssertion) IsError() *ContractAssertion {
	a.t.Helper()
	if a.resp.Code < 2000 {
		a.t.Errorf("expected error code (>= 2000), got %d", a.resp.Code)
	}
	return a
}

// HasCode asserts the response has a specific code.
func (a *ContractAssertion) HasCode(expected xcode.Xcode) *ContractAssertion {
	a.t.Helper()
	if a.resp.Code != expected {
		a.t.Errorf("expected code %d, got %d: %s", expected, a.resp.Code, a.resp.Msg)
	}
	return a
}

// HasMessage asserts the response message contains expected text.
func (a *ContractAssertion) HasMessage(expected string) *ContractAssertion {
	a.t.Helper()
	if a.resp.Msg != expected {
		a.t.Errorf("expected message %q, got %q", expected, a.resp.Msg)
	}
	return a
}

// ContainsMessage asserts the response message contains expected text.
func (a *ContractAssertion) ContainsMessage(expected string) *ContractAssertion {
	a.t.Helper()
	if !containsString(a.resp.Msg, expected) {
		a.t.Errorf("expected message to contain %q, got %q", expected, a.resp.Msg)
	}
	return a
}

// HasData asserts the response has non-nil data.
func (a *ContractAssertion) HasData() *ContractAssertion {
	a.t.Helper()
	if a.resp.Data == nil {
		a.t.Error("expected response to have data, got nil")
	}
	return a
}

// HasNoData asserts the response has no data.
func (a *ContractAssertion) HasNoData() *ContractAssertion {
	a.t.Helper()
	if a.resp.Data != nil {
		a.t.Errorf("expected response to have no data, got: %v", a.resp.Data)
	}
	return a
}

// DataContains asserts data contains a specific key/value.
func (a *ContractAssertion) DataContains(key string, expectedValue any) *ContractAssertion {
	a.t.Helper()
	if a.resp.Data == nil {
		a.t.Fatal("cannot assert data contains key: data is nil")
	}
	data, ok := a.resp.Data.(map[string]any)
	if !ok {
		a.t.Fatalf("data is not a map, got %T", a.resp.Data)
	}
	val, exists := data[key]
	if !exists {
		a.t.Errorf("expected data to contain key %q", key)
		return a
	}
	if val != expectedValue {
		a.t.Errorf("expected data[%q] = %v, got %v", key, expectedValue, val)
	}
	return a
}

// DataHasList asserts data contains a list with expected count.
func (a *ContractAssertion) DataHasList(key string, expectedCount int) *ContractAssertion {
	a.t.Helper()
	if a.resp.Data == nil {
		a.t.Fatal("cannot assert data has list: data is nil")
	}
	data, ok := a.resp.Data.(map[string]any)
	if !ok {
		a.t.Fatalf("data is not a map, got %T", a.resp.Data)
	}
	list, exists := data[key]
	if !exists {
		a.t.Errorf("expected data to contain key %q", key)
		return a
	}
	listSlice, ok := list.([]any)
	if !ok {
		a.t.Fatalf("data[%q] is not a list, got %T", key, list)
	}
	if len(listSlice) != expectedCount {
		a.t.Errorf("expected data[%q] to have %d items, got %d", key, expectedCount, len(listSlice))
	}
	return a
}

// GetResponse returns the parsed API response.
func (a *ContractAssertion) GetResponse() APIResponse {
	return a.resp
}

// GetRawBody returns the raw response body.
func (a *ContractAssertion) GetRawBody() []byte {
	return a.rawBody
}

// ============================================================================
// Error Code Category Assertions
// ============================================================================

// IsClientError asserts the response is a client error (2000-2999).
func (a *ContractAssertion) IsClientError() *ContractAssertion {
	a.t.Helper()
	if a.resp.Code < 2000 || a.resp.Code >= 3000 {
		a.t.Errorf("expected client error code (2000-2999), got %d", a.resp.Code)
	}
	return a
}

// IsServerError asserts the response is a server error (3000-3999).
func (a *ContractAssertion) IsServerError() *ContractAssertion {
	a.t.Helper()
	if a.resp.Code < 3000 || a.resp.Code >= 4000 {
		a.t.Errorf("expected server error code (3000-3999), got %d", a.resp.Code)
	}
	return a
}

// IsBusinessError asserts the response is a business error (4000-4999).
func (a *ContractAssertion) IsBusinessError() *ContractAssertion {
	a.t.Helper()
	if a.resp.Code < 4000 || a.resp.Code >= 5000 {
		a.t.Errorf("expected business error code (4000-4999), got %d", a.resp.Code)
	}
	return a
}

// ============================================================================
// Common API Contract Tests
// ============================================================================

// AssertStandardSuccessResponse asserts response follows standard success format.
func AssertStandardSuccessResponse(t *testing.T, body []byte) {
	t.Helper()
	AssertContract(t, body).
		IsSuccess().
		HasMessage("请求成功")
}

// AssertErrorResponse asserts response follows standard error format.
func AssertErrorResponse(t *testing.T, body []byte, expectedCode xcode.Xcode) {
	t.Helper()
	AssertContract(t, body).
		HasCode(expectedCode).
		IsError()
}

// AssertUnauthorizedResponse asserts response is unauthorized.
func AssertUnauthorizedResponse(t *testing.T, body []byte) {
	t.Helper()
	AssertContract(t, body).
		HasCode(xcode.Unauthorized).
		ContainsMessage("未认证")
}

// AssertForbiddenResponse asserts response is forbidden.
func AssertForbiddenResponse(t *testing.T, body []byte) {
	t.Helper()
	AssertContract(t, body).
		HasCode(xcode.Forbidden).
		ContainsMessage("无权限")
}

// AssertNotFoundResponse asserts response is not found.
func AssertNotFoundResponse(t *testing.T, body []byte) {
	t.Helper()
	AssertContract(t, body).
		HasCode(xcode.NotFound).
		ContainsMessage("不存在")
}

// AssertParamErrorResponse asserts response is a parameter error.
func AssertParamErrorResponse(t *testing.T, body []byte) {
	t.Helper()
	AssertContract(t, body).
		HasCode(xcode.ParamError).
		ContainsMessage("参数错误")
}

// Helper function - uses shared contains function from assertions.go
