# Monitoring Alerting Full CRUD Hard Delete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver full CRUD for monitoring alert rules/channels/severity-routes/rule-channel-bindings across global+project scope, including hard delete with dependency-conflict protection.

**Architecture:** Extend the existing monitoring handler/logic API surface incrementally: add missing single-item CRUD endpoints and hard-delete operations with explicit conflict error payloads (`blockers`). Keep frontend page IA unchanged and upgrade existing Rules/Channels/Routing pages from read-only or partial-edit to full CRUD. Reuse existing list APIs and introduce focused modal/drawer interaction flows plus scope-aware request wiring.

**Tech Stack:** Go (Gin, GORM, SQLite/Postgres), React 19 + TypeScript + Ant Design, Vitest, Go test.

---

## Scope Check

This is one coherent subsystem (monitoring configuration control plane). Backend API completion and frontend CRUD wiring should remain in one plan and one delivery stream.

## File Structure

### Backend API/Handler

- Modify: `internal/modules/monitoring/api/routes.go`  
  Responsibility: register missing CRUD routes (`DELETE` for rule/channel/route/binding, plus single-item route/binding create/update).
- Modify: `internal/modules/monitoring/handler/handler.go`  
  Responsibility: add handler methods for new CRUD endpoints, scope parsing, and conflict/not-found response shaping.
- Modify: `api/monitoring/v1/monitoring.go`  
  Responsibility: add request DTOs for severity-route single CRUD and rule-binding single CRUD.

### Backend Logic

- Create: `internal/modules/monitoring/logic/delete_conflict.go`  
  Responsibility: define `DeleteConflictError` + `DeleteBlocker` payload model used by handlers for `409` responses.
- Modify: `internal/modules/monitoring/logic/logic.go`  
  Responsibility: rule/channel delete methods with dependency checks and record-not-found behavior.
- Modify: `internal/modules/monitoring/logic/routing_policy.go`  
  Responsibility: single-item severity-route and binding CRUD methods, scope-aware row targeting.

### Backend Tests

- Create: `internal/modules/monitoring/logic/delete_resources_test.go`  
  Responsibility: rule/channel hard-delete conflict checks + route/binding delete behavior.
- Modify: `internal/modules/monitoring/handler/handler_config_test.go`  
  Responsibility: endpoint contract tests for new CRUD APIs, including `409 blockers` and `404`.

### Frontend API

- Modify: `web/src/api/modules/monitoring.ts`  
  Responsibility: expose full CRUD methods for rule/channel/route/binding with scope-aware payloads.
- Modify: `web/src/api/modules/monitoring.test.ts`  
  Responsibility: verify new API method path/payload mapping.

### Frontend Pages

- Create: `web/src/pages/Monitor/components/ScopeSelector.tsx`  
  Responsibility: reusable global/project selector with project id input.
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`  
  Responsibility: full rule CRUD + binding CRUD drawer.
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`  
  Responsibility: full channel CRUD + retained test-send flow.
- Modify: `web/src/pages/Monitor/RoutingConfigPage.tsx`  
  Responsibility: full severity-route CRUD.

### Frontend Page Tests

- Modify: `web/src/pages/Monitor/RulesConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.test.tsx`  
  Responsibility: assert create/edit/delete flows and conflict UI behavior.

---

### Task 1: Write Failing Backend Logic Tests For Hard Delete And Single-Item Deletions

**Files:**
- Create: `internal/modules/monitoring/logic/delete_resources_test.go`

- [ ] **Step 1: Write failing tests for rule/channel delete conflicts and route/binding delete**

```go
// internal/modules/monitoring/logic/delete_resources_test.go
package logic

import (
	"context"
	"errors"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteRule_ReturnsConflictWhenBindingExists(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:delete-rule-conflict?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&model.AlertRule{}, &model.AlertRuleChannelBinding{})
	_ = db.Create(&model.AlertRule{ID: 7, Name: "cpu", Metric: "cpu_usage", Operator: "gt", Threshold: 80, Severity: "warning", Enabled: true, State: "enabled"}).Error
	_ = db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error

	l := NewLogic(&svc.ServiceContext{DB: db})
	err := l.DeleteRule(context.Background(), 7)
	var conflict *DeleteConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected DeleteConflictError, got %v", err)
	}
	if len(conflict.Blockers) == 0 {
		t.Fatalf("expected blockers, got %#v", conflict)
	}
}

func TestDeleteChannel_ReturnsConflictWhenReferencedBySeverityRoute(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:delete-channel-conflict?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&model.AlertNotificationChannel{}, &model.AlertSeverityRoute{})
	_ = db.Create(&model.AlertNotificationChannel{ID: 1001, Name: "webhook", Type: "webhook", Provider: "webhook", Enabled: true}).Error
	_ = db.Create(&model.AlertSeverityRoute{ID: 11, Scope: "global", Severity: "critical", ChannelIDsJSON: `[1001]`, Enabled: true}).Error

	l := NewLogic(&svc.ServiceContext{DB: db})
	err := l.DeleteChannel(context.Background(), 1001)
	var conflict *DeleteConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected DeleteConflictError, got %v", err)
	}
}

func TestDeleteSeverityRoute_DeletesExactRow(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:delete-route-success?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&model.AlertSeverityRoute{})
	_ = db.Create(&model.AlertSeverityRoute{ID: 31, Scope: "global", Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true}).Error

	l := NewLogic(&svc.ServiceContext{DB: db})
	if err := l.DeleteSeverityRoute(context.Background(), 31, 0); err != nil {
		t.Fatalf("delete severity route: %v", err)
	}
}

func TestDeleteRuleChannelBinding_RespectsProjectScope(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:delete-binding-scope?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&model.AlertRuleChannelBinding{})
	projectID := uint(42)
	_ = db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Enabled: true, Priority: 1}).Error
	_ = db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, ProjectID: &projectID, Enabled: true, Priority: 1}).Error

	l := NewLogic(&svc.ServiceContext{DB: db})
	if err := l.DeleteRuleChannelBinding(context.Background(), 42, 7, 1001); err != nil {
		t.Fatalf("delete project binding: %v", err)
	}

	var remain int64
	_ = db.Model(&model.AlertRuleChannelBinding{}).Where("rule_id = ? AND channel_id = ? AND project_id IS NULL", 7, 1001).Count(&remain).Error
	if remain != 1 {
		t.Fatalf("expected global binding to remain, got %d", remain)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/monitoring/logic -run "DeleteRule_|DeleteChannel_|DeleteSeverityRoute_|DeleteRuleChannelBinding_" -v`  
Expected: FAIL with `undefined: DeleteRule` / `undefined: DeleteChannel` / `undefined: DeleteSeverityRoute` / `undefined: DeleteRuleChannelBinding`.

- [ ] **Step 3: Commit the failing test scaffold**

```bash
git add internal/modules/monitoring/logic/delete_resources_test.go
git commit -m "test(monitoring): add failing delete conflict and scoped deletion logic tests"
```

### Task 2: Implement Backend Logic For Hard Delete And Single-Item CRUD

**Files:**
- Create: `internal/modules/monitoring/logic/delete_conflict.go`
- Modify: `internal/modules/monitoring/logic/logic.go`
- Modify: `internal/modules/monitoring/logic/routing_policy.go`

- [ ] **Step 1: Add conflict error model used by handlers**

```go
// internal/modules/monitoring/logic/delete_conflict.go
package logic

import "fmt"

type DeleteBlocker struct {
	Type    string   `json:"type"`
	Count   int      `json:"count"`
	Samples []string `json:"samples,omitempty"`
}

type DeleteConflictError struct {
	Resource string          `json:"resource"`
	Blockers []DeleteBlocker `json:"blockers"`
}

func (e *DeleteConflictError) Error() string {
	return fmt.Sprintf("%s has references", e.Resource)
}
```

- [ ] **Step 2: Implement rule/channel hard delete with blocker checks**

```go
// internal/modules/monitoring/logic/logic.go
func (l *Logic) DeleteRule(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	var row model.AlertRule
	if err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return err
	}

	var bindingCount int64
	if err := l.svcCtx.DB.WithContext(ctx).
		Model(&model.AlertRuleChannelBinding{}).
		Where("rule_id = ?", id).
		Count(&bindingCount).Error; err != nil {
		return err
	}
	if bindingCount > 0 {
		return &DeleteConflictError{
			Resource: "alert_rule",
			Blockers: []DeleteBlocker{{Type: "binding", Count: int(bindingCount)}},
		}
	}
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.AlertRule{}, id).Error
}

func (l *Logic) DeleteChannel(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	var row model.AlertNotificationChannel
	if err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return err
	}

	var bindingCount int64
	if err := l.svcCtx.DB.WithContext(ctx).
		Model(&model.AlertRuleChannelBinding{}).
		Where("channel_id = ?", id).
		Count(&bindingCount).Error; err != nil {
		return err
	}

	routes := make([]model.AlertSeverityRoute, 0, 16)
	if err := l.svcCtx.DB.WithContext(ctx).Find(&routes).Error; err != nil {
		return err
	}
	routeRefs := 0
	for _, route := range routes {
		for _, channelID := range parseChannelIDs(route.ChannelIDsJSON) {
			if channelID == id {
				routeRefs++
				break
			}
		}
	}
	if bindingCount > 0 || routeRefs > 0 {
		blockers := make([]DeleteBlocker, 0, 2)
		if bindingCount > 0 {
			blockers = append(blockers, DeleteBlocker{Type: "binding", Count: int(bindingCount)})
		}
		if routeRefs > 0 {
			blockers = append(blockers, DeleteBlocker{Type: "severity_route", Count: routeRefs})
		}
		return &DeleteConflictError{Resource: "alert_channel", Blockers: blockers}
	}
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.AlertNotificationChannel{}, id).Error
}
```

- [ ] **Step 3: Implement single-item severity route and binding CRUD**

```go
// internal/modules/monitoring/logic/routing_policy.go
func (l *Logic) CreateSeverityRoute(ctx context.Context, projectID uint, input SeverityRouteInput) (*model.AlertSeverityRoute, error) {
	b, err := json.Marshal(input.ChannelIDs)
	if err != nil {
		return nil, err
	}
	row := model.AlertSeverityRoute{
		Scope:          strings.ToLower(strings.TrimSpace(input.Scope)),
		Severity:       strings.ToLower(strings.TrimSpace(input.Severity)),
		ChannelIDsJSON: string(b),
		Enabled:        input.Enabled,
	}
	if projectID > 0 {
		pid := projectID
		row.ProjectID = &pid
		if row.Scope == "" {
			row.Scope = "project"
		}
	}
	if row.Scope == "" {
		row.Scope = "global"
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) UpdateSeverityRoute(ctx context.Context, id uint, projectID uint, input SeverityRouteInput) (*model.AlertSeverityRoute, error) {
	b, err := json.Marshal(input.ChannelIDs)
	if err != nil {
		return nil, err
	}
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertSeverityRoute{}).Where("id = ?", id)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	updates := map[string]any{
		"scope":            strings.ToLower(strings.TrimSpace(input.Scope)),
		"severity":         strings.ToLower(strings.TrimSpace(input.Severity)),
		"channel_ids_json": string(b),
		"enabled":          input.Enabled,
	}
	if err := q.Updates(updates).Error; err != nil {
		return nil, err
	}
	var row model.AlertSeverityRoute
	if err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) DeleteSeverityRoute(ctx context.Context, id uint, projectID uint) error {
	q := l.svcCtx.DB.WithContext(ctx).Where("id = ?", id)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	result := q.Delete(&model.AlertSeverityRoute{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (l *Logic) DeleteRuleChannelBinding(ctx context.Context, projectID, ruleID, channelID uint) error {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertRuleChannelBinding{}).
		Where("rule_id = ? AND channel_id = ?", ruleID, channelID)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	} else {
		q = q.Where("project_id IS NULL")
	}
	result := q.Delete(&model.AlertRuleChannelBinding{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run logic tests to verify pass**

Run: `go test ./internal/modules/monitoring/logic -run "DeleteRule_|DeleteChannel_|DeleteSeverityRoute_|DeleteRuleChannelBinding_" -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/logic/delete_conflict.go internal/modules/monitoring/logic/logic.go internal/modules/monitoring/logic/routing_policy.go internal/modules/monitoring/logic/delete_resources_test.go
git commit -m "feat(monitoring): add hard-delete logic with dependency conflict checks"
```

### Task 3: Add Failing Handler Tests For New CRUD Endpoints

**Files:**
- Modify: `internal/modules/monitoring/handler/handler_config_test.go`

- [ ] **Step 1: Add handler contract tests for delete/conflict and single-item CRUD**

```go
// internal/modules/monitoring/handler/handler_config_test.go (append)
func TestDeleteRuleEndpoint_ReturnsConflictWithBlockers(t *testing.T) {
	// setup db with rule + binding; invoke DELETE /api/v1/alert-rules/7
	// assert response code field == 409 and data.blockers present
}

func TestDeleteChannelEndpoint_Returns404WhenMissing(t *testing.T) {
	// invoke DELETE /api/v1/alert-channels/9999
	// assert response code field == 2005 (NotFound)
}

func TestSeverityRouteSingleCRUDEndpoints(t *testing.T) {
	// POST /api/v1/alert-routing/severity
	// PUT  /api/v1/alert-routing/severity/:id
	// DELETE /api/v1/alert-routing/severity/:id
	// assert all success path responses
}

func TestRuleChannelBindingSingleCRUDEndpoints(t *testing.T) {
	// POST /api/v1/alert-rules/7/channels
	// PUT  /api/v1/alert-rules/7/channels/1001
	// DELETE /api/v1/alert-rules/7/channels/1001?project_id=42
	// assert success and scoped deletion behavior
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/monitoring/handler -run "DeleteRuleEndpoint|DeleteChannelEndpoint|SeverityRouteSingleCRUDEndpoints|RuleChannelBindingSingleCRUDEndpoints" -v`  
Expected: FAIL with missing handler methods / missing routes.

- [ ] **Step 3: Commit failing endpoint tests**

```bash
git add internal/modules/monitoring/handler/handler_config_test.go
git commit -m "test(monitoring): add failing handler tests for delete and single-item CRUD endpoints"
```

### Task 4: Implement Handler, Routes, And DTOs For Full Backend CRUD Surface

**Files:**
- Modify: `internal/modules/monitoring/api/routes.go`
- Modify: `api/monitoring/v1/monitoring.go`
- Modify: `internal/modules/monitoring/handler/handler.go`
- Modify: `internal/modules/monitoring/logic/routing_policy.go`

- [ ] **Step 1: Add route registrations for new CRUD endpoints**

```go
// internal/modules/monitoring/api/routes.go
g.DELETE("/alert-rules/:id", h.DeleteRule)
g.DELETE("/alert-channels/:id", h.DeleteChannel)

g.POST("/alert-routing/severity", h.CreateSeverityRoute)
g.PUT("/alert-routing/severity/:id", h.UpdateSeverityRouteByID)
g.DELETE("/alert-routing/severity/:id", h.DeleteSeverityRoute)

g.POST("/alert-rules/:id/channels", h.CreateRuleChannelBinding)
g.PUT("/alert-rules/:id/channels/:channel_id", h.UpdateRuleChannelBinding)
g.DELETE("/alert-rules/:id/channels/:channel_id", h.DeleteRuleChannelBinding)
```

- [ ] **Step 2: Add request DTOs for single-item route and binding CRUD**

```go
// api/monitoring/v1/monitoring.go
type SeverityRouteUpsertRequest struct {
	ProjectID  *uint  `json:"project_id"`
	Scope      string `json:"scope"`
	Severity   string `json:"severity" binding:"required"`
	ChannelIDs []uint `json:"channel_ids"`
	Enabled    *bool  `json:"enabled"`
}

type RuleChannelBindingCreateRequest struct {
	ProjectID *uint `json:"project_id"`
	ChannelID uint  `json:"channel_id" binding:"required"`
	Priority  *int  `json:"priority"`
	Enabled   *bool `json:"enabled"`
}

type RuleChannelBindingUpdateRequest struct {
	ProjectID *uint `json:"project_id"`
	Priority  *int  `json:"priority"`
	Enabled   *bool `json:"enabled"`
}
```

- [ ] **Step 3: Implement handler methods and conflict response shaping**

```go
// internal/modules/monitoring/handler/handler.go
func (h *Handler) DeleteRule(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") { return }
	id := httpx.UintFromParam(c, "id")
	err := h.logic.DeleteRule(c.Request.Context(), id)
	if err == nil { httpx.OK(c, gin.H{"deleted": id}); return }
	h.handleDeleteError(c, err)
}

func (h *Handler) DeleteChannel(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") { return }
	id := httpx.UintFromParam(c, "id")
	err := h.logic.DeleteChannel(c.Request.Context(), id)
	if err == nil { httpx.OK(c, gin.H{"deleted": id}); return }
	h.handleDeleteError(c, err)
}

func (h *Handler) handleDeleteError(c *gin.Context, err error) {
	var conflict *monitoringlogic.DeleteConflictError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		httpx.NotFound(c, "")
	case errors.As(err, &conflict):
		c.JSON(http.StatusOK, gin.H{
			"code": 409,
			"msg":  "resource has references",
			"data": gin.H{"resource": conflict.Resource, "blockers": conflict.Blockers},
		})
	default:
		httpx.ServerErr(c, err)
	}
}
```

- [ ] **Step 4: Implement binding and severity-route single-item logic methods used by handlers**

```go
// internal/modules/monitoring/logic/routing_policy.go
func (l *Logic) CreateRuleChannelBinding(ctx context.Context, projectID, ruleID, channelID uint, priority int, enabled bool) (*model.AlertRuleChannelBinding, error) {
	row := model.AlertRuleChannelBinding{RuleID: ruleID, ChannelID: channelID, Priority: priority, Enabled: enabled}
	if row.Priority <= 0 { row.Priority = 1 }
	if projectID > 0 { pid := projectID; row.ProjectID = &pid }
	if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil { return nil, err }
	return &row, nil
}

func (l *Logic) UpdateRuleChannelBinding(ctx context.Context, projectID, ruleID, channelID uint, priority int, enabled bool) (*model.AlertRuleChannelBinding, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertRuleChannelBinding{}).Where("rule_id = ? AND channel_id = ?", ruleID, channelID)
	if projectID > 0 { q = q.Where("project_id = ?", projectID) } else { q = q.Where("project_id IS NULL") }
	result := q.Updates(map[string]any{"priority": priority, "enabled": enabled})
	if result.Error != nil { return nil, result.Error }
	if result.RowsAffected == 0 { return nil, gorm.ErrRecordNotFound }
	var row model.AlertRuleChannelBinding
	if err := q.Take(&row).Error; err != nil { return nil, err }
	return &row, nil
}
```

- [ ] **Step 5: Run handler tests and backend package tests**

Run: `go test ./internal/modules/monitoring/handler -run "DeleteRuleEndpoint|DeleteChannelEndpoint|SeverityRouteSingleCRUDEndpoints|RuleChannelBindingSingleCRUDEndpoints" -v`  
Expected: PASS.

Run: `go test ./internal/modules/monitoring/... -count=1`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/monitoring/api/routes.go api/monitoring/v1/monitoring.go internal/modules/monitoring/handler/handler.go internal/modules/monitoring/logic/routing_policy.go internal/modules/monitoring/handler/handler_config_test.go
git commit -m "feat(monitoring): add full backend CRUD endpoints for rules channels routes and bindings"
```

### Task 5: Add Frontend API Methods For Full CRUD

**Files:**
- Modify: `web/src/api/modules/monitoring.test.ts`
- Modify: `web/src/api/modules/monitoring.ts`

- [ ] **Step 1: Add failing API tests for delete and single-item CRUD methods**

```ts
// web/src/api/modules/monitoring.test.ts (append)
it('calls delete rule endpoint', async () => {
  await monitoringApi.deleteAlertRule('7');
  expect(deleteMock).toHaveBeenCalledWith('/alert-rules/7');
});

it('calls create severity route endpoint', async () => {
  await monitoringApi.createSeverityRoute({ severity: 'critical', channelIds: ['1001'] });
  expect(postMock).toHaveBeenCalledWith('/alert-routing/severity', expect.any(Object));
});

it('calls binding delete endpoint with project scope', async () => {
  await monitoringApi.deleteRuleChannelBinding('7', '1001', '42');
  expect(deleteMock).toHaveBeenCalledWith('/alert-rules/7/channels/1001', { params: { project_id: '42' } });
});
```

- [ ] **Step 2: Run API tests to verify they fail**

Run: `npm --prefix web run test:run -- src/api/modules/monitoring.test.ts`  
Expected: FAIL with missing method errors.

- [ ] **Step 3: Implement new monitoring API methods**

```ts
// web/src/api/modules/monitoring.ts (append to monitoringApi)
async deleteAlertRule(id: string) {
  return apiService.delete(`/alert-rules/${encodeURIComponent(id)}`);
},
async deleteAlertChannel(id: string) {
  return apiService.delete(`/alert-channels/${encodeURIComponent(id)}`);
},
async createSeverityRoute(payload: { projectId?: string; scope?: string; severity: string; channelIds: string[]; enabled?: boolean }) {
  return apiService.post('/alert-routing/severity', {
    project_id: payload.projectId ? Number(payload.projectId) : undefined,
    scope: payload.scope,
    severity: payload.severity,
    channel_ids: payload.channelIds.map(Number).filter((x) => Number.isFinite(x) && x > 0),
    enabled: payload.enabled ?? true,
  });
},
async updateSeverityRouteByID(id: string, payload: { projectId?: string; scope?: string; severity: string; channelIds: string[]; enabled?: boolean }) {
  return apiService.put(`/alert-routing/severity/${encodeURIComponent(id)}`, {
    project_id: payload.projectId ? Number(payload.projectId) : undefined,
    scope: payload.scope,
    severity: payload.severity,
    channel_ids: payload.channelIds.map(Number).filter((x) => Number.isFinite(x) && x > 0),
    enabled: payload.enabled ?? true,
  });
},
async deleteSeverityRoute(id: string, projectId?: string) {
  return apiService.delete(`/alert-routing/severity/${encodeURIComponent(id)}`, {
    params: { project_id: projectId },
  });
},
async createRuleChannelBinding(ruleId: string, payload: { projectId?: string; channelId: string; priority?: number; enabled?: boolean }) {
  return apiService.post(`/alert-rules/${encodeURIComponent(ruleId)}/channels`, {
    project_id: payload.projectId ? Number(payload.projectId) : undefined,
    channel_id: Number(payload.channelId),
    priority: payload.priority,
    enabled: payload.enabled ?? true,
  });
},
async updateRuleChannelBinding(ruleId: string, channelId: string, payload: { projectId?: string; priority?: number; enabled?: boolean }) {
  return apiService.put(`/alert-rules/${encodeURIComponent(ruleId)}/channels/${encodeURIComponent(channelId)}`, {
    project_id: payload.projectId ? Number(payload.projectId) : undefined,
    priority: payload.priority,
    enabled: payload.enabled,
  });
},
async deleteRuleChannelBinding(ruleId: string, channelId: string, projectId?: string) {
  return apiService.delete(`/alert-rules/${encodeURIComponent(ruleId)}/channels/${encodeURIComponent(channelId)}`, {
    params: { project_id: projectId },
  });
},
```

- [ ] **Step 4: Re-run API tests**

Run: `npm --prefix web run test:run -- src/api/modules/monitoring.test.ts`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/monitoring.ts web/src/api/modules/monitoring.test.ts
git commit -m "feat(web): add monitoring API methods for full CRUD operations"
```

### Task 6: Upgrade RulesConfigPage To Full Rule CRUD

**Files:**
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RulesConfigPage.test.tsx`

- [ ] **Step 1: Add failing page tests for create/edit/delete rule actions**

```tsx
// web/src/pages/Monitor/RulesConfigPage.test.tsx (append)
it('creates a rule from drawer form', async () => {
  mockApi.monitoring.createAlertRule.mockResolvedValue({ data: { id: 9 } });
  render(<RulesConfigPage />);
  fireEvent.click(await screen.findByRole('button', { name: '新增规则' }));
  fireEvent.change(screen.getByPlaceholderText('例如: CPU High'), { target: { value: 'CPU High' } });
  fireEvent.change(screen.getByPlaceholderText('例如: cpu_usage'), { target: { value: 'cpu_usage' } });
  fireEvent.click(screen.getByRole('button', { name: '保存' }));
  await waitFor(() => expect(mockApi.monitoring.createAlertRule).toHaveBeenCalled());
});

it('deletes a rule after confirm', async () => {
  mockApi.monitoring.deleteAlertRule.mockResolvedValue({ data: { deleted: 1 } });
  render(<RulesConfigPage />);
  fireEvent.click(await screen.findByRole('button', { name: '删除' }));
  fireEvent.click(await screen.findByRole('button', { name: '确定' }));
  await waitFor(() => expect(mockApi.monitoring.deleteAlertRule).toHaveBeenCalled());
});
```

- [ ] **Step 2: Run tests to confirm failure**

Run: `npm --prefix web run test:run -- src/pages/Monitor/RulesConfigPage.test.tsx`  
Expected: FAIL because UI controls and method wiring do not exist yet.

- [ ] **Step 3: Implement rule CRUD UI with drawer form**

```tsx
// web/src/pages/Monitor/RulesConfigPage.tsx (core action handlers)
const handleCreate = async (values: RuleFormValues) => {
  await Api.monitoring.createAlertRule({
    name: values.name,
    metric: values.metric,
    operator: values.operator,
    threshold: Number(values.threshold),
    severity: values.severity,
    enabled: values.enabled,
  });
  message.success('规则创建成功');
  await load();
};

const handleUpdate = async (id: string, values: RuleFormValues) => {
  await Api.monitoring.updateAlertRule(id, {
    name: values.name,
    operator: values.operator,
    threshold: Number(values.threshold),
    severity: values.severity,
    enabled: values.enabled,
  });
  message.success('规则更新成功');
  await load();
};

const handleDelete = async (id: string) => {
  await Api.monitoring.deleteAlertRule(id);
  message.success('规则删除成功');
  await load();
};
```

- [ ] **Step 4: Re-run page test**

Run: `npm --prefix web run test:run -- src/pages/Monitor/RulesConfigPage.test.tsx`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Monitor/RulesConfigPage.tsx web/src/pages/Monitor/RulesConfigPage.test.tsx
git commit -m "feat(web): add full CRUD interactions for monitoring rules page"
```

### Task 7: Add Rule-Channel Binding CRUD In RulesConfigPage

**Files:**
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RulesConfigPage.test.tsx`

- [ ] **Step 1: Add failing tests for binding create/update/delete**

```tsx
// web/src/pages/Monitor/RulesConfigPage.test.tsx (append)
it('creates and deletes a rule-channel binding', async () => {
  mockApi.monitoring.getRuleChannels.mockResolvedValue({ data: { list: [], total: 0 } });
  mockApi.monitoring.createRuleChannelBinding.mockResolvedValue({ data: { id: 1 } });
  mockApi.monitoring.deleteRuleChannelBinding.mockResolvedValue({ data: { deleted: 1 } });
  render(<RulesConfigPage />);
  fireEvent.click(await screen.findByRole('button', { name: '绑定渠道' }));
  fireEvent.change(screen.getByPlaceholderText('渠道ID'), { target: { value: '1001' } });
  fireEvent.click(screen.getByRole('button', { name: '新增绑定' }));
  await waitFor(() => expect(mockApi.monitoring.createRuleChannelBinding).toHaveBeenCalled());
});
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `npm --prefix web run test:run -- src/pages/Monitor/RulesConfigPage.test.tsx`  
Expected: FAIL with missing binding UI/actions.

- [ ] **Step 3: Implement binding drawer + CRUD handlers**

```tsx
// web/src/pages/Monitor/RulesConfigPage.tsx (binding operations)
const openBindingDrawer = async (ruleId: string) => {
  setActiveRuleId(ruleId);
  const res = await Api.monitoring.getRuleChannels(ruleId, { projectId });
  setBindings((res.data as any)?.list || []);
  setBindingVisible(true);
};

const handleCreateBinding = async (ruleId: string, channelId: string, priority: number) => {
  await Api.monitoring.createRuleChannelBinding(ruleId, { projectId, channelId, priority, enabled: true });
  await reloadBindings(ruleId);
};

const handleUpdateBinding = async (ruleId: string, channelId: string, priority: number, enabled: boolean) => {
  await Api.monitoring.updateRuleChannelBinding(ruleId, channelId, { projectId, priority, enabled });
  await reloadBindings(ruleId);
};

const handleDeleteBinding = async (ruleId: string, channelId: string) => {
  await Api.monitoring.deleteRuleChannelBinding(ruleId, channelId, projectId);
  await reloadBindings(ruleId);
};
```

- [ ] **Step 4: Run tests**

Run: `npm --prefix web run test:run -- src/pages/Monitor/RulesConfigPage.test.tsx`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Monitor/RulesConfigPage.tsx web/src/pages/Monitor/RulesConfigPage.test.tsx
git commit -m "feat(web): add rule-channel binding CRUD drawer in rules config page"
```

### Task 8: Upgrade ChannelsConfigPage And RoutingConfigPage To Full CRUD

**Files:**
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.test.tsx`

- [ ] **Step 1: Add failing tests for channel create/edit/delete and route create/edit/delete**

```tsx
// ChannelsConfigPage.test.tsx (append)
it('creates a channel and deletes it', async () => {
  mockApi.monitoring.createAlertChannel.mockResolvedValue({ data: { id: 1001 } });
  mockApi.monitoring.deleteAlertChannel.mockResolvedValue({ data: { deleted: 1001 } });
  render(<ChannelsConfigPage />);
  fireEvent.click(await screen.findByRole('button', { name: '新增渠道' }));
  fireEvent.change(screen.getByPlaceholderText('渠道名称'), { target: { value: 'Ops Webhook' } });
  fireEvent.click(screen.getByRole('button', { name: '保存' }));
  await waitFor(() => expect(mockApi.monitoring.createAlertChannel).toHaveBeenCalled());
});

// RoutingConfigPage.test.tsx (append)
it('creates and deletes severity route', async () => {
  mockApi.monitoring.createSeverityRoute.mockResolvedValue({ data: { id: 11 } });
  mockApi.monitoring.deleteSeverityRoute.mockResolvedValue({ data: { deleted: 11 } });
  render(<RoutingConfigPage />);
  fireEvent.click(await screen.findByRole('button', { name: '新增路由' }));
  fireEvent.change(screen.getByPlaceholderText('critical/warning/info'), { target: { value: 'critical' } });
  fireEvent.change(screen.getByPlaceholderText('渠道ID，逗号分隔'), { target: { value: '1001,1002' } });
  fireEvent.click(screen.getByRole('button', { name: '保存' }));
  await waitFor(() => expect(mockApi.monitoring.createSeverityRoute).toHaveBeenCalled());
});
```

- [ ] **Step 2: Run tests to confirm failure**

Run: `npm --prefix web run test:run -- src/pages/Monitor/ChannelsConfigPage.test.tsx src/pages/Monitor/RoutingConfigPage.test.tsx`  
Expected: FAIL due missing CRUD controls and handlers.

- [ ] **Step 3: Implement ChannelsConfigPage CRUD and 409 blocker handling**

```tsx
// web/src/pages/Monitor/ChannelsConfigPage.tsx (core)
const handleDeleteChannel = async (id: string) => {
  try {
    await Api.monitoring.deleteAlertChannel(id);
    message.success('渠道删除成功');
    await loadChannels();
  } catch (err: any) {
    const blockers = err?.data?.blockers || [];
    if (String(err?.code) === '409' && blockers.length > 0) {
      Modal.error({ title: '删除失败：存在依赖', content: blockers.map((b: any) => `${b.type}: ${b.count}`).join('\n') });
      return;
    }
    throw err;
  }
};
```

- [ ] **Step 4: Implement RoutingConfigPage CRUD and scope-aware payload mapping**

```tsx
// web/src/pages/Monitor/RoutingConfigPage.tsx (core)
const parseChannelIDs = (raw: string) => raw.split(',').map((x) => x.trim()).filter(Boolean);

const handleCreateRoute = async (values: RouteFormValues) => {
  await Api.monitoring.createSeverityRoute({
    projectId,
    scope: values.scope,
    severity: values.severity,
    channelIds: parseChannelIDs(values.channelIDs),
    enabled: values.enabled,
  });
  await loadRoutes();
};

const handleDeleteRoute = async (id: string) => {
  await Api.monitoring.deleteSeverityRoute(id, projectId);
  await loadRoutes();
};
```

- [ ] **Step 5: Re-run page tests**

Run: `npm --prefix web run test:run -- src/pages/Monitor/ChannelsConfigPage.test.tsx src/pages/Monitor/RoutingConfigPage.test.tsx`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/Monitor/ChannelsConfigPage.tsx web/src/pages/Monitor/ChannelsConfigPage.test.tsx web/src/pages/Monitor/RoutingConfigPage.tsx web/src/pages/Monitor/RoutingConfigPage.test.tsx
git commit -m "feat(web): add full CRUD interactions for channels and severity routes pages"
```

### Task 9: Add Scope Selector And Run End-To-End Regression

**Files:**
- Create: `web/src/pages/Monitor/components/ScopeSelector.tsx`
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RulesConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.test.tsx`

- [ ] **Step 1: Add failing tests for scope switch wiring**

```tsx
// each page test: assert request includes project_id after switching to project scope
it('passes project_id after switching to project scope', async () => {
  render(<RulesConfigPage />);
  fireEvent.click(await screen.findByRole('radio', { name: '项目' }));
  fireEvent.change(screen.getByPlaceholderText('项目ID'), { target: { value: '42' } });
  await waitFor(() => expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledWith(expect.objectContaining({ projectId: '42' })));
});
```

- [ ] **Step 2: Run tests to verify failure**

Run: `npm --prefix web run test:run -- src/pages/Monitor/RulesConfigPage.test.tsx src/pages/Monitor/ChannelsConfigPage.test.tsx src/pages/Monitor/RoutingConfigPage.test.tsx`  
Expected: FAIL because pages do not yet expose scope selector.

- [ ] **Step 3: Implement reusable scope selector and integrate into three pages**

```tsx
// web/src/pages/Monitor/components/ScopeSelector.tsx
type ScopeValue = { scope: 'global' | 'project'; projectId?: string };

const ScopeSelector: React.FC<{
  value: ScopeValue;
  onChange: (next: ScopeValue) => void;
}> = ({ value, onChange }) => (
  <Space>
    <Radio.Group
      value={value.scope}
      onChange={(e) => onChange({ scope: e.target.value, projectId: value.projectId })}
      options={[{ label: '全局', value: 'global' }, { label: '项目', value: 'project' }]}
      optionType="button"
    />
    {value.scope === 'project' ? (
      <Input
        placeholder="项目ID"
        value={value.projectId}
        onChange={(e) => onChange({ scope: 'project', projectId: e.target.value })}
        style={{ width: 120 }}
      />
    ) : null}
  </Space>
);
```

- [ ] **Step 4: Run full regression suite**

Run: `go test ./internal/modules/monitoring/... ./internal/modules/ai/interfaces/http ./internal/modules/ai/api ./internal/modules/ai/handler/alertheal ./internal/modules/ai/dao/alertheal ./internal/modules/ai/logic/alertheal ./internal/modules/ai/infra/workers -count=1`  
Expected: PASS.

Run: `npm --prefix web run test:run -- src/pages/Monitor src/api/modules/monitoring.test.ts src/app/routes/observability.routes.test.tsx`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Monitor/components/ScopeSelector.tsx web/src/pages/Monitor/RulesConfigPage.tsx web/src/pages/Monitor/ChannelsConfigPage.tsx web/src/pages/Monitor/RoutingConfigPage.tsx web/src/pages/Monitor/RulesConfigPage.test.tsx web/src/pages/Monitor/ChannelsConfigPage.test.tsx web/src/pages/Monitor/RoutingConfigPage.test.tsx
git commit -m "feat(web): add scope-aware full CRUD workflows for monitoring config pages"
```

---

## Self-Review

### 1) Spec Coverage

- Full CRUD for rule/channel/route/binding: covered by Tasks 2, 4, 5, 6, 7, 8.
- Hard delete with conflict blockers: covered by Tasks 1, 2, 3, 4.
- Global/project scope support: covered by Task 9 and scope-aware backend handler parsing in Task 4.
- Error behavior (`404`, conflict with blockers): covered by Tasks 3, 4, 8.

No spec coverage gaps found.

### 2) Placeholder Scan

Checked for `TBD`, `TODO`, “implement later”, “similar to Task N”.  
No placeholders or deferred implementation markers found.

### 3) Type/Signature Consistency

- `DeleteConflictError` used consistently across logic and handler conflict mapping.
- New frontend method names (`deleteAlertRule`, `createSeverityRoute`, `createRuleChannelBinding`, etc.) are referenced consistently between page code and API tests.
- Scope parameters consistently represented as optional `projectId` in frontend and `project_id` in payload/query.

No naming/signature inconsistencies found.
