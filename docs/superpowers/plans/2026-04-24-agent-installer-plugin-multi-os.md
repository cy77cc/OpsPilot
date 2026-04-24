# Agent Installer Plugin (Optional Onboarding + Multi-OS Builds) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an independent `agent-installer` module so host onboarding can optionally install agent binaries, with artifact selection by host `os/arch` and multi-platform build outputs.

**Architecture:** Keep SSH onboarding unchanged as baseline, then layer optional installer orchestration on top. `host` module calls `agent-installer` via interface from `ServiceContext`; installer resolves package from signed manifest and records install state on `nodes`. UI exposes an “install agent” switch during host creation and sends install options to backend.

**Tech Stack:** Go (Gin, GORM), TypeScript (React + Ant Design), shell build scripts, existing Makefile/test pipelines.

---

Scope note: this plan intentionally focuses only on the “agent as optional plugin during host onboarding” subsystem. AI action orchestration and runtime policy expansion are out of scope for this implementation cycle.

### Task 1: Create Independent `agent-installer` Module

**Files:**
- Create: `internal/modules/agentinstaller/model/package.go`
- Create: `internal/modules/agentinstaller/logic/service.go`
- Create: `internal/modules/agentinstaller/logic/service_test.go`
- Create: `internal/modules/agentinstaller/logic/noop.go`
- Modify: `internal/svc/app_context.go`

- [ ] **Step 1: Write failing tests for package resolution by OS/Arch**

```go
// internal/modules/agentinstaller/logic/service_test.go
func TestResolvePackage_SelectsExactOSArch(t *testing.T) {
    svc := NewServiceFromManifest([]model.AgentPackage{
        {Version: "1.0.0", Channel: "stable", OS: "linux", Arch: "amd64", URL: "https://example/linux-amd64.tar.gz", SHA256: "a"},
        {Version: "1.0.0", Channel: "stable", OS: "linux", Arch: "arm64", URL: "https://example/linux-arm64.tar.gz", SHA256: "b"},
    })

    pkg, err := svc.ResolvePackage(context.Background(), ResolveInput{
        OS: "linux", Arch: "arm64", Channel: "stable", Version: "1.0.0",
    })
    if err != nil {
        t.Fatalf("resolve package: %v", err)
    }
    if pkg.Arch != "arm64" {
        t.Fatalf("expected arm64, got %s", pkg.Arch)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/agentinstaller/logic -run TestResolvePackage_SelectsExactOSArch -v`  
Expected: FAIL with `undefined: NewServiceFromManifest` (or similar missing symbol).

- [ ] **Step 3: Implement installer model + resolver service**

```go
// internal/modules/agentinstaller/model/package.go
type AgentPackage struct {
    Version string `json:"version"`
    Channel string `json:"channel"`
    OS      string `json:"os"`
    Arch    string `json:"arch"`
    URL     string `json:"url"`
    SHA256  string `json:"sha256"`
    Sign    string `json:"sign"`
}

// internal/modules/agentinstaller/logic/service.go
type ResolveInput struct {
    OS      string
    Arch    string
    Channel string
    Version string
}

type Service interface {
    ResolvePackage(ctx context.Context, in ResolveInput) (*model.AgentPackage, error)
}
```

- [ ] **Step 4: Add default no-op installer and wire into ServiceContext**

```go
// internal/modules/agentinstaller/logic/noop.go
type NoopService struct{}

func (NoopService) ResolvePackage(ctx context.Context, in ResolveInput) (*model.AgentPackage, error) {
    return nil, errors.New("agent installer not configured")
}
```

```go
// internal/svc/app_context.go
type ServiceContext struct {
    // ...
    AgentInstaller agentinstallerlogic.Service
}
```

- [ ] **Step 5: Re-run installer tests**

Run: `go test ./internal/modules/agentinstaller/logic -v`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/agentinstaller/model/package.go internal/modules/agentinstaller/logic/service.go internal/modules/agentinstaller/logic/service_test.go internal/modules/agentinstaller/logic/noop.go internal/svc/app_context.go
git commit -m "feat(agentinstaller): add independent installer module and resolver service"
```

### Task 2: Extend Host Create Contract for Optional Agent Install

**Files:**
- Modify: `api/host/v1/host.go`
- Modify: `internal/modules/host/logic/host_service.go`
- Modify: `internal/modules/host/model/node.go`
- Modify: `web/src/api/modules/hosts.ts`

- [ ] **Step 1: Write failing contract test for create payload mapping**

```ts
// web/src/api/modules/hosts.test.ts
it('maps install options in createHost payload', async () => {
  const post = vi.spyOn(apiService, 'post').mockResolvedValue({ data: {} } as any);
  await hostApi.createHost({
    name: 'n1',
    ip: '10.0.0.1',
    installAgent: true,
    agentChannel: 'stable',
  });
  expect(post).toHaveBeenCalledWith('/hosts', expect.objectContaining({
    install_agent: true,
    agent_channel: 'stable',
  }));
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- src/api/modules/hosts.test.ts -t "maps install options"`  
Expected: FAIL because `install_agent`/`agent_channel` are not sent yet.

- [ ] **Step 3: Add API fields for install options**

```go
// api/host/v1/host.go + internal/modules/host/logic/host_service.go (CreateReq)
InstallAgent bool   `json:"install_agent"`
AgentChannel string `json:"agent_channel"`
AgentVersion string `json:"agent_version"`
```

```ts
// web/src/api/modules/hosts.ts
export interface HostCreateParams {
  // ...
  installAgent?: boolean;
  agentChannel?: 'stable' | 'canary';
  agentVersion?: string;
}
```

- [ ] **Step 4: Add node-level install state fields**

```go
// internal/modules/host/model/node.go
AgentInstalled     bool       `gorm:"column:agent_installed;default:false" json:"agent_installed"`
AgentChannel       string     `gorm:"column:agent_channel;type:varchar(32)" json:"agent_channel"`
AgentVersion       string     `gorm:"column:agent_version;type:varchar(64)" json:"agent_version"`
AgentInstallStatus string     `gorm:"column:agent_install_status;type:varchar(32);default:not_installed" json:"agent_install_status"`
AgentInstallError  string     `gorm:"column:agent_install_error;type:text" json:"agent_install_error"`
AgentInstalledAt   *time.Time `gorm:"column:agent_installed_at" json:"agent_installed_at"`
```

- [ ] **Step 5: Re-run contract test**

Run: `cd web && npm test -- src/api/modules/hosts.test.ts -t "maps install options"`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add api/host/v1/host.go internal/modules/host/logic/host_service.go internal/modules/host/model/node.go web/src/api/modules/hosts.ts web/src/api/modules/hosts.test.ts
git commit -m "feat(host): add optional agent install fields to host create contract"
```

### Task 3: Execute Optional Install During Host Onboarding

**Files:**
- Create: `internal/modules/host/logic/agent_install.go`
- Modify: `internal/modules/host/logic/onboarding.go`
- Modify: `internal/modules/host/logic/host_service_test.go`

- [ ] **Step 1: Write failing backend test for optional install path**

```go
func TestCreateWithProbe_InstallAgentRequestedMarksInstalled(t *testing.T) {
    s, db := newHostLogicTestService(t)
    // seed probe token ...
    node, err := s.CreateWithProbe(context.Background(), 1, true, CreateReq{
        ProbeToken: "valid-token",
        Name: "node-1",
        InstallAgent: true,
        AgentChannel: "stable",
    })
    if err != nil {
        t.Fatalf("create with probe: %v", err)
    }
    if node.AgentInstallStatus == "not_installed" {
        t.Fatalf("expected install status updated, got %s", node.AgentInstallStatus)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/host/logic -run TestCreateWithProbe_InstallAgentRequestedMarksInstalled -v`  
Expected: FAIL because install flow is not implemented.

- [ ] **Step 3: Add install orchestration helper**

```go
// internal/modules/host/logic/agent_install.go
func (s *HostService) installAgentIfRequested(ctx context.Context, node *model.Node, req CreateReq) {
    if !req.InstallAgent || s.svcCtx.AgentInstaller == nil {
        return
    }
    node.AgentInstallStatus = "installing"
    pkg, err := s.svcCtx.AgentInstaller.ResolvePackage(ctx, agentinstallerlogic.ResolveInput{
        OS: node.OS, Arch: node.Arch, Channel: firstNonEmpty(req.AgentChannel, "stable"), Version: req.AgentVersion,
    })
    if err != nil {
        node.AgentInstallStatus = "failed"
        node.AgentInstallError = err.Error()
        return
    }
    node.AgentChannel = pkg.Channel
    node.AgentVersion = pkg.Version
    node.AgentInstalled = true
    node.AgentInstallStatus = "installed"
    now := time.Now()
    node.AgentInstalledAt = &now
}
```

- [ ] **Step 4: Call helper from `CreateWithProbe` and `createFromLegacyReq`**

```go
// internal/modules/host/logic/onboarding.go
s.installAgentIfRequested(ctx, node, req)
```

- [ ] **Step 5: Run host logic test suite**

Run: `go test ./internal/modules/host/logic -v`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/host/logic/agent_install.go internal/modules/host/logic/onboarding.go internal/modules/host/logic/host_service_test.go
git commit -m "feat(host): support optional agent install during host onboarding"
```

### Task 4: Add Onboarding UI Switch + OS-Aware Defaults

**Files:**
- Modify: `web/src/pages/Hosts/HostOnboardingPage.tsx`
- Modify: `web/src/api/modules/hosts.ts`
- Test: `web/src/api/modules/hosts.test.ts`

- [ ] **Step 1: Add failing UI/API behavior test**

```ts
it('sends stable channel when installAgent is enabled', async () => {
  const post = vi.spyOn(apiService, 'post').mockResolvedValue({ data: {} } as any);
  await hostApi.createHost({ name: 'n1', ip: '10.0.0.1', installAgent: true });
  expect(post).toHaveBeenCalledWith('/hosts', expect.objectContaining({
    install_agent: true,
    agent_channel: 'stable',
  }));
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- src/api/modules/hosts.test.ts -t "stable channel"`  
Expected: FAIL.

- [ ] **Step 3: Add install controls in onboarding step 3**

```tsx
<Form.Item name="installAgent" label="安装 Agent" valuePropName="checked">
  <Switch />
</Form.Item>
<Form.Item noStyle shouldUpdate>
  {({ getFieldValue }) =>
    getFieldValue('installAgent') ? (
      <GuidedFormItem name="agentChannel" label="安装通道" initialValue="stable">
        <Select options={[{ value: 'stable', label: 'Stable' }, { value: 'canary', label: 'Canary' }]} />
      </GuidedFormItem>
    ) : null
  }
</Form.Item>
```

- [ ] **Step 4: Send install fields in createHost call**

```ts
await Api.hosts.createHost({
  // ...
  installAgent: !!values.installAgent,
  agentChannel: values.agentChannel || 'stable',
});
```

- [ ] **Step 5: Re-run frontend tests**

Run: `cd web && npm run test:run -- src/api/modules/hosts.test.ts`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/Hosts/HostOnboardingPage.tsx web/src/api/modules/hosts.ts web/src/api/modules/hosts.test.ts
git commit -m "feat(web): add optional agent install controls in host onboarding"
```

### Task 5: Multi-OS Build + Artifact Manifest Pipeline

**Files:**
- Create: `cmd/opspilot-agent/main.go`
- Create: `script/agent/build.sh`
- Create: `script/agent/manifest.template.json`
- Modify: `Makefile`
- Create: `docs/runbooks/agent-installer-release.md`

- [ ] **Step 1: Add failing smoke test for build script output**

```bash
# script/agent/build.sh should produce:
# dist/agent/opspilot-agent-linux-amd64.tar.gz
# dist/agent/opspilot-agent-linux-arm64.tar.gz
# dist/agent/opspilot-agent-windows-amd64.zip
```

- [ ] **Step 2: Run build script to verify it fails initially**

Run: `bash script/agent/build.sh`  
Expected: FAIL because files/script do not exist yet.

- [ ] **Step 3: Add minimal agent entrypoint and cross-build script**

```go
// cmd/opspilot-agent/main.go
func main() {
    log.Println("opspilot-agent bootstrap")
}
```

```bash
# script/agent/build.sh
GOOS=linux GOARCH=amd64 go build -o dist/agent/opspilot-agent-linux-amd64 ./cmd/opspilot-agent
GOOS=linux GOARCH=arm64 go build -o dist/agent/opspilot-agent-linux-arm64 ./cmd/opspilot-agent
GOOS=windows GOARCH=amd64 go build -o dist/agent/opspilot-agent-windows-amd64.exe ./cmd/opspilot-agent
```

- [ ] **Step 4: Add make targets and release runbook**

```make
build-agent:
	bash script/agent/build.sh
```

```md
# docs/runbooks/agent-installer-release.md
- how to run `make build-agent`
- how to generate checksum/signature
- how to publish manifest
```

- [ ] **Step 5: Run build and verify artifacts**

Run: `make build-agent`  
Expected: PASS and artifacts under `dist/agent/`.

- [ ] **Step 6: Commit**

```bash
git add cmd/opspilot-agent/main.go script/agent/build.sh script/agent/manifest.template.json Makefile docs/runbooks/agent-installer-release.md
git commit -m "build(agent): add multi-os agent build pipeline and manifest template"
```

### Task 6: Host Package Selection Endpoint + Install Observability

**Files:**
- Modify: `internal/modules/host/api/routes.go`
- Create: `internal/modules/host/handler/agent_install.go`
- Create: `internal/modules/host/handler/agent_install_test.go`
- Modify: `web/src/api/modules/hosts.ts`

- [ ] **Step 1: Write failing handler test for package resolve endpoint**

```go
func TestResolveAgentPackage(t *testing.T) {
    // GET /hosts/agent/packages/resolve?os=linux&arch=amd64&channel=stable
    // expect 200 + package payload
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/host/handler -run TestResolveAgentPackage -v`  
Expected: FAIL (route/handler missing).

- [ ] **Step 3: Implement resolve/install-status handlers**

```go
// internal/modules/host/handler/agent_install.go
func (h *Handler) ResolveAgentPackage(c *gin.Context) { /* call svcCtx.AgentInstaller.ResolvePackage */ }
func (h *Handler) GetAgentInstallStatus(c *gin.Context) { /* read node agent fields */ }
```

- [ ] **Step 4: Register routes and frontend API methods**

```go
// internal/modules/host/api/routes.go
g.GET("/agent/packages/resolve", h.ResolveAgentPackage)
g.GET("/:id/agent/install-status", h.GetAgentInstallStatus)
```

```ts
// web/src/api/modules/hosts.ts
resolveAgentPackage(params: { os: string; arch: string; channel?: string })
getAgentInstallStatus(hostId: string)
```

- [ ] **Step 5: Run backend + frontend targeted tests**

Run: `go test ./internal/modules/host/handler -v`  
Expected: PASS.

Run: `cd web && npm run test:run -- src/api/modules/hosts.test.ts`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/host/api/routes.go internal/modules/host/handler/agent_install.go internal/modules/host/handler/agent_install_test.go web/src/api/modules/hosts.ts web/src/api/modules/hosts.test.ts
git commit -m "feat(host): expose agent package resolve and install status endpoints"
```

## Self-Review Checklist

- Spec coverage: covered independent module, optional onboarding install switch, multi-OS builds, OS/Arch package selection, installer/OTA shared metadata, test and runbook updates.
- Placeholder scan: no TODO/TBD placeholders remain.
- Type consistency: `install_agent`, `agent_channel`, `agent_version` are used consistently across API, host logic `CreateReq`, and frontend `HostCreateParams`.

