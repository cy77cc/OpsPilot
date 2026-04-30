# OpsAgent Host Plugin Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a general host plugin framework in the platform, integrate `opsagent` as the first plugin, install it during host onboarding via SSH, ingest its metrics over gRPC, and force AI host execution through the plugin sandbox instead of SSH.

**Architecture:** Add a dedicated `hostplugin` domain for catalog, instance lifecycle, config revisions, and install tasks; add an `opsagent` runtime domain for gRPC registration, heartbeat, metrics, and remote execution; then re-route AI host execution to require an online plugin instance with matching capabilities. SSH remains the installation control plane only.

**Tech Stack:** Go, Gin, GORM, versioned SQL migrations, gRPC/protobuf, existing SSH client, React, TypeScript, Ant Design, Vitest, Go test

---

## File Structure

### Backend files to create

- `storage/migrations/20260429_0001_create_host_plugin_tables.sql`
- `internal/modules/hostplugin/model/plugin.go`
- `internal/modules/hostplugin/logic/service.go`
- `internal/modules/hostplugin/logic/catalog.go`
- `internal/modules/hostplugin/logic/install.go`
- `internal/modules/hostplugin/logic/config.go`
- `internal/modules/hostplugin/logic/task.go`
- `internal/modules/hostplugin/logic/service_test.go`
- `internal/modules/hostplugin/logic/install_test.go`
- `internal/modules/hostplugin/handler/handler.go`
- `internal/modules/hostplugin/api/routes.go`
- `internal/modules/opsagent/model/session.go`
- `internal/modules/opsagent/logic/server.go`
- `internal/modules/opsagent/logic/session_registry.go`
- `internal/modules/opsagent/logic/metrics_ingest.go`
- `internal/modules/opsagent/logic/exec_dispatch.go`
- `internal/modules/opsagent/logic/server_test.go`
- `internal/modules/opsagent/logic/exec_dispatch_test.go`

### Backend files to modify

- `internal/core/config/config.go`
- `configs/config.yaml`
- `internal/bootstrap/modules.go`
- `internal/svc/app_context.go`
- `internal/modules/host/logic/onboarding.go`
- `internal/modules/host/handler/host_mutation.go`
- `internal/modules/host/handler/host_query.go`
- `internal/modules/host/api/routes.go`
- `api/host/v1/host.go`
- `internal/modules/ai/agent/tools/host/runtime.go`
- `internal/modules/ai/agent/tools/host/tools.go`
- `internal/modules/ai/agent/tools/host/tools_test.go`
- `internal/modules/ai/agent/tools/factory.go`

### Frontend files to modify

- `web/src/api/modules/hosts.ts`
- `web/src/types/host.ts`
- `web/src/pages/Hosts/HostOnboardingPage.tsx`
- `web/src/pages/Hosts/Detail/index.tsx`
- `web/src/pages/Hosts/Detail/components/HostBasicInfoCard.tsx`
- `web/src/pages/Hosts/Detail/components/HostDetailTabs.tsx`
- `web/src/pages/Hosts/Detail/index.test.tsx`
- `web/src/api/modules/hosts.test.ts`

### Frontend files to create

- `web/src/pages/Hosts/Detail/tabs/PluginTab.tsx`
- `web/src/pages/Hosts/Detail/tabs/PluginTab.test.tsx`

## Task 1: Add Persistent Host Plugin Schema

**Files:**
- Create: `storage/migrations/20260429_0001_create_host_plugin_tables.sql`
- Create: `internal/modules/hostplugin/model/plugin.go`
- Test: `internal/modules/hostplugin/logic/service_test.go`

- [ ] **Step 1: Write the failing model migration test**

```go
func TestHostPluginModels_AutoMigrateAndPersist(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	err = db.AutoMigrate(
		&hostpluginmodel.HostPlugin{},
		&hostpluginmodel.HostPluginVersion{},
		&hostpluginmodel.HostPluginInstance{},
		&hostpluginmodel.HostPluginConfigRevision{},
		&hostpluginmodel.HostPluginTask{},
		&hostpluginmodel.HostPluginTaskLog{},
	)
	if err != nil {
		t.Fatalf("auto migrate hostplugin models: %v", err)
	}

	if !db.Migrator().HasTable(&hostpluginmodel.HostPluginInstance{}) {
		t.Fatalf("expected host_plugin_instances table")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/hostplugin/logic -run TestHostPluginModels_AutoMigrateAndPersist -v`
Expected: FAIL with `package .../internal/modules/hostplugin/logic: no Go files` or undefined model types.

- [ ] **Step 3: Create the SQL migration**

```sql
-- +migrate Up
CREATE TABLE host_plugins (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  plugin_key VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(128) NOT NULL,
  category VARCHAR(64) NOT NULL,
  description TEXT NOT NULL,
  default_version VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE host_plugin_versions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  plugin_id BIGINT NOT NULL,
  version VARCHAR(64) NOT NULL,
  arch VARCHAR(32) NOT NULL,
  package_path VARCHAR(255) NOT NULL,
  install_entry VARCHAR(128) NOT NULL,
  upgrade_entry VARCHAR(128) NOT NULL,
  uninstall_entry VARCHAR(128) NOT NULL,
  checksum VARCHAR(128) NOT NULL,
  capabilities_json JSON NOT NULL,
  config_schema_json JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_host_plugin_version_arch (plugin_id, version, arch)
);

CREATE TABLE host_plugin_instances (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  host_id BIGINT NOT NULL,
  plugin_id BIGINT NOT NULL,
  desired_version VARCHAR(64) NOT NULL,
  installed_version VARCHAR(64) NOT NULL DEFAULT '',
  install_status VARCHAR(32) NOT NULL DEFAULT 'pending',
  runtime_status VARCHAR(32) NOT NULL DEFAULT 'pending_online',
  health_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
  agent_id VARCHAR(128) NOT NULL DEFAULT '',
  last_seen_at TIMESTAMP NULL,
  capabilities_json JSON NOT NULL,
  last_error TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_host_plugin_instance (host_id, plugin_id)
);

-- +migrate Down
DROP TABLE IF EXISTS host_plugin_task_logs;
DROP TABLE IF EXISTS host_plugin_tasks;
DROP TABLE IF EXISTS host_plugin_config_revisions;
DROP TABLE IF EXISTS host_plugin_instances;
DROP TABLE IF EXISTS host_plugin_versions;
DROP TABLE IF EXISTS host_plugins;
```

- [ ] **Step 4: Add the GORM models**

```go
type HostPlugin struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	PluginKey      string    `gorm:"column:plugin_key;type:varchar(64);not null;uniqueIndex"`
	Name           string    `gorm:"column:name;type:varchar(128);not null"`
	Category       string    `gorm:"column:category;type:varchar(64);not null"`
	Description    string    `gorm:"column:description;type:text;not null"`
	DefaultVersion string    `gorm:"column:default_version;type:varchar(64);not null"`
	Status         string    `gorm:"column:status;type:varchar(32);not null;default:active"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (HostPlugin) TableName() string { return "host_plugins" }
```

- [ ] **Step 5: Run tests and migration status checks**

Run: `go test ./internal/modules/hostplugin/logic -run TestHostPluginModels_AutoMigrateAndPersist -v`
Expected: PASS

Run: `go run ./cmd/opspilot migrate status`
Expected: new `20260429_0001_create_host_plugin_tables.sql` listed as pending or applied depending on local DB state.

- [ ] **Step 6: Commit**

```bash
git add storage/migrations/20260429_0001_create_host_plugin_tables.sql internal/modules/hostplugin/model/plugin.go internal/modules/hostplugin/logic/service_test.go
git commit -m "feat(hostplugin): add plugin persistence models"
```

## Task 2: Scaffold Host Plugin Module And Catalog Service

**Files:**
- Create: `internal/modules/hostplugin/logic/service.go`
- Create: `internal/modules/hostplugin/logic/catalog.go`
- Create: `internal/modules/hostplugin/api/routes.go`
- Create: `internal/modules/hostplugin/handler/handler.go`
- Modify: `internal/bootstrap/modules.go`
- Test: `internal/modules/hostplugin/logic/service_test.go`

- [ ] **Step 1: Write the failing catalog seed test**

```go
func TestEnsureDefaultCatalogSeedsOpsAgent(t *testing.T) {
	db := openHostPluginTestDB(t)
	svc := NewService(&svc.ServiceContext{DB: db})

	if err := svc.EnsureDefaultCatalog(context.Background()); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	var count int64
	if err := db.Model(&hostpluginmodel.HostPlugin{}).Where("plugin_key = ?", "opsagent").Count(&count).Error; err != nil {
		t.Fatalf("count plugin: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected opsagent catalog entry, got %d", count)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/hostplugin/logic -run TestEnsureDefaultCatalogSeedsOpsAgent -v`
Expected: FAIL with `undefined: NewService` or `undefined: EnsureDefaultCatalog`.

- [ ] **Step 3: Create the service and catalog seed logic**

```go
type Service struct {
	svcCtx *svc.ServiceContext
}

func NewService(svcCtx *svc.ServiceContext) *Service {
	return &Service{svcCtx: svcCtx}
}

func (s *Service) EnsureDefaultCatalog(ctx context.Context) error {
	return s.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		plugin := hostpluginmodel.HostPlugin{
			PluginKey:      "opsagent",
			Name:           "OpsAgent",
			Category:       "host-observability",
			Description:    "Host metrics and sandbox execution plugin",
			DefaultVersion: "nodeagentx-dc57fbc-dirty",
			Status:         "active",
		}
		if err := tx.Where("plugin_key = ?", plugin.PluginKey).FirstOrCreate(&plugin).Error; err != nil {
			return err
		}
		return nil
	})
}
```

- [ ] **Step 4: Register HTTP routes for host plugin reads and task actions**

```go
func RegisterHostPluginHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := handler.NewHandler(svcCtx)
	g := v1.Group("/host-plugins", middleware.JWTAuth())
	g.GET("/catalog", h.ListCatalog)
	g.GET("/hosts/:id/instances", h.ListHostInstances)
	g.POST("/instances/:instance_id/actions", h.RunInstanceAction)
	g.GET("/tasks/:task_id", h.GetTask)
	g.GET("/tasks/:task_id/logs", h.ListTaskLogs)
}
```

- [ ] **Step 5: Mount the module**

```go
func RegisterModules(ctx context.Context, appCtx *svc.ServiceContext, engine *gin.Engine) {
	v1 := engine.Group("/api/v1")
	hostapi.RegisterHostHandlers(v1, appCtx)
	hostpluginapi.RegisterHostPluginHandlers(v1, appCtx)
	aiapi.RegisterAIHandlers(v1, appCtx)
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/modules/hostplugin/... ./internal/bootstrap/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/modules/hostplugin internal/bootstrap/modules.go
git commit -m "feat(hostplugin): scaffold catalog service and routes"
```

## Task 3: Extend Host Onboarding To Request Plugin Installation

**Files:**
- Modify: `api/host/v1/host.go`
- Modify: `internal/modules/host/logic/onboarding.go`
- Modify: `internal/modules/host/handler/host_mutation.go`
- Modify: `web/src/api/modules/hosts.ts`
- Modify: `web/src/types/host.ts`
- Modify: `web/src/pages/Hosts/HostOnboardingPage.tsx`
- Test: `internal/modules/host/logic/host_service_test.go`
- Test: `web/src/api/modules/hosts.test.ts`

- [ ] **Step 1: Write the failing backend request-shape test**

```go
func TestCreateWithProbe_CreatesHostAndPluginInstance(t *testing.T) {
	svc, db := setupHostServiceTest(t)
	req := CreateReq{
		Name: "host-a",
		IP: "10.0.0.8",
		PluginInstalls: []PluginInstallReq{{
			PluginKey: "opsagent",
			Version:   "nodeagentx-dc57fbc-dirty",
		}},
	}

	_, err := svc.CreateWithProbe(context.Background(), 1, true, req)
	if err != nil {
		t.Fatalf("create host with plugin: %v", err)
	}

	var count int64
	db.Table("host_plugin_instances").Where("host_id > 0").Count(&count)
	if count != 1 {
		t.Fatalf("expected one plugin instance, got %d", count)
	}
}
```

- [ ] **Step 2: Add API request fields**

```go
type PluginInstallReq struct {
	PluginKey string `json:"plugin_key"`
	Version   string `json:"version"`
}

type CreateReq struct {
	ProbeToken     string             `json:"probe_token"`
	Name           string             `json:"name"`
	PluginInstalls []PluginInstallReq `json:"plugin_installs"`
}
```

- [ ] **Step 3: Persist plugin install intent during host creation**

```go
if err := s.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	if err := tx.Create(node).Error; err != nil {
		return err
	}
	for _, item := range req.PluginInstalls {
		if err := hostpluginlogic.NewService(s.svcCtx).CreatePendingInstance(ctx, tx, uint64(node.ID), item.PluginKey, item.Version, userID); err != nil {
			return err
		}
	}
	return reassignOnboardingTrustedHostKeys(tx, uint64(node.ID), userID, probe.IP, probe.Port)
}); err != nil {
	return nil, err
}
```

- [ ] **Step 4: Expose plugin install selection in the host onboarding frontend**

```ts
export interface HostPluginInstallInput {
  pluginKey: string;
  version: string;
}

export interface HostCreateParams {
  probeToken: string;
  pluginInstalls?: HostPluginInstallInput[];
}

plugin_installs: (data.pluginInstalls || []).map((item) => ({
  plugin_key: item.pluginKey,
  version: item.version,
})),
```

- [ ] **Step 5: Add the onboarding form controls**

```tsx
<GuidedFormItem name="installOpsAgent" label="主机插件">
  <Radio.Group>
    <Radio value="none">暂不安装</Radio>
    <Radio value="opsagent">安装 OpsAgent</Radio>
  </Radio.Group>
</GuidedFormItem>

{form.getFieldValue('installOpsAgent') === 'opsagent' && (
  <GuidedFormItem name="opsagentVersion" label="OpsAgent 版本" rules={[{ required: true }]}>
    <Select options={catalogVersions.map((v) => ({ label: v.version, value: v.version }))} />
  </GuidedFormItem>
)}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/modules/host/logic -run TestCreateWithProbe_CreatesHostAndPluginInstance -v`
Expected: PASS

Run: `npm test -- --runInBand web/src/api/modules/hosts.test.ts`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add api/host/v1/host.go internal/modules/host/logic/onboarding.go internal/modules/host/handler/host_mutation.go web/src/api/modules/hosts.ts web/src/types/host.ts web/src/pages/Hosts/HostOnboardingPage.tsx
git commit -m "feat(host): capture host plugin install intent during onboarding"
```

## Task 4: Implement SSH-Based Plugin Install Task Execution

**Files:**
- Create: `internal/modules/hostplugin/logic/install.go`
- Create: `internal/modules/hostplugin/logic/task.go`
- Create: `internal/modules/hostplugin/logic/install_test.go`
- Modify: `internal/modules/host/handler/host_mutation.go`
- Modify: `internal/modules/host/api/routes.go`

- [ ] **Step 1: Write the failing package resolution test**

```go
func TestResolvePackageForHost_SelectsByArchitecture(t *testing.T) {
	svc := newHostPluginServiceForTest(t)
	version, err := svc.ResolveVersionForHost(context.Background(), "opsagent", "nodeagentx-dc57fbc-dirty", "amd64")
	if err != nil {
		t.Fatalf("resolve package: %v", err)
	}
	if !strings.Contains(version.PackagePath, "linux-amd64.tar.gz") {
		t.Fatalf("expected amd64 package path, got %s", version.PackagePath)
	}
}
```

- [ ] **Step 2: Implement package resolution and install task state transitions**

```go
func (s *Service) ResolveVersionForHost(ctx context.Context, pluginKey, version, arch string) (*hostpluginmodel.HostPluginVersion, error) {
	var row hostpluginmodel.HostPluginVersion
	err := s.svcCtx.DB.WithContext(ctx).
		Joins("JOIN host_plugins ON host_plugins.id = host_plugin_versions.plugin_id").
		Where("host_plugins.plugin_key = ? AND host_plugin_versions.version = ? AND host_plugin_versions.arch = ?", pluginKey, version, arch).
		First(&row).Error
	return &row, err
}
```

- [ ] **Step 3: Implement SSH install orchestration**

```go
func (s *Service) RunInstallTask(ctx context.Context, instanceID uint64) error {
	instance, host, version, err := s.loadInstallContext(ctx, instanceID)
	if err != nil {
		return err
	}
	task, err := s.startTask(ctx, instance.ID, "install")
	if err != nil {
		return err
	}
	defer s.finishTask(ctx, task, err)

	cmds := []string{
		fmt.Sprintf("mkdir -p /tmp/opspilot/plugins/%d", instance.ID),
		fmt.Sprintf("tar xzf %s -C /tmp/opspilot/plugins/%d", version.PackagePath, instance.ID),
		s.renderInstallCommand(instance, version),
	}
	return s.runSSHCommands(ctx, host, task, cmds)
}
```

- [ ] **Step 4: Trigger install execution after host creation**

```go
go func(instanceIDs []uint64) {
	for _, instanceID := range instanceIDs {
		if err := hostpluginlogic.NewService(h.svcCtx).RunInstallTask(context.Background(), instanceID); err != nil {
			logger.L().Error("host plugin install failed", logger.Error(err))
		}
	}
}(createdInstanceIDs)
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/modules/hostplugin/logic -run TestResolvePackageForHost_SelectsByArchitecture -v`
Expected: PASS

Run: `go test ./internal/modules/host/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/hostplugin/logic/install.go internal/modules/hostplugin/logic/task.go internal/modules/hostplugin/logic/install_test.go internal/modules/host/handler/host_mutation.go
git commit -m "feat(hostplugin): install opsagent over ssh tasks"
```

## Task 5: Add OpsAgent gRPC Runtime And Session Registry

**Files:**
- Modify: `internal/core/config/config.go`
- Modify: `configs/config.yaml`
- Create: `internal/modules/opsagent/model/session.go`
- Create: `internal/modules/opsagent/logic/session_registry.go`
- Create: `internal/modules/opsagent/logic/server.go`
- Create: `internal/modules/opsagent/logic/server_test.go`
- Modify: `internal/svc/app_context.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Write the failing registration test**

```go
func TestConnect_RegistrationMarksInstanceOnline(t *testing.T) {
	db := openOpsAgentTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}
	registry := NewSessionRegistry()
	server := NewServer(svcCtx, registry)

	stream := newFakeAgentServiceConnectServer(t,
		&pb.AgentMessage{Payload: &pb.AgentMessage_Registration{
			Registration: &pb.AgentRegistration{
				AgentId: "agent-host-1",
				Token:   "token-1",
			},
		}},
	)

	err := server.Connect(stream)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
}
```

- [ ] **Step 2: Add config for the gRPC listener**

```go
type OpsAgent struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Insecure bool   `mapstructure:"insecure"`
}

type Config struct {
	Server   Server   `mapstructure:"server"`
	OpsAgent OpsAgent `mapstructure:"opsagent"`
}
```

- [ ] **Step 3: Implement the session registry**

```go
type SessionRegistry struct {
	mu       sync.RWMutex
	byAgent  map[string]*Session
	byHostID map[uint64]*Session
}

func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{
		byAgent:  map[string]*Session{},
		byHostID: map[uint64]*Session{},
	}
}
```

- [ ] **Step 4: Implement `AgentService.Connect`**

```go
func (s *Server) Connect(stream pb.AgentService_ConnectServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	reg := msg.GetRegistration()
	if reg == nil {
		return status.Error(codes.InvalidArgument, "first message must be registration")
	}

	instance, err := s.bindRegistration(stream.Context(), reg)
	if err != nil {
		return err
	}
	s.registry.Put(instance.HostID, reg.AgentId, stream)
	return s.consumeMessages(stream.Context(), instance, stream)
}
```

- [ ] **Step 5: Start a dedicated gRPC listener beside the HTTP server**

```go
if config.CFG.OpsAgent.Enable {
	go func() {
		if err := startOpsAgentGRPCServer(ctx, svcCtx); err != nil {
			logger.L().Error("opsagent grpc server failed", logger.Error(err))
		}
	}()
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/modules/opsagent/... -v`
Expected: PASS

Run: `go test ./internal/server/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/core/config/config.go configs/config.yaml internal/modules/opsagent internal/svc/app_context.go internal/server/server.go
git commit -m "feat(opsagent): add grpc runtime and session registry"
```

## Task 6: Ingest Heartbeats, Config Updates, And Metrics

**Files:**
- Create: `internal/modules/opsagent/logic/metrics_ingest.go`
- Modify: `internal/modules/opsagent/logic/server.go`
- Create: `internal/modules/opsagent/logic/server_test.go`
- Modify: `internal/modules/host/handler/host_query.go`

- [ ] **Step 1: Write the failing heartbeat update test**

```go
func TestHandleHeartbeat_UpdatesPluginInstanceRuntimeState(t *testing.T) {
	svc := newOpsAgentServerForTest(t)
	err := svc.handleHeartbeat(context.Background(), &pb.Heartbeat{
		AgentId: "agent-host-1",
		Status:  "online",
	})
	if err != nil {
		t.Fatalf("handle heartbeat: %v", err)
	}
}
```

- [ ] **Step 2: Persist heartbeat and capability snapshots**

```go
func (s *Server) handleHeartbeat(ctx context.Context, hb *pb.Heartbeat) error {
	return s.svcCtx.DB.WithContext(ctx).
		Table("host_plugin_instances").
		Where("agent_id = ?", hb.AgentId).
		Updates(map[string]any{
			"runtime_status": "online",
			"health_status":  "healthy",
			"last_seen_at":   time.Now(),
		}).Error
}
```

- [ ] **Step 3: Implement metric normalization**

```go
func NormalizeMetricBatch(hostID uint64, batch *pb.MetricBatch) []hostmodel.HostHealthSnapshot {
	out := make([]hostmodel.HostHealthSnapshot, 0, len(batch.GetMetrics()))
	for _, metric := range batch.GetMetrics() {
		out = append(out, hostmodel.HostHealthSnapshot{
			HostID:    hostID,
			State:     "healthy",
			SummaryJSON: mustJSON(map[string]any{
				"name":   metric.GetName(),
				"tags":   metric.GetTags(),
				"fields": metric.GetFields(),
			}),
		})
	}
	return out
}
```

- [ ] **Step 4: Wire config update acknowledgements**

```go
func (s *Server) sendConfigUpdate(ctx context.Context, session *Session, revision hostpluginmodel.HostPluginConfigRevision) error {
	return session.Stream.Send(&pb.PlatformMessage{
		Payload: &pb.PlatformMessage_ConfigUpdate{
			ConfigUpdate: &pb.ConfigUpdate{
				ConfigYaml: []byte(revision.ConfigYAML),
				Version:    int64(revision.Version),
			},
		},
	})
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/modules/opsagent/... -run 'TestHandleHeartbeat_UpdatesPluginInstanceRuntimeState' -v`
Expected: PASS

Run: `go test ./internal/modules/host/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/opsagent/logic/metrics_ingest.go internal/modules/opsagent/logic/server.go internal/modules/opsagent/logic/server_test.go internal/modules/host/handler/host_query.go
git commit -m "feat(opsagent): persist heartbeats and ingest metrics"
```

## Task 7: Force AI Host Execution Through Plugin Sandbox

**Files:**
- Create: `internal/modules/opsagent/logic/exec_dispatch.go`
- Create: `internal/modules/opsagent/logic/exec_dispatch_test.go`
- Modify: `internal/modules/ai/agent/tools/host/runtime.go`
- Modify: `internal/modules/ai/agent/tools/host/tools.go`
- Modify: `internal/modules/ai/agent/tools/host/tools_test.go`
- Modify: `internal/modules/ai/agent/tools/factory.go`

- [ ] **Step 1: Write the failing no-fallback test**

```go
func TestHostExec_DeniesWhenPluginIsMissing(t *testing.T) {
	svcCtx := &svc.ServiceContext{DB: openHostToolDBWithoutPlugin(t)}
	ctx := runtimectx.WithServices(context.Background(), svcCtx)
	tool := HostExec(ctx)

	_, err := invokeHostExec(tool, `{"target":"10.0.0.8","command":"uptime"}`)
	if err == nil || !strings.Contains(err.Error(), "plugin required") {
		t.Fatalf("expected plugin required error, got %v", err)
	}
}
```

- [ ] **Step 2: Add plugin-backed dispatch**

```go
func DispatchSandboxCommand(ctx context.Context, svcCtx *svc.ServiceContext, node *hostmodel.Node, command string) (*HostExecOutput, error) {
	instance, err := hostpluginlogic.NewService(svcCtx).RequireOnlineCapability(ctx, uint64(node.ID), "exec.shell")
	if err != nil {
		return nil, fmt.Errorf("plugin required: %w", err)
	}
	result, err := opsagentlogic.NewDispatcher(svcCtx).ExecuteCommand(ctx, instance, command)
	if err != nil {
		return nil, err
	}
	return &HostExecOutput{HostID: int(node.ID), Command: command, Stdout: result.Stdout, Stderr: result.Stderr, ExitCode: result.ExitCode, Status: "completed"}, nil
}
```

- [ ] **Step 3: Remove SSH fallback from AI execution path**

```go
if node == nil {
	return nil, fmt.Errorf("plugin required: localhost execution is not supported for host_exec")
}

return DispatchSandboxCommand(ctx, svcCtx, node, cmd)
```

- [ ] **Step 4: Add script dispatch mapping**

```go
if script != "" {
	return dispatchSandboxScript(ctx, svcCtx, node, "sh", script)
}
return DispatchSandboxCommand(ctx, svcCtx, node, cmd)
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/modules/ai/agent/tools/host -v`
Expected: PASS

Run: `go test ./internal/modules/opsagent/logic -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/opsagent/logic/exec_dispatch.go internal/modules/opsagent/logic/exec_dispatch_test.go internal/modules/ai/agent/tools/host/runtime.go internal/modules/ai/agent/tools/host/tools.go internal/modules/ai/agent/tools/host/tools_test.go internal/modules/ai/agent/tools/factory.go
git commit -m "feat(ai): require opsagent sandbox for host execution"
```

## Task 8: Expose Plugin State In Host APIs And UI

**Files:**
- Modify: `web/src/api/modules/hosts.ts`
- Modify: `web/src/types/host.ts`
- Modify: `web/src/pages/Hosts/Detail/index.tsx`
- Modify: `web/src/pages/Hosts/Detail/components/HostBasicInfoCard.tsx`
- Create: `web/src/pages/Hosts/Detail/tabs/PluginTab.tsx`
- Create: `web/src/pages/Hosts/Detail/tabs/PluginTab.test.tsx`
- Modify: `web/src/pages/Hosts/Detail/components/HostDetailTabs.tsx`
- Modify: `internal/modules/host/handler/host_query.go`

- [ ] **Step 1: Write the failing host-detail API normalization test**

```ts
it('maps plugin instances from host detail', async () => {
  mockGet.mockResolvedValue({
    data: {
      id: 1,
      name: 'host-a',
      plugin_instances: [{ plugin_key: 'opsagent', runtime_status: 'online', installed_version: 'nodeagentx-dc57fbc-dirty' }],
    },
  });

  const res = await hostsApi.getHostDetail('1');
  expect(res.data.pluginInstances?.[0].pluginKey).toBe('opsagent');
});
```

- [ ] **Step 2: Add plugin instance fields to backend detail responses**

```go
httpx.OK(c, gin.H{
	"id": node.ID,
	"name": node.Name,
	"plugin_instances": instances,
})
```

- [ ] **Step 3: Add host detail types and API mapping**

```ts
export interface HostPluginInstance {
  pluginKey: string;
  installedVersion: string;
  installStatus: string;
  runtimeStatus: string;
  healthStatus: string;
  lastSeenAt?: string;
}
```

- [ ] **Step 4: Add the Plugin tab**

```tsx
const PluginTab: React.FC<{ host: Host }> = ({ host }) => (
  <Card>
    <Table
      rowKey={(row) => `${row.pluginKey}-${row.installedVersion}`}
      dataSource={host.pluginInstances || []}
      columns={[
        { title: '插件', dataIndex: 'pluginKey' },
        { title: '版本', dataIndex: 'installedVersion' },
        { title: '安装状态', dataIndex: 'installStatus' },
        { title: '运行状态', dataIndex: 'runtimeStatus' },
        { title: '最近心跳', dataIndex: 'lastSeenAt' },
      ]}
    />
  </Card>
);
```

- [ ] **Step 5: Run tests**

Run: `npm test -- --runInBand web/src/api/modules/hosts.test.ts web/src/pages/Hosts/Detail/index.test.tsx web/src/pages/Hosts/Detail/tabs/PluginTab.test.tsx`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/api/modules/hosts.ts web/src/types/host.ts web/src/pages/Hosts/Detail/index.tsx web/src/pages/Hosts/Detail/components/HostBasicInfoCard.tsx web/src/pages/Hosts/Detail/components/HostDetailTabs.tsx web/src/pages/Hosts/Detail/tabs/PluginTab.tsx web/src/pages/Hosts/Detail/tabs/PluginTab.test.tsx internal/modules/host/handler/host_query.go
git commit -m "feat(host-ui): show host plugin lifecycle and runtime state"
```

## Task 9: Add End-To-End Validation And Operational Documentation

**Files:**
- Modify: `docs/platform-integration-guide.md`
- Modify: `docs/superpowers/specs/2026-04-29-opsagent-host-plugin-design.md`
- Create: `internal/modules/opsagent/logic/exec_dispatch_test.go`
- Create: `internal/modules/hostplugin/logic/install_test.go`

- [ ] **Step 1: Write the failing integration tests**

```go
func TestExecuteCommand_ReturnsStructuredFailureWhenInstanceOffline(t *testing.T) {
	dispatcher := newDispatcherForOfflineInstance(t)
	_, err := dispatcher.ExecuteCommand(context.Background(), offlineInstance, "uptime")
	if err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("expected offline error, got %v", err)
	}
}
```

- [ ] **Step 2: Add the offline and capability mismatch assertions**

```go
func TestRequireOnlineCapability_RejectsCapabilityMismatch(t *testing.T) {
	svc := newHostPluginServiceForTest(t)
	_, err := svc.RequireOnlineCapability(context.Background(), 1, "exec.script.python")
	if err == nil || !strings.Contains(err.Error(), "capability") {
		t.Fatalf("expected capability error, got %v", err)
	}
}
```

- [ ] **Step 3: Update the integration guide with platform-owned responsibilities**

```md
## Platform integration in OpsPilot

- Host onboarding can request OpsAgent installation.
- Installation, upgrade, and uninstall are SSH-driven plugin tasks.
- Runtime metrics and sandbox execution are gRPC-driven.
- AI host execution now requires an online OpsAgent plugin instance and does not fall back to SSH.
```

- [ ] **Step 4: Run the verification matrix**

Run: `go test ./internal/modules/hostplugin/... ./internal/modules/opsagent/... ./internal/modules/ai/agent/tools/host/... ./internal/modules/host/... -v`
Expected: PASS

Run: `npm test -- --runInBand web/src/api/modules/hosts.test.ts web/src/pages/Hosts/Detail/index.test.tsx web/src/pages/Hosts/Detail/tabs/PluginTab.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docs/platform-integration-guide.md docs/superpowers/specs/2026-04-29-opsagent-host-plugin-design.md internal/modules/hostplugin/logic/install_test.go internal/modules/opsagent/logic/exec_dispatch_test.go
git commit -m "test(docs): verify opsagent host plugin integration flow"
```

## Coverage Check

This plan covers the approved spec sections as follows:

- Host plugin framework and catalog: Tasks 1-2
- Host onboarding plugin selection and pending instances: Task 3
- SSH-based install control plane: Task 4
- gRPC registration, heartbeat, config, metrics: Tasks 5-6
- AI sandbox-only host execution: Task 7
- Host detail and operator visibility: Task 8
- Errors, tests, and docs handoff: Task 9

## Plan Review Notes

- Function names are kept consistent across tasks: `EnsureDefaultCatalog`, `RunInstallTask`, `RequireOnlineCapability`, `ExecuteCommand`.
- The plan assumes a new `storage/migrations/` directory may need to be created in this checkout because the migration runner already expects that path.
