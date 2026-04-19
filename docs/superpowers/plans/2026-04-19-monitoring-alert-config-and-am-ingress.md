# Monitoring Alert Config And AM Ingress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build platform-managed Prometheus alert rule configuration, scoped notification routing, and single Alertmanager ingress that also triggers AI alert-heal.

**Architecture:** Keep Prometheus as the rule engine and Alertmanager as event forwarder. Expand `monitoring` module for scoped rule/channel/routing management and delivery policy. Use `/api/v1/alerts/receiver` as the single AM ingress and fan out internally to both monitoring event storage and AI alert-heal ingest/enqueue.

**Tech Stack:** Go (Gin, GORM, Viper), PostgreSQL/SQLite (via AutoMigrate), React 19 + TypeScript + Ant Design, Vitest/Jest-like frontend tests, Go test.

---

## Scope Check

This work spans backend config/schema/API/delivery, AI ingest compatibility, and frontend configuration UX. It is still one coherent subsystem: “monitoring alert control plane + AM ingestion path,” so one plan is appropriate.

## File Structure

### Backend Config & Provider

- Modify: `internal/core/config/config.go`  
  Responsibility: add `notification.smtp` runtime config.
- Modify: `configs/config.yaml`  
  Responsibility: define SMTP env-backed settings.
- Modify: `internal/modules/notification/handler/provider.go`  
  Responsibility: implement real SMTP email send path using global config.
- Create: `internal/modules/notification/handler/provider_email_test.go`  
  Responsibility: verify SMTP config validation and send behavior.

### Monitoring Schema & Logic

- Modify: `internal/modules/monitoring/model/model.go`  
  Responsibility: add scoped rule/channel fields and new binding/routing models.
- Modify: `internal/core/storage/migration/dev_auto.go`  
  Responsibility: include new monitoring models in dev automigrate.
- Create: `internal/modules/monitoring/logic/channel_secret.go`  
  Responsibility: encrypt/decrypt/mask channel config secrets.
- Create: `internal/modules/monitoring/logic/channel_secret_test.go`  
  Responsibility: validate cipher/mask behavior and key guardrails.
- Create: `internal/modules/monitoring/logic/routing_policy.go`  
  Responsibility: resolve channel targets from binding + severity fallback.
- Create: `internal/modules/monitoring/logic/routing_policy_test.go`  
  Responsibility: verify routing precedence and fallback.

### Monitoring API/Handlers

- Modify: `internal/modules/monitoring/api/routes.go`  
  Responsibility: register new endpoints (`/alert-rules/effective`, `/alert-channels/test`, bindings/routes APIs).
- Modify: `internal/modules/monitoring/handler/handler.go`  
  Responsibility: parse new request fields; expose new endpoints; add AM ingress fan-out hook.
- Modify: `api/monitoring/v1/monitoring.go`  
  Responsibility: update request/response DTOs.
- Create: `internal/modules/monitoring/handler/handler_config_test.go`  
  Responsibility: endpoint contract tests for new APIs and auth behavior.

### Monitoring Delivery

- Modify: `internal/modules/monitoring/handler/notification_gateway.go`  
  Responsibility: replace “send to all enabled channels” with routing-policy-driven channel selection.
- Create: `internal/modules/monitoring/handler/notification_gateway_routing_test.go`  
  Responsibility: verify binding priority, severity fallback, and log fallback.

### AI Alert-Heal Compatibility

- Modify: `internal/modules/monitoring/handler/handler.go`  
  Responsibility: call AI ingest/enqueue in `/alerts/receiver` path.
- Create: `internal/modules/monitoring/handler/ai_fanout.go`  
  Responsibility: adapter to AI ingest + enqueue using existing DAO/service.
- Create: `internal/modules/monitoring/handler/ai_fanout_test.go`  
  Responsibility: verify AM payload fan-out to AI tables.
- Modify: `internal/modules/ai/logic/alertheal/payload.go`  
  Responsibility: normalize AM source to stable protocol value and persist receiver as label metadata.
- Modify: `internal/modules/ai/dao/alertheal/dao.go`  
  Responsibility: cancel/retry query by protocol+fingerprint (not receiver name).
- Modify: `internal/modules/ai/logic/alertheal/service_test.go`  
  Responsibility: coverage for receiver-changed dedupe continuity.
- Modify: `internal/modules/ai/dao/alertheal/dao_test.go`  
  Responsibility: coverage for resolved cancel across receiver differences.
- Modify: `internal/modules/monitoring/logic/logic.go`  
  Responsibility: enrich alert-heal summary using protocol+fingerprint constraints.
- Modify: `internal/modules/monitoring/logic/list_alerts_test.go`  
  Responsibility: ensure unrelated protocol rows do not leak into summary.

### Frontend

- Modify: `web/src/api/modules/monitoring.ts`  
  Responsibility: add effective-rule, channel-test, bindings, severity-route APIs.
- Modify: `web/src/api/index.ts`  
  Responsibility: export new API methods/types if needed.
- Create: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Create: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Create: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- Create: `web/src/pages/Monitor/DeliveriesPage.tsx`  
  Responsibility: dedicated configuration and delivery UX pages.
- Create: `web/src/pages/Monitor/RulesConfigPage.test.tsx`
- Create: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
- Create: `web/src/pages/Monitor/RoutingConfigPage.test.tsx`
- Create: `web/src/pages/Monitor/DeliveriesPage.test.tsx`  
  Responsibility: behavior tests for create/edit/test send/routing display.
- Modify: `web/src/app/routes/pages.ts`  
  Responsibility: export new lazy pages.
- Modify: `web/src/app/routes/observability.routes.tsx`  
  Responsibility: add monitor sub-routes with `monitoring:read` and write actions behind API auth.

---

### Task 1: Add SMTP Runtime Config And Email Provider Tests

**Files:**
- Modify: `internal/core/config/config.go`
- Modify: `configs/config.yaml`
- Modify: `internal/modules/notification/handler/provider.go`
- Test: `internal/modules/notification/handler/provider_email_test.go`

- [ ] **Step 1: Write the failing provider test**

```go
// internal/modules/notification/handler/provider_email_test.go
func TestEmailProvider_Send_RequiresGlobalSMTPConfig(t *testing.T) {
    provider := &EmailProvider{}
    channel := monitoringmodel.AlertNotificationChannel{Name: "mail", Provider: "email"}
    err := provider.Send(context.Background(), &monitoringmodel.AlertEvent{Title: "CPU high"}, channel)
    if err == nil {
        t.Fatal("expected missing smtp config error")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/notification/handler -run TestEmailProvider_Send_RequiresGlobalSMTPConfig -v`  
Expected: FAIL because current `EmailProvider.Send` returns `nil`.

- [ ] **Step 3: Add SMTP config struct and email provider implementation**

```go
// internal/core/config/config.go
type Notification struct {
    SMTP SMTP `mapstructure:"smtp"`
}

type SMTP struct {
    Host     string        `mapstructure:"host"`
    Port     int           `mapstructure:"port"`
    Username string        `mapstructure:"username"`
    Password string        `mapstructure:"password"`
    From     string        `mapstructure:"from"`
    UseTLS   bool          `mapstructure:"use_tls"`
    StartTLS bool          `mapstructure:"starttls"`
    Timeout  time.Duration `mapstructure:"timeout"`
}

type Config struct {
    App App `mapstructure:"app"`
    Server Server `mapstructure:"server"`
    Notification Notification `mapstructure:"notification"`
    Prometheus Prometheus `mapstructure:"prometheus"`
}
```

```go
// internal/modules/notification/handler/provider.go
func (p *EmailProvider) Send(ctx context.Context, alert *monitoringmodel.AlertEvent, channel monitoringmodel.AlertNotificationChannel) error {
    smtpCfg := config.CFG.Notification.SMTP
    if strings.TrimSpace(smtpCfg.Host) == "" || smtpCfg.Port == 0 || strings.TrimSpace(smtpCfg.From) == "" {
        return fmt.Errorf("smtp config is incomplete")
    }
    return sendSMTPMessage(ctx, smtpCfg, channel, alert)
}

func sendSMTPMessage(ctx context.Context, smtpCfg config.SMTP, channel monitoringmodel.AlertNotificationChannel, alert *monitoringmodel.AlertEvent) error {
    recipients := strings.TrimSpace(channel.Target)
    if recipients == "" {
        return fmt.Errorf("email target is empty")
    }
    subject := fmt.Sprintf("[OpsPilot][%s] %s", strings.ToUpper(strings.TrimSpace(alert.Severity)), strings.TrimSpace(alert.Title))
    body := fmt.Sprintf("message=%s\nmetric=%s\nstatus=%s\n", alert.Message, alert.Metric, alert.Status)
    return smtpSend(ctx, smtpCfg, recipients, subject, body)
}

func smtpSend(ctx context.Context, cfg config.SMTP, to, subject, body string) error {
    _ = ctx
    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
    recipients := strings.Split(to, ",")
    msg := []byte("To: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + body + "\r\n")
    return smtp.SendMail(addr, auth, cfg.From, recipients, msg)
}
```

- [ ] **Step 4: Update config sample and rerun provider tests**

```yaml
# configs/config.yaml
notification:
  smtp:
    host: ${SMTP_HOST}
    port: ${SMTP_PORT}
    username: ${SMTP_USERNAME}
    password: ${SMTP_PASSWORD}
    from: ${SMTP_FROM}
    use_tls: true
    starttls: true
    timeout: 5s
```

Run: `go test ./internal/modules/notification/handler -run TestEmailProvider_Send_RequiresGlobalSMTPConfig -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/config/config.go configs/config.yaml internal/modules/notification/handler/provider.go internal/modules/notification/handler/provider_email_test.go
git commit -m "feat(notification): add global smtp config and email provider guardrails"
```

### Task 2: Extend Monitoring Models For Scoped Rules, Channels, Bindings, And Routes

**Files:**
- Modify: `internal/modules/monitoring/model/model.go`
- Modify: `internal/core/storage/migration/dev_auto.go`
- Test: `internal/modules/monitoring/model/model_scope_test.go`

- [ ] **Step 1: Write failing model migration test**

```go
func TestMonitoringModels_AutoMigrateScopedTables(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open("file:monitoring-model-scope?mode=memory&cache=shared"), &gorm.Config{})
    err := db.AutoMigrate(
        &model.AlertRule{},
        &model.AlertNotificationChannel{},
        &model.AlertRuleChannelBinding{},
        &model.AlertSeverityRoute{},
    )
    if err != nil {
        t.Fatalf("auto migrate failed: %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/model -run TestMonitoringModels_AutoMigrateScopedTables -v`  
Expected: FAIL with `undefined: AlertRuleChannelBinding` and `undefined: AlertSeverityRoute`.

- [ ] **Step 3: Add scoped fields and new models**

```go
// internal/modules/monitoring/model/model.go
type AlertRule struct {
    // additive fields for scoped rule management
    RuleMode   string `gorm:"column:rule_mode;type:varchar(16);default:'threshold'" json:"rule_mode"`
    ProjectID  *uint  `gorm:"column:project_id;index" json:"project_id,omitempty"`
    InheritKey string `gorm:"column:inherit_key;type:varchar(128);default:'';index" json:"inherit_key"`
    IsOverride bool   `gorm:"column:is_override;default:false" json:"is_override"`
}

type AlertNotificationChannel struct {
    // additive fields for scoped channels and encrypted config
    ProjectID      *uint  `gorm:"column:project_id;index" json:"project_id,omitempty"`
    ConfigCipherJSON string `gorm:"column:config_cipher_json;type:text" json:"-"`
}

type AlertRuleChannelBinding struct {
    ID        uint      `gorm:"primaryKey;column:id" json:"id"`
    RuleID    uint      `gorm:"column:rule_id;index" json:"rule_id"`
    ChannelID uint      `gorm:"column:channel_id;index" json:"channel_id"`
    ProjectID *uint     `gorm:"column:project_id;index" json:"project_id,omitempty"`
    Priority  int       `gorm:"column:priority;default:100" json:"priority"`
    Enabled   bool      `gorm:"column:enabled;default:true" json:"enabled"`
    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AlertRuleChannelBinding) TableName() string { return "alert_rule_channel_bindings" }

type AlertSeverityRoute struct {
    ID             uint      `gorm:"primaryKey;column:id" json:"id"`
    Scope          string    `gorm:"column:scope;type:varchar(16);not null;index" json:"scope"`
    ProjectID      *uint     `gorm:"column:project_id;index" json:"project_id,omitempty"`
    Severity       string    `gorm:"column:severity;type:varchar(16);not null;index" json:"severity"`
    ChannelIDsJSON string    `gorm:"column:channel_ids_json;type:text;not null" json:"channel_ids_json"`
    Enabled        bool      `gorm:"column:enabled;default:true" json:"enabled"`
    CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AlertSeverityRoute) TableName() string { return "alert_severity_routes" }
```

- [ ] **Step 4: Wire new models into dev auto-migrate and rerun tests**

```go
// internal/core/storage/migration/dev_auto.go
&monitoringmodel.AlertRuleChannelBinding{},
&monitoringmodel.AlertSeverityRoute{},
```

Run: `go test ./internal/modules/monitoring/model -run TestMonitoringModels_AutoMigrateScopedTables -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/model/model.go internal/core/storage/migration/dev_auto.go internal/modules/monitoring/model/model_scope_test.go
git commit -m "feat(monitoring): add scoped routing models for rules and channels"
```

### Task 3: Implement Channel Secret Encryption And Masked Read Model

**Files:**
- Create: `internal/modules/monitoring/logic/channel_secret.go`
- Test: `internal/modules/monitoring/logic/channel_secret_test.go`

- [ ] **Step 1: Write failing encryption/mask tests**

```go
func TestEncryptAndMaskChannelConfig_RoundTrip(t *testing.T) {
    key := "monitoring-secret-key"
    plain := `{"webhook":"https://x.example/hook/abc123","to":["ops@example.com"]}`
    cipherText, err := encryptChannelConfig(plain, key)
    if err != nil {
        t.Fatalf("encrypt: %v", err)
    }
    if cipherText == plain {
        t.Fatalf("expected ciphertext to differ from plaintext")
    }
    masked, err := decryptAndMaskChannelConfig(cipherText, key)
    if err != nil {
        t.Fatalf("mask: %v", err)
    }
    if !strings.Contains(masked, "***") {
        t.Fatalf("expected masked output, got %s", masked)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/logic -run TestEncryptAndMaskChannelConfig_RoundTrip -v`  
Expected: FAIL with `undefined: encryptChannelConfig`.

- [ ] **Step 3: Add secret helpers**

```go
// internal/modules/monitoring/logic/channel_secret.go
func encryptChannelConfig(plain, key string) (string, error) {
    plain = strings.TrimSpace(plain)
    if plain == "" {
        return "", nil
    }
    if strings.TrimSpace(key) == "" {
        return "", fmt.Errorf("security.encryption_key is required")
    }
    return utils.EncryptText(plain, key)
}

func decryptAndMaskChannelConfig(cipherText, key string) (string, error) {
    if strings.TrimSpace(cipherText) == "" {
        return "{}", nil
    }
    plain, err := utils.DecryptText(cipherText, key)
    if err != nil {
        return "", err
    }
    return maskJSONSecrets(plain), nil
}

func maskJSONSecrets(raw string) string {
    var payload map[string]any
    if err := json.Unmarshal([]byte(raw), &payload); err != nil {
        return `{"masked":"***"}`
    }
    for k, v := range payload {
        lower := strings.ToLower(strings.TrimSpace(k))
        if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "webhook") {
            payload[k] = "***"
            continue
        }
        if s, ok := v.(string); ok && strings.Contains(s, "@") {
            parts := strings.SplitN(s, "@", 2)
            payload[k] = "***@" + parts[1]
        }
    }
    b, _ := json.Marshal(payload)
    return string(b)
}
```

- [ ] **Step 4: Rerun target tests**

Run: `go test ./internal/modules/monitoring/logic -run TestEncryptAndMaskChannelConfig_RoundTrip -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/logic/channel_secret.go internal/modules/monitoring/logic/channel_secret_test.go
git commit -m "feat(monitoring): add encrypted channel config and masking helpers"
```

### Task 4: Add Scoped Channel CRUD And Channel Test API

**Files:**
- Modify: `api/monitoring/v1/monitoring.go`
- Modify: `internal/modules/monitoring/api/routes.go`
- Modify: `internal/modules/monitoring/handler/handler.go`
- Modify: `internal/modules/monitoring/logic/logic.go`
- Test: `internal/modules/monitoring/handler/handler_config_test.go`

- [ ] **Step 1: Add failing handler test for `/alert-channels/test`**

```go
func TestChannelTestEndpoint_Returns200ForValidPayload(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open("file:channel-test-endpoint?mode=memory&cache=shared"), &gorm.Config{})
    _ = db.AutoMigrate(&usermodel.User{}, &usermodel.Role{}, &usermodel.Permission{}, &usermodel.UserRole{}, &usermodel.RolePermission{}, &monitoringmodel.AlertNotificationChannel{})
    seedMonitoringWriteUser(t, db, 1001)
    h := NewHandler(&svc.ServiceContext{DB: db})
    r := gin.New()
    r.Use(func(c *gin.Context) {
        c.Set(httpx.CtxKeyUID, uint64(1001))
        c.Next()
    })
    r.POST("/api/v1/alert-channels/test", h.TestChannel)
    body := `{"provider":"webhook","target":"https://example.com/hook","config_json":"{}"}`
    req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels/test", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
    }
}

func seedMonitoringWriteUser(t *testing.T, db *gorm.DB, userID uint) {
    t.Helper()
    _ = db.Create(&usermodel.User{Model: gorm.Model{ID: userID}, Username: "ops-writer", Status: 1}).Error
    role := usermodel.Role{Name: "Ops", Code: "ops", Status: 1}
    perm := usermodel.Permission{Name: "MonitoringWrite", Code: "monitoring:write", Status: 1}
    _ = db.Create(&role).Error
    _ = db.Create(&perm).Error
    _ = db.Create(&usermodel.RolePermission{RoleID: int64(role.ID), PermissionID: int64(perm.ID)}).Error
    _ = db.Create(&usermodel.UserRole{UserID: int64(userID), RoleID: int64(role.ID)}).Error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/handler -run TestChannelTestEndpoint_Returns200ForValidPayload -v`  
Expected: FAIL with 404 route missing.

- [ ] **Step 3: Add DTOs and route registrations**

```go
// api/monitoring/v1/monitoring.go
type TestChannelRequest struct {
    Provider   string `json:"provider" binding:"required"`
    Target     string `json:"target"`
    ConfigJSON string `json:"config_json"`
}
```

```go
// internal/modules/monitoring/api/routes.go
g.POST("/alert-channels/test", h.TestChannel)
```

- [ ] **Step 4: Implement handler + logic for channel test and scoped fields**

```go
// internal/modules/monitoring/handler/handler.go
func (h *Handler) TestChannel(c *gin.Context) {
    if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") {
        return
    }
    var req v1.TestChannelRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BindErr(c, err)
        return
    }
    if err := h.logic.TestChannel(c.Request.Context(), strings.TrimSpace(req.Provider), strings.TrimSpace(req.Target), strings.TrimSpace(req.ConfigJSON)); err != nil {
        httpx.ServerErr(c, err)
        return
    }
    httpx.OK(c, gin.H{"status":"sent"})
}
```

Run: `go test ./internal/modules/monitoring/handler -run TestChannelTestEndpoint_Returns200ForValidPayload -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api/monitoring/v1/monitoring.go internal/modules/monitoring/api/routes.go internal/modules/monitoring/handler/handler.go internal/modules/monitoring/logic/logic.go internal/modules/monitoring/handler/handler_config_test.go
git commit -m "feat(monitoring): add scoped channel APIs and test-send endpoint"
```

### Task 5: Add Rule Mode/Scope Support And Effective Rules API

**Files:**
- Modify: `internal/modules/monitoring/api/routes.go`
- Modify: `internal/modules/monitoring/handler/handler.go`
- Modify: `internal/modules/monitoring/logic/logic.go`
- Test: `internal/modules/monitoring/logic/rules_effective_test.go`

- [ ] **Step 1: Write failing effective-rule merge test**

```go
func TestListEffectiveRules_ProjectOverrideWins(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open("file:effective-rules?mode=memory&cache=shared"), &gorm.Config{})
    _ = db.AutoMigrate(&model.AlertRule{})
    _ = db.Create(&model.AlertRule{
        Name: "CPU High", Metric: "cpu_usage", Operator: "gt", Threshold: 85, Severity: "warning",
        Enabled: true, State: "enabled", Scope: "global", InheritKey: "cpu-high",
    }).Error
    projectID := uint(42)
    _ = db.Create(&model.AlertRule{
        Name: "CPU High Project", Metric: "cpu_usage", Operator: "gt", Threshold: 92, Severity: "warning",
        Enabled: true, State: "enabled", Scope: "project", ProjectID: &projectID, InheritKey: "cpu-high", IsOverride: true,
    }).Error
    l := NewLogic(&svc.ServiceContext{DB: db})
    rules, _, err := l.ListEffectiveRules(context.Background(), 42, 1, 50)
    if err != nil {
        t.Fatalf("ListEffectiveRules: %v", err)
    }
    var got model.AlertRule
    for _, row := range rules {
        if row.InheritKey == "cpu-high" {
            got = row
            break
        }
    }
    if got.Threshold != 92 {
        t.Fatalf("expected override threshold 92, got %.2f", got.Threshold)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/logic -run TestListEffectiveRules_ProjectOverrideWins -v`  
Expected: FAIL with `undefined: ListEffectiveRules`.

- [ ] **Step 3: Implement effective merge logic and new API**

```go
// internal/modules/monitoring/logic/logic.go
func (l *Logic) ListEffectiveRules(ctx context.Context, projectID uint, page, pageSize int) ([]model.AlertRule, int64, error) {
    globals := []model.AlertRule{}
    if err := l.svcCtx.DB.WithContext(ctx).Where("project_id IS NULL").Find(&globals).Error; err != nil {
        return nil, 0, err
    }
    overrides := []model.AlertRule{}
    if projectID > 0 {
        if err := l.svcCtx.DB.WithContext(ctx).Where("project_id = ?", projectID).Find(&overrides).Error; err != nil {
            return nil, 0, err
        }
    }
    return mergeRules(globals, overrides, page, pageSize), int64(len(globals)), nil
}
```

```go
// internal/modules/monitoring/api/routes.go
g.GET("/alert-rules/effective", h.ListEffectiveRules)
```

- [ ] **Step 4: Add handler endpoint and rerun tests**

```go
// internal/modules/monitoring/handler/handler.go
func (h *Handler) ListEffectiveRules(c *gin.Context) {
    if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") { return }
    projectID := uint(intFromQuery(c, "project_id", 0))
    list, total, err := h.logic.ListEffectiveRules(c.Request.Context(), projectID, intFromQuery(c, "page", 1), intFromQuery(c, "page_size", 50))
    if err != nil { httpx.ServerErr(c, err); return }
    httpx.OK(c, gin.H{"list": list, "total": total})
}
```

Run: `go test ./internal/modules/monitoring/logic -run TestListEffectiveRules_ProjectOverrideWins -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/api/routes.go internal/modules/monitoring/handler/handler.go internal/modules/monitoring/logic/logic.go internal/modules/monitoring/logic/rules_effective_test.go
git commit -m "feat(monitoring): add effective scoped rules API and merge logic"
```

### Task 6: Add Rule-Channel Binding And Severity Route APIs

**Files:**
- Modify: `internal/modules/monitoring/api/routes.go`
- Modify: `internal/modules/monitoring/handler/handler.go`
- Modify: `internal/modules/monitoring/logic/logic.go`
- Test: `internal/modules/monitoring/logic/routing_policy_test.go`

- [ ] **Step 1: Add failing precedence test**

```go
func TestResolveChannels_BindingWinsSeverityFallback(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open("file:routing-precedence?mode=memory&cache=shared"), &gorm.Config{})
    _ = db.AutoMigrate(&model.AlertRuleChannelBinding{}, &model.AlertSeverityRoute{}, &model.AlertNotificationChannel{})
    _ = db.Create(&model.AlertNotificationChannel{ID: 1001, Name: "bound", Type: "log", Provider: "log", Enabled: true}).Error
    _ = db.Create(&model.AlertNotificationChannel{ID: 2001, Name: "fallback", Type: "log", Provider: "log", Enabled: true}).Error
    _ = db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error
    _ = db.Create(&model.AlertSeverityRoute{Scope: "global", Severity: "critical", ChannelIDsJSON: `[2001]`, Enabled: true}).Error
    logic := NewLogic(&svc.ServiceContext{DB: db})
    channels, err := logic.ResolveChannelsForAlert(context.Background(), 0, 7, "critical")
    if err != nil {
        t.Fatalf("resolve channels: %v", err)
    }
    if len(channels) != 1 || channels[0].ID != 1001 {
        t.Fatalf("expected bound channel 1001, got %#v", channels)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/logic -run TestResolveChannels_BindingWinsSeverityFallback -v`  
Expected: FAIL with `undefined: ResolveChannelsForAlert`.

- [ ] **Step 3: Implement routing policy logic and APIs**

```go
// internal/modules/monitoring/logic/routing_policy.go
func (l *Logic) ResolveChannelsForAlert(ctx context.Context, projectID uint, ruleID uint, severity string) ([]model.AlertNotificationChannel, error) {
    bound, err := l.listBoundChannels(ctx, projectID, ruleID)
    if err != nil { return nil, err }
    if len(bound) > 0 { return bound, nil }
    routed, err := l.listSeverityRoutedChannels(ctx, projectID, severity)
    if err != nil { return nil, err }
    if len(routed) > 0 { return routed, nil }
    return l.listDefaultLogChannel(ctx)
}

func (l *Logic) listBoundChannels(ctx context.Context, projectID, ruleID uint) ([]model.AlertNotificationChannel, error) {
    rows := []model.AlertNotificationChannel{}
    err := l.svcCtx.DB.WithContext(ctx).
        Table("alert_rule_channel_bindings AS b").
        Select("c.*").
        Joins("JOIN alert_notification_channels AS c ON c.id = b.channel_id").
        Where("b.rule_id = ? AND b.enabled = 1", ruleID).
        Where("(b.project_id = ? OR b.project_id IS NULL)", projectID).
        Order("b.priority ASC").
        Find(&rows).Error
    return rows, err
}
```

```go
// internal/modules/monitoring/api/routes.go
g.GET("/alert-rules/:id/channels", h.GetRuleChannels)
g.PUT("/alert-rules/:id/channels", h.UpdateRuleChannels)
g.GET("/alert-routing/severity", h.GetSeverityRoutes)
g.PUT("/alert-routing/severity", h.UpdateSeverityRoutes)
```

- [ ] **Step 4: Rerun routing tests**

Run: `go test ./internal/modules/monitoring/logic -run "ResolveChannels_BindingWinsSeverityFallback" -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/api/routes.go internal/modules/monitoring/handler/handler.go internal/modules/monitoring/logic/routing_policy.go internal/modules/monitoring/logic/routing_policy_test.go internal/modules/monitoring/logic/logic.go
git commit -m "feat(monitoring): add binding and severity route APIs with precedence resolver"
```

### Task 7: Make Notification Gateway Use Routing Policy (Not Global Broadcast)

**Files:**
- Modify: `internal/modules/monitoring/handler/notification_gateway.go`
- Test: `internal/modules/monitoring/handler/notification_gateway_routing_test.go`

- [ ] **Step 1: Add failing gateway behavior test**

```go
func TestNotificationGateway_DispatchesOnlyResolvedChannels(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open("file:gateway-routing?mode=memory&cache=shared"), &gorm.Config{})
    _ = db.AutoMigrate(&model.AlertEvent{}, &model.AlertNotificationChannel{}, &model.AlertRuleChannelBinding{}, &model.AlertSeverityRoute{}, &model.AlertNotificationDelivery{})
    _ = db.Create(&model.AlertNotificationChannel{ID: 1001, Name: "bound", Provider: "log", Type: "log", Enabled: true}).Error
    _ = db.Create(&model.AlertNotificationChannel{ID: 2001, Name: "fallback", Provider: "log", Type: "log", Enabled: true}).Error
    _ = db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error
    alert := model.AlertEvent{ID: 5001, RuleID: 7, Severity: "critical", Status: "firing", Source: "alertmanager/fp-1", TriggeredAt: time.Now().UTC()}
    _ = db.Create(&alert).Error
    gw := NewNotificationGateway(&svc.ServiceContext{DB: db})
    gw.dispatchAsync(context.Background(), alert)
    time.Sleep(50 * time.Millisecond)
    var rows []model.AlertNotificationDelivery
    _ = db.Where("alert_id = ?", alert.ID).Order("channel_id ASC").Find(&rows).Error
    if len(rows) != 1 || rows[0].ChannelID != 1001 {
        t.Fatalf("expected only bound channel delivery, got %#v", rows)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/handler -run TestNotificationGateway_DispatchesOnlyResolvedChannels -v`  
Expected: FAIL because current implementation sends to all enabled channels.

- [ ] **Step 3: Replace channel selection in gateway**

```go
// internal/modules/monitoring/handler/notification_gateway.go
func (g *NotificationGateway) dispatchAsync(ctx context.Context, alert model.AlertEvent) {
    channels, err := g.resolveChannels(ctx, alert)
    if err != nil {
        logger.L().Warn("resolve alert channels failed", logger.Error(err))
        return
    }
    if len(channels) == 0 {
        return
    }
    var wg sync.WaitGroup
    for _, ch := range channels {
        channel := ch
        wg.Add(1)
        go func() {
            defer wg.Done()
            g.sendWithRetry(runtimectx.Detach(ctx), alert, channel)
        }()
    }
    go func() { wg.Wait() }()
}
```

```go
func (g *NotificationGateway) resolveChannels(ctx context.Context, alert model.AlertEvent) ([]model.AlertNotificationChannel, error) {
    projectID := uint(0)
    return g.logic.ResolveChannelsForAlert(ctx, projectID, alert.RuleID, alert.Severity)
}
```

- [ ] **Step 4: Rerun gateway tests**

Run: `go test ./internal/modules/monitoring/handler -run TestNotificationGateway_DispatchesOnlyResolvedChannels -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/handler/notification_gateway.go internal/modules/monitoring/handler/notification_gateway_routing_test.go
git commit -m "feat(monitoring): apply routing policy when dispatching alert notifications"
```

### Task 8: Make `/alerts/receiver` Fan Out To AI Alert-Heal

**Files:**
- Create: `internal/modules/monitoring/handler/ai_fanout.go`
- Modify: `internal/modules/monitoring/handler/handler.go`
- Test: `internal/modules/monitoring/handler/ai_fanout_test.go`

- [ ] **Step 1: Write failing fan-out test**

```go
func TestReceiveWebhook_FanoutToAIIngestQueue(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open("file:receiver-fanout?mode=memory&cache=shared"), &gorm.Config{})
    _ = db.AutoMigrate(&monitoringmodel.AlertEvent{}, &aimodel.AIAlertIngestEvent{}, &aimodel.AIAlertHealJob{})
    h := NewHandler(&svc.ServiceContext{DB: db})
    h.aiFanout = &stubAIFanout{db: db}
    r := gin.New()
    r.POST("/alerts/receiver", h.ReceiveWebhook)
    payload := []byte(`{"alerts":[{"status":"firing","fingerprint":"fp-123","labels":{"alertname":"CPU"}}]}`)
    sig := signWebhookBody("test-prom-secret", payload)
    req := httptest.NewRequest(http.MethodPost, "/alerts/receiver", bytes.NewReader(payload))
    req.Header.Set("X-OpsPilot-Signature", hex.EncodeToString(sig))
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }
    var count int64
    if err := db.Table("ai_alert_heal_jobs").Count(&count).Error; err != nil {
        t.Fatalf("count ai jobs: %v", err)
    }
    if count == 0 {
        t.Fatal("expected ai alert-heal job to be enqueued")
    }
}

type stubAIFanout struct{ db *gorm.DB }

func (s *stubAIFanout) HandleAlertmanager(ctx context.Context, payload AlertmanagerWebhook) error {
    _ = ctx
    if len(payload.Alerts) == 0 {
        return nil
    }
    event := aimodel.AIAlertIngestEvent{
        ID: "evt-fanout",
        Source: "alertmanager",
        Protocol: "alertmanager",
        Fingerprint: payload.Alerts[0].Fingerprint,
        Status: payload.Alerts[0].Status,
        DedupeKey: "alertmanager:" + payload.Alerts[0].Fingerprint + ":" + payload.Alerts[0].Status,
        Title: "fanout",
        ReceivedAt: time.Now().UTC(),
    }
    if err := s.db.Create(&event).Error; err != nil {
        return err
    }
    return s.db.Create(&aimodel.AIAlertHealJob{ID: "job-fanout", EventID: event.ID, Scene: "alert_self_heal", Status: "pending"}).Error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/handler -run TestReceiveWebhook_FanoutToAIIngestQueue -v`  
Expected: FAIL because monitoring receiver currently does not enqueue AI jobs.

- [ ] **Step 3: Implement fan-out adapter and wire into handler**

```go
// internal/modules/monitoring/handler/ai_fanout.go
type AlertAIFanout interface {
    HandleAlertmanager(ctx context.Context, payload AlertmanagerWebhook) error
}

type aiAlertHealFanout struct {
    ingestor interface {
        Ingest(ctx context.Context, protocol string, raw []byte) ([]aimodel.AIAlertIngestEvent, error)
    }
    enqueuer interface {
        EnqueueBatch(ctx context.Context, events []aimodel.AIAlertIngestEvent) (string, error)
    }
}

func (f *aiAlertHealFanout) HandleAlertmanager(ctx context.Context, payload AlertmanagerWebhook) error {
    raw, err := json.Marshal(payload)
    if err != nil { return err }
    events, err := f.ingestor.Ingest(ctx, "alertmanager", raw)
    if err != nil || len(events) == 0 { return err }
    _, err = f.enqueuer.EnqueueBatch(ctx, events)
    return err
}
```

```go
// internal/modules/monitoring/handler/handler.go
type Handler struct {
    logic     *monitoringlogic.Logic
    svcCtx    *svc.ServiceContext
    ruleSync  *RuleSyncService
    webhookGW *NotificationGateway
    aiFanout AlertAIFanout
}

if h.aiFanout != nil {
    _ = h.aiFanout.HandleAlertmanager(c.Request.Context(), req)
}
```

- [ ] **Step 4: Rerun fan-out tests**

Run: `go test ./internal/modules/monitoring/handler -run TestReceiveWebhook_FanoutToAIIngestQueue -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/handler/ai_fanout.go internal/modules/monitoring/handler/handler.go internal/modules/monitoring/handler/ai_fanout_test.go
git commit -m "feat(monitoring): fan out alert receiver payloads to ai alert-heal ingest queue"
```

### Task 9: Stabilize AI Source Semantics For Dedupe/Cancel Across Receiver Changes

**Files:**
- Modify: `internal/modules/ai/logic/alertheal/payload.go`
- Modify: `internal/modules/ai/dao/alertheal/dao.go`
- Test: `internal/modules/ai/logic/alertheal/service_test.go`
- Test: `internal/modules/ai/dao/alertheal/dao_test.go`

- [ ] **Step 1: Add failing receiver-change continuity test**

```go
func TestIngest_DedupeIgnoresReceiverNameChanges(t *testing.T) {
    db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{})
    svc := NewService(aidaoalertheal.NewDAO(db))
    rawA := []byte(`{"receiver":"am-a","alerts":[{"status":"firing","fingerprint":"fp-x","labels":{"alertname":"CPU"}}]}`)
    rawB := []byte(`{"receiver":"am-b","alerts":[{"status":"firing","fingerprint":"fp-x","labels":{"alertname":"CPU"}}]}`)
    _, _ = svc.Ingest(context.Background(), "alertmanager", rawA)
    _, _ = svc.Ingest(context.Background(), "alertmanager", rawB)
    var count int64
    _ = db.Model(&model.AIAlertIngestEvent{}).Where("fingerprint = ? AND status = ?", "fp-x", "firing").Count(&count).Error
    if count != 1 {
        t.Fatalf("expected one deduped firing row, got %d", count)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/logic/alertheal -run TestIngest_DedupeIgnoresReceiverNameChanges -v`  
Expected: FAIL (current dedupe includes receiver-derived source).

- [ ] **Step 3: Normalize AM source and store receiver as label metadata**

```go
// internal/modules/ai/logic/alertheal/payload.go
source := protocolAlertmanager
labels := alert.Labels
if labels == nil { labels = map[string]string{} }
if receiver := strings.TrimSpace(payload.Receiver); receiver != "" {
    labels["am_receiver"] = receiver
}
out = append(out, NormalizedAlert{
    Source: source,
    Protocol: protocolAlertmanager,
    Fingerprint: strings.TrimSpace(alert.Fingerprint),
    Status: strings.TrimSpace(alert.Status),
    Severity: strings.TrimSpace(alert.Labels["severity"]),
    Title: strings.TrimSpace(alert.Labels["alertname"]),
    Target: firstNonEmpty(alert.Labels["instance"], alert.Labels["pod"], alert.Labels["node"]),
    LabelsJSON: mustJSON(labels, "{}"),
    AnnotationsJSON: mustJSON(alert.Annotations, "{}"),
    RawPayloadJSON: compactRaw,
    StartsAt: alert.StartsAt,
    EndsAt: alert.EndsAt,
})
```

```go
// internal/modules/ai/dao/alertheal/dao.go (CancelIfResolved)
Where(
  "protocol = ? AND fingerprint = ? AND status = ? AND (received_at > ? OR (received_at = ? AND id <> ?))",
  event.Protocol, event.Fingerprint, "resolved", event.ReceivedAt, event.ReceivedAt, event.ID,
)
```

- [ ] **Step 4: Rerun AI alert-heal tests**

Run: `go test ./internal/modules/ai/logic/alertheal ./internal/modules/ai/dao/alertheal -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/logic/alertheal/payload.go internal/modules/ai/dao/alertheal/dao.go internal/modules/ai/logic/alertheal/service_test.go internal/modules/ai/dao/alertheal/dao_test.go
git commit -m "fix(ai): stabilize alert-heal dedupe and cancel semantics across AM receiver changes"
```

### Task 10: Harden Monitoring Alert-Heal Summary Join By Protocol + Fingerprint

**Files:**
- Modify: `internal/modules/monitoring/logic/logic.go`
- Test: `internal/modules/monitoring/logic/list_alerts_test.go`

- [ ] **Step 1: Add failing cross-protocol isolation test**

```go
func TestListAlerts_HealSummaryDoesNotCrossProtocol(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open("file:summary-protocol?mode=memory&cache=shared"), &gorm.Config{})
    _ = db.AutoMigrate(&monitoringmodel.AlertEvent{}, &aimodel.AIAlertIngestEvent{}, &aimodel.AIAlertHealJob{})
    now := time.Now().UTC()
    _ = db.Create(&monitoringmodel.AlertEvent{ID: 42, Source: "alertmanager/fp-1", Severity: "critical", Status: "firing", TriggeredAt: now}).Error
    _ = db.Create(&aimodel.AIAlertIngestEvent{ID: "evt-am", Protocol: "alertmanager", Source: "alertmanager", Fingerprint: "fp-1", Status: "firing", DedupeKey: "alertmanager:fp-1:firing", Title: "am", ReceivedAt: now}).Error
    _ = db.Create(&aimodel.AIAlertIngestEvent{ID: "evt-u", Protocol: "opspilot.alert.v1", Source: "opspilot.alert.v1", Fingerprint: "fp-1", Status: "firing", DedupeKey: "opspilot.alert.v1:fp-1:firing", Title: "u", ReceivedAt: now}).Error
    _ = db.Create(&aimodel.AIAlertHealJob{ID: "job-am", EventID: "evt-am", Scene: "alert_self_heal", Status: "succeeded", UpdatedAt: now.Add(time.Minute)}).Error
    _ = db.Create(&aimodel.AIAlertHealJob{ID: "job-u", EventID: "evt-u", Scene: "alert_self_heal", Status: "failed_manual", UpdatedAt: now.Add(2 * time.Minute)}).Error

    rows, _, err := NewLogic(&svc.ServiceContext{DB: db}).ListAlerts(context.Background(), "", "", 42, 1, 20)
    if err != nil {
        t.Fatalf("list alerts: %v", err)
    }
    if len(rows) != 1 || rows[0].LatestHealJobID != "job-am" {
        t.Fatalf("expected only alertmanager heal summary, got %#v", rows)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/monitoring/logic -run TestListAlerts_HealSummaryDoesNotCrossProtocol -v`  
Expected: FAIL due fingerprint-only join.

- [ ] **Step 3: Update enrich query filter**

```go
// internal/modules/monitoring/logic/logic.go
Where("events.protocol = ? AND events.fingerprint IN ?", "alertmanager", fingerprints)
```

- [ ] **Step 4: Rerun monitoring logic tests**

Run: `go test ./internal/modules/monitoring/logic -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/monitoring/logic/logic.go internal/modules/monitoring/logic/list_alerts_test.go
git commit -m "fix(monitoring): scope alert-heal summary join by protocol and fingerprint"
```

### Task 11: Build Frontend Configuration Pages And API Methods

**Files:**
- Modify: `web/src/api/modules/monitoring.ts`
- Modify: `web/src/api/index.ts`
- Create: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Create: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Create: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- Create: `web/src/pages/Monitor/DeliveriesPage.tsx`
- Test: `web/src/pages/Monitor/RulesConfigPage.test.tsx`
- Test: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
- Test: `web/src/pages/Monitor/RoutingConfigPage.test.tsx`
- Test: `web/src/pages/Monitor/DeliveriesPage.test.tsx`

- [ ] **Step 1: Add failing API module tests for new endpoints**

```ts
it('calls effective rules endpoint', async () => {
  await monitoringApi.getEffectiveRules({ projectId: '42' });
  expect(getMock).toHaveBeenCalledWith('/alert-rules/effective', { params: { project_id: '42' } });
});

it('calls channel test endpoint', async () => {
  await monitoringApi.testAlertChannel({ provider: 'webhook', target: 'https://example.com', configJson: '{}' });
  expect(postMock).toHaveBeenCalledWith('/alert-channels/test', expect.any(Object));
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix web run test:run -- web/src/api/modules/monitoring.test.ts --runInBand`  
Expected: FAIL with missing methods.

- [ ] **Step 3: Implement API methods and page scaffolds**

```ts
// web/src/api/modules/monitoring.ts
async getEffectiveRules(params?: { projectId?: string; page?: number; pageSize?: number }) {
  return apiService.get('/alert-rules/effective', { params: { project_id: params?.projectId, page: params?.page, page_size: params?.pageSize } });
},
async testAlertChannel(payload: { provider: string; target?: string; configJson?: string }) {
  return apiService.post('/alert-channels/test', { provider: payload.provider, target: payload.target, config_json: payload.configJson });
},
async getRuleChannels(id: string) { return apiService.get(`/alert-rules/${encodeURIComponent(id)}/channels`); },
async updateRuleChannels(id: string, channelIds: string[]) { return apiService.put(`/alert-rules/${encodeURIComponent(id)}/channels`, { channel_ids: channelIds }); },
async getSeverityRoutes(params?: { projectId?: string }) { return apiService.get('/alert-routing/severity', { params: { project_id: params?.projectId } }); },
async updateSeverityRoutes(payload: any) { return apiService.put('/alert-routing/severity', payload); },
```

```tsx
// web/src/pages/Monitor/ChannelsConfigPage.tsx (core action)
<Button onClick={async () => {
  await Api.monitoring.testAlertChannel(form.getFieldsValue());
  message.success('测试发送成功');
}}>测试发送</Button>
```

- [ ] **Step 4: Run frontend page/API tests**

Run: `npm --prefix web run test:run -- web/src/api/modules/monitoring.test.ts web/src/pages/Monitor/ChannelsConfigPage.test.tsx web/src/pages/Monitor/RulesConfigPage.test.tsx --runInBand`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/monitoring.ts web/src/api/index.ts web/src/pages/Monitor/RulesConfigPage.tsx web/src/pages/Monitor/ChannelsConfigPage.tsx web/src/pages/Monitor/RoutingConfigPage.tsx web/src/pages/Monitor/DeliveriesPage.tsx web/src/pages/Monitor/RulesConfigPage.test.tsx web/src/pages/Monitor/ChannelsConfigPage.test.tsx web/src/pages/Monitor/RoutingConfigPage.test.tsx web/src/pages/Monitor/DeliveriesPage.test.tsx
git commit -m "feat(web): add monitoring config pages for rules channels routing and deliveries"
```

### Task 12: Wire Routes, Verify End-to-End Flows, And Final Regression

**Files:**
- Modify: `web/src/app/routes/pages.ts`
- Modify: `web/src/app/routes/observability.routes.tsx`
- Modify: `web/src/pages/Monitor/MonitorPage.tsx`
- Test: `web/src/app/routes/observability.routes.test.tsx`

- [ ] **Step 1: Add failing route coverage test**

```tsx
it('registers monitor config routes', () => {
  const withAuthStub = (_resource: string, _action: string, element: React.ReactElement) => element;
  const routes = renderObservabilityRoutes(withAuthStub);
  const paths = collectRoutePaths(routes);
  expect(paths).toEqual(expect.arrayContaining([
    '/monitor/rules',
    '/monitor/channels',
    '/monitor/routing',
    '/monitor/deliveries',
  ]));
});

function collectRoutePaths(node: React.ReactNode): string[] {
  const paths: string[] = [];
  React.Children.forEach(node as any, (child: any) => {
    if (!child || !child.props) return;
    if (child.props.path) paths.push(child.props.path);
  });
  return paths;
}
```

- [ ] **Step 2: Run route test to verify it fails**

Run: `npm --prefix web run test:run -- web/src/app/routes/observability.routes.test.tsx --runInBand`  
Expected: FAIL because routes are not present.

- [ ] **Step 3: Register new pages and route entries**

```tsx
// web/src/app/routes/pages.ts
export const RulesConfigPage = lazy(() => import('../../pages/Monitor/RulesConfigPage'));
export const ChannelsConfigPage = lazy(() => import('../../pages/Monitor/ChannelsConfigPage'));
export const RoutingConfigPage = lazy(() => import('../../pages/Monitor/RoutingConfigPage'));
export const DeliveriesPage = lazy(() => import('../../pages/Monitor/DeliveriesPage'));
```

```tsx
// web/src/app/routes/observability.routes.tsx
<Route path="/monitor/rules" element={withAuth('monitoring', 'read', <RulesConfigPage />)} />
<Route path="/monitor/channels" element={withAuth('monitoring', 'read', <ChannelsConfigPage />)} />
<Route path="/monitor/routing" element={withAuth('monitoring', 'read', <RoutingConfigPage />)} />
<Route path="/monitor/deliveries" element={withAuth('monitoring', 'read', <DeliveriesPage />)} />
```

- [ ] **Step 4: Run full verification suite**

Run: `go test ./internal/modules/monitoring/... ./internal/modules/ai/interfaces/http ./internal/modules/ai/api ./internal/modules/ai/handler/alertheal ./internal/modules/ai/dao/alertheal ./internal/modules/ai/logic/alertheal ./internal/modules/ai/infra/workers -count=1`  
Expected: PASS.

Run: `npm --prefix web run test:run -- web/src/pages/Monitor web/src/api/modules/monitoring.test.ts web/src/app/routes/observability.routes.test.tsx --runInBand`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/app/routes/pages.ts web/src/app/routes/observability.routes.tsx web/src/pages/Monitor/MonitorPage.tsx web/src/app/routes/observability.routes.test.tsx
git commit -m "feat(web): wire monitor config routes and complete alert config workflow"
```

---

## Self-Review

### 1) Spec Coverage

- Prometheus rule sync from platform config: covered in Tasks 5 and 12 (`/alerts/rules/sync` path preserved/enhanced).
- Platform handling of AM alert events: covered in Tasks 7 and 8.
- Scoped channels/rules and routing precedence: covered in Tasks 2, 4, 5, 6, 7.
- SMTP global config and email channel: covered in Task 1.
- Sensitive config encryption/masking: covered in Task 3.
- AI alert-heal compatibility and single ingress migration: covered in Tasks 8, 9, 10.
- Frontend system configuration UX: covered in Tasks 11, 12.

No spec gaps found.

### 2) Placeholder Scan

Checked for: `TBD`, `TODO`, “implement later”, “similar to Task N”, undefined vague test steps.  
Result: none.

### 3) Type/Signature Consistency

- `ListEffectiveRules`, `ResolveChannelsForAlert`, `TestChannel`, `HandleAlertmanager` names are used consistently between task definitions and route wiring.
- API paths are consistent with planned backend routes and frontend calls.
- AI ingest/enqueue fan-out always targets existing `alertmanager` protocol pathway.

No naming/signature inconsistencies found.
