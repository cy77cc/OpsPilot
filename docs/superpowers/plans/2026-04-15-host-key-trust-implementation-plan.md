# Host Key Trust Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an explicit host-key trust workflow that preserves strict SSH verification while letting authorized users trust unknown or rotated host keys and retry the original host operation.

**Architecture:** Implement a dual-write bridge: trusted host keys are stored in the host domain for audit and product UX, while the runtime SSH verifier still uses a managed `known_hosts` file. Unknown or mismatched host keys are normalized into structured host-domain errors, surfaced through host APIs, and resolved via a dedicated trust endpoint that persists DB state and synchronizes `known_hosts` atomically.

**Tech Stack:** Go, Gin, Gorm, `golang.org/x/crypto/ssh`, `golang.org/x/crypto/ssh/knownhosts`, React, TypeScript, Ant Design, Vitest.

---

## Scope Check

This spec touches one cohesive subsystem: host SSH trust management across all host SSH entrypoints. It affects backend host logic, SSH client error normalization, one new persistence model, and frontend host UX. It does not need to be split into separate plans.

## File Structure Lock

**Backend persistence and migration**
- Modify: `internal/modules/host/model/node.go`
- Create: `internal/modules/host/model/trusted_host_key.go`
- Modify: `internal/core/storage/migration/dev_auto.go`

**Backend SSH error normalization and `known_hosts` sync**
- Modify: `internal/client/ssh/known_hosts.go`
- Modify: `internal/client/ssh/ssh.go`
- Modify: `internal/client/ssh/ssh_test.go`

**Backend host trust service + API**
- Modify: `internal/modules/host/logic/probe.go`
- Modify: `internal/modules/host/logic/onboarding.go`
- Modify: `internal/modules/host/logic/host_service.go`
- Create: `internal/modules/host/logic/trusted_host_keys.go`
- Modify: `internal/modules/host/handler/common.go`
- Modify: `internal/modules/host/handler/host_mutation.go`
- Modify: `internal/modules/host/handler/host_exec.go`
- Modify: `internal/modules/host/handler/terminal_session.go`
- Modify: `internal/modules/host/handler/files_handler.go`
- Modify: `internal/modules/host/api/routes.go`
- Modify tests: `internal/modules/host/logic/probe_test.go`
- Create tests: `internal/modules/host/handler/trusted_host_keys_test.go`

**Frontend host-key trust UX**
- Modify: `web/src/api/modules/hosts.ts`
- Create: `web/src/components/Hosts/HostKeyTrustModal.tsx`
- Create: `web/src/hooks/useHostKeyTrust.ts`
- Modify: `web/src/pages/Hosts/HostOnboardingPage.tsx`
- Modify: `web/src/pages/Hosts/HostDetailPage.tsx`
- Modify: `web/src/pages/Hosts/HostTerminalPage.tsx`
- Modify tests: `web/src/pages/Hosts/HostDetailPage.test.tsx`
- Create tests: `web/src/hooks/useHostKeyTrust.test.tsx`

**Docs**
- Modify: `README.md`
- Modify: `docs/reviews/2026-04-14-full-architecture-security-review.md`

---

### Task 1: Add trusted host key persistence model

**Files:**
- Create: `internal/modules/host/model/trusted_host_key.go`
- Modify: `internal/core/storage/migration/dev_auto.go`
- Test: `internal/modules/host/logic/probe_test.go`

- [ ] **Step 1: Write the failing persistence test**

```go
func TestTrustedHostKeyModel_AutoMigrates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&model.TrustedHostKey{}); err != nil {
		t.Fatalf("auto migrate trusted host key: %v", err)
	}

	item := &model.TrustedHostKey{
		HostID:            10,
		Host:              "118.193.38.89",
		Port:              13012,
		Algorithm:         "ssh-ed25519",
		FingerprintSHA256: "SHA256:test-fingerprint",
		PublicKey:         "ssh-ed25519 AAAATEST",
		Status:            model.TrustedHostKeyStatusTrusted,
		CreatedBy:         1,
		ConfirmedAt:       time.Now(),
		LastSeenAt:        time.Now(),
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("create trusted host key: %v", err)
	}
}
```

- [ ] **Step 2: Run the focused test to verify failure**

Run: `go test ./internal/modules/host/logic -run TestTrustedHostKeyModel_AutoMigrates -v`  
Expected: FAIL because `TrustedHostKey` does not exist yet.

- [ ] **Step 3: Add the model**

```go
type TrustedHostKeyStatus string

const (
	TrustedHostKeyStatusTrusted TrustedHostKeyStatus = "trusted"
	TrustedHostKeyStatusRotated TrustedHostKeyStatus = "rotated"
	TrustedHostKeyStatusRevoked TrustedHostKeyStatus = "revoked"
)

type TrustedHostKey struct {
	ID                uint64               `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	HostID            uint64               `gorm:"column:host_id;index;not null" json:"host_id"`
	Host              string               `gorm:"column:host;type:varchar(255);not null;index" json:"host"`
	Port              int                  `gorm:"column:port;not null;index" json:"port"`
	Algorithm         string               `gorm:"column:algorithm;type:varchar(64);not null" json:"algorithm"`
	FingerprintSHA256 string               `gorm:"column:fingerprint_sha256;type:varchar(128);not null;index" json:"fingerprint_sha256"`
	PublicKey         string               `gorm:"column:public_key;type:text;not null" json:"public_key"`
	Status            TrustedHostKeyStatus `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	CreatedBy         uint64               `gorm:"column:created_by;not null;index" json:"created_by"`
	ConfirmedAt       time.Time            `gorm:"column:confirmed_at;not null" json:"confirmed_at"`
	LastSeenAt        time.Time            `gorm:"column:last_seen_at;not null" json:"last_seen_at"`
	CreatedAt         time.Time            `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time            `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TrustedHostKey) TableName() string { return "host_trusted_keys" }
```

- [ ] **Step 4: Register the model in dev auto-migrate**

```go
return db.AutoMigrate(
	&hostmodel.Node{},
	&hostmodel.NodeEvent{},
	&hostmodel.SSHKey{},
	&hostmodel.TrustedHostKey{},
	&hostmodel.HostCloudAccount{},
```

- [ ] **Step 5: Run the focused test to verify it passes**

Run: `go test ./internal/modules/host/logic -run TestTrustedHostKeyModel_AutoMigrates -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/host/model/trusted_host_key.go internal/core/storage/migration/dev_auto.go internal/modules/host/logic/probe_test.go
git commit -m "feat(host): persist trusted ssh host keys"
```

### Task 2: Normalize SSH host-key verification errors

**Files:**
- Modify: `internal/client/ssh/known_hosts.go`
- Modify: `internal/client/ssh/ssh_test.go`
- Test: `internal/client/ssh/ssh_test.go`

- [ ] **Step 1: Write the failing actionable-error assertion**

```go
func TestNewSSHClient_UnknownHostKeyErrorIncludesActionableDetails(t *testing.T) {
	host, port, shutdown := startTestPasswordSSHServer(t, "tester", "secret")
	defer shutdown()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(knownHostsPath, nil, 0o600); err != nil {
		t.Fatalf("write empty known_hosts file: %v", err)
	}
	t.Setenv(knownHostsPathEnvKey, knownHostsPath)

	_, err := NewSSHClient("tester", "secret", host, port, "", "")
	if err == nil {
		t.Fatal("expected unknown host key rejection")
	}
	if !strings.Contains(err.Error(), "fingerprint ") {
		t.Fatalf("expected fingerprint detail, got %v", err)
	}
	if !strings.Contains(err.Error(), knownHostsPath) {
		t.Fatalf("expected known_hosts path detail, got %v", err)
	}
}
```

- [ ] **Step 2: Run the SSH client tests to verify failure**

Run: `go test ./internal/client/ssh -run TestNewSSHClient_UnknownHostKeyErrorIncludesActionableDetails -v`  
Expected: FAIL because current error text is too generic.

- [ ] **Step 3: Wrap known_hosts failures with actionable detail**

```go
func formatHostKeyVerificationError(err error, hostname string, remote net.Addr, key ssh.PublicKey, knownHostsPath string) error {
	target := strings.TrimSpace(hostname)
	if target == "" && remote != nil {
		target = remote.String()
	}
	fingerprint := ssh.FingerprintSHA256(key)

	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) == 0 {
			return fmt.Errorf("ssh host key for %s is unknown (fingerprint %s); add it to %s or set %s: %w", target, fingerprint, knownHostsPath, knownHostsPathEnvKey, err)
		}
		return fmt.Errorf("ssh host key mismatch for %s (presented fingerprint %s); verify and update %s: %w", target, fingerprint, knownHostsPath, err)
	}
	return err
}
```

- [ ] **Step 4: Apply the wrapper inside the callback returned by `loadKnownHostsVerifier`**

```go
return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	if err := callback(hostname, remote, key); err != nil {
		return formatHostKeyVerificationError(err, hostname, remote, key, knownHostsPath)
	}
	return nil
}, nil
```

- [ ] **Step 5: Run the SSH client tests**

Run: `go test ./internal/client/ssh -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/client/ssh/known_hosts.go internal/client/ssh/ssh_test.go
git commit -m "feat(ssh): surface actionable host-key verification errors"
```

### Task 3: Return structured host-key trust metadata from host operations

**Files:**
- Modify: `internal/modules/host/logic/host_service.go`
- Modify: `internal/modules/host/logic/probe.go`
- Modify: `internal/modules/host/logic/onboarding.go`
- Modify: `internal/modules/host/logic/probe_test.go`

- [ ] **Step 1: Write a failing host-logic test for structured host-key errors**

```go
func TestUpdateCredentials_ReturnsStructuredHostKeyFailure(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)

	node := &model.Node{
		Name: "host-key-error-node",
		IP: "127.0.0.1",
		Port: 1,
		SSHUser: "root",
		Status: "offline",
		Source: "manual_ssh",
	}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	_, resp, err := hostSvc.UpdateCredentials(context.Background(), uint64(node.ID), UpdateCredentialsReq{
		AuthType: "password",
		Username: "root",
		Password: "secret",
		Port: 1,
	})
	if err == nil {
		t.Fatal("expected update credentials error")
	}
	if resp == nil || resp.ErrorCode != "ssh_host_key_unknown" {
		t.Fatalf("expected structured host-key error, got %#v", resp)
	}
}
```

- [ ] **Step 2: Run the focused logic test to verify failure**

Run: `go test ./internal/modules/host/logic -run TestUpdateCredentials_ReturnsStructuredHostKeyFailure -v`  
Expected: FAIL because current probe response only returns generic connect/auth codes.

- [ ] **Step 3: Add structured host-key payload fields to `ProbeResp`**

```go
type HostKeyTrustHint struct {
	Host              string   `json:"host"`
	Port              int      `json:"port"`
	Algorithm         string   `json:"algorithm"`
	FingerprintSHA256 string   `json:"fingerprint_sha256"`
	PublicKey         string   `json:"public_key"`
	KnownHostsPath    string   `json:"known_hosts_path,omitempty"`
	TrustedFingerprints []string `json:"trusted_fingerprints,omitempty"`
}

type ProbeResp struct {
	ProbeToken string            `json:"probe_token"`
	Reachable  bool              `json:"reachable"`
	LatencyMS  int64             `json:"latency_ms"`
	Facts      ProbeFacts        `json:"facts"`
	Warnings   []string          `json:"warnings"`
	ErrorCode  string            `json:"error_code,omitempty"`
	Message    string            `json:"message,omitempty"`
	HostKey    *HostKeyTrustHint `json:"host_key,omitempty"`
	ExpiresAt  time.Time         `json:"expires_at"`
}
```

- [ ] **Step 4: Map unknown/mismatch/revoked host-key failures in `Probe`**

```go
if err != nil {
	resp.ErrorCode, resp.Message = mapProbeError(err)
	if trustHint := hostKeyTrustHintFromError(req.IP, req.Port, err); trustHint != nil {
		resp.ErrorCode = trustHint.ErrorCode
		resp.Message = trustHint.Message
		resp.HostKey = trustHint.HostKey
	}
}
```

- [ ] **Step 5: Preserve the richer probe detail when `UpdateCredentials` fails**

```go
if !resp.Reachable {
	if strings.TrimSpace(resp.Message) != "" {
		return &backup, resp, fmt.Errorf("credential probe failed: %s", resp.Message)
	}
	return &backup, resp, errors.New("credential probe failed")
}
```

- [ ] **Step 6: Run host logic tests**

Run: `go test ./internal/modules/host/logic -run 'TestUpdateCredentials_EncryptsSSHPassword|TestUpdateCredentials_ReturnsProbeFailureDetail|TestUpdateCredentials_ReturnsStructuredHostKeyFailure' -v`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/modules/host/logic/host_service.go internal/modules/host/logic/probe.go internal/modules/host/logic/onboarding.go internal/modules/host/logic/probe_test.go
git commit -m "feat(host): expose structured host-key trust hints from probe flows"
```

### Task 4: Implement trusted host key service and trust endpoint

**Files:**
- Create: `internal/modules/host/logic/trusted_host_keys.go`
- Modify: `internal/modules/host/handler/common.go`
- Modify: `internal/modules/host/handler/host_mutation.go`
- Modify: `internal/modules/host/api/routes.go`
- Create: `internal/modules/host/handler/trusted_host_keys_test.go`

- [ ] **Step 1: Write a failing handler test for trust creation**

```go
func TestTrustHostKey_CreatesTrustedEntry(t *testing.T) {
	db, hostSvc := newTrustedHostKeyHandlerTestDeps(t)
	seedHost(t, db, 10, "118.193.38.89", 13012)

	body := `{"host":"118.193.38.89","port":13012,"algorithm":"ssh-ed25519","fingerprint_sha256":"SHA256:test","public_key":"ssh-ed25519 AAAATEST","replace_existing":false}`
	ctx, recorder := newHostMutationTestContext(http.MethodPost, "/api/v1/hosts/10/trust-host-key", strings.NewReader(body), gin.Params{{Key: "id", Value: "10"}}, 1)

	h := &Handler{svcCtx: &svc.ServiceContext{DB: db}, hostService: hostSvc}
	h.TrustHostKey(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}
```

- [ ] **Step 2: Run the failing handler test**

Run: `go test ./internal/modules/host/handler -run TestTrustHostKey_CreatesTrustedEntry -v`  
Expected: FAIL because endpoint and service do not exist yet.

- [ ] **Step 3: Add trusted-host-key service primitives**

```go
type TrustHostKeyReq struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Algorithm         string `json:"algorithm"`
	FingerprintSHA256 string `json:"fingerprint_sha256"`
	PublicKey         string `json:"public_key"`
	ReplaceExisting   bool   `json:"replace_existing"`
}

func (s *HostService) TrustHostKey(ctx context.Context, hostID, operator uint64, req TrustHostKeyReq) (*model.TrustedHostKey, error) {
	// validate host, persist trusted/rotated state, sync known_hosts atomically
}

func (s *HostService) ListTrustedHostKeys(ctx context.Context, hostID uint64) ([]model.TrustedHostKey, error) {
	// ordered by confirmed_at desc
}
```

- [ ] **Step 4: Add a handler and route**

```go
func (h *Handler) TrustHostKey(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:trust_host_key", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req hostlogic.TrustHostKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	item, err := h.hostService.TrustHostKey(c.Request.Context(), id, getUID(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, item)
}
```

Route:

```go
g.POST("/:id/trust-host-key", h.TrustHostKey)
g.GET("/:id/trusted-host-keys", h.ListTrustedHostKeys)
```

- [ ] **Step 5: Implement `known_hosts` synchronization**

```go
func syncKnownHostsEntry(path string, host string, port int, publicKey string) error {
	// read file, remove prior lines for [host]:port and host:port, append canonical line, write temp file, rename atomically
}
```

- [ ] **Step 6: Run handler tests**

Run: `go test ./internal/modules/host/handler -run 'TestTrustHostKey_CreatesTrustedEntry|TestTrustHostKey_RotatesExistingEntry' -v`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/modules/host/logic/trusted_host_keys.go internal/modules/host/handler/common.go internal/modules/host/handler/host_mutation.go internal/modules/host/api/routes.go internal/modules/host/handler/trusted_host_keys_test.go
git commit -m "feat(host): add explicit host-key trust endpoint and persistence"
```

### Task 5: Apply structured host-key handling to all host SSH entrypoints

**Files:**
- Modify: `internal/modules/host/handler/host_mutation.go`
- Modify: `internal/modules/host/handler/host_exec.go`
- Modify: `internal/modules/host/handler/terminal_session.go`
- Modify: `internal/modules/host/handler/files_handler.go`
- Test: `internal/modules/host/handler/trusted_host_keys_test.go`

- [ ] **Step 1: Write a failing handler test for health-check host-key payload**

```go
func TestHealthCheck_ReturnsHostKeyTrustPayload(t *testing.T) {
	// stub host service / node + unknown host key error
	// assert response.data.error_message contains actionable message
	// assert response.data.host_key.fingerprint_sha256 is present
}
```

- [ ] **Step 2: Run the focused test to verify failure**

Run: `go test ./internal/modules/host/handler -run TestHealthCheck_ReturnsHostKeyTrustPayload -v`  
Expected: FAIL because handlers currently only pass flat strings.

- [ ] **Step 3: Add a shared host-key error response helper**

```go
func hostKeyErrorPayload(message string, hostKey any) gin.H {
	return gin.H{
		"reachable": false,
		"message": message,
		"host_key": hostKey,
	}
}
```

- [ ] **Step 4: Apply the helper to SSH check, health check, terminal, and file operations**

```go
if trustErr := hostlogic.HostKeyTrustErrorFromError(err); trustErr != nil {
	httpx.OK(c, hostKeyErrorPayload(trustErr.Message, trustErr.HostKey))
	return
}
```

- [ ] **Step 5: Run handler coverage**

Run: `go test ./internal/modules/host/handler -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/host/handler/host_mutation.go internal/modules/host/handler/host_exec.go internal/modules/host/handler/terminal_session.go internal/modules/host/handler/files_handler.go internal/modules/host/handler/trusted_host_keys_test.go
git commit -m "feat(host): unify host-key trust responses across ssh-backed handlers"
```

### Task 6: Add frontend trust API and retry hook

**Files:**
- Modify: `web/src/api/modules/hosts.ts`
- Create: `web/src/hooks/useHostKeyTrust.ts`
- Create: `web/src/hooks/useHostKeyTrust.test.tsx`

- [ ] **Step 1: Write the failing hook test**

```tsx
it('calls trustHostKey then retries original operation exactly once', async () => {
  const trustHostKey = vi.fn().mockResolvedValue({ data: {} });
  const original = vi.fn()
    .mockRejectedValueOnce({
      businessCode: 2000,
      message: 'ssh host key verification failed',
      response: {
        data: {
          data: {
            error_type: 'ssh_host_key_unknown',
            host_key: {
              host: '118.193.38.89',
              port: 13012,
              fingerprint_sha256: 'SHA256:test',
              algorithm: 'ssh-ed25519',
              public_key: 'ssh-ed25519 AAAATEST',
            },
          },
        },
      },
    })
    .mockResolvedValueOnce({ data: { reachable: true } });

  // render hook, confirm trust, verify retry count == 2 total calls
})
```

- [ ] **Step 2: Run the hook test to verify failure**

Run: `npm test -- src/hooks/useHostKeyTrust.test.tsx --silent`  
Expected: FAIL because hook and trust API do not exist yet.

- [ ] **Step 3: Extend `hosts.ts` with trust types and API methods**

```ts
export interface HostKeyTrustPayload {
  host: string;
  port: number;
  algorithm: string;
  fingerprintSha256: string;
  publicKey: string;
  trustedFingerprints?: string[];
}

export interface HostKeyTrustErrorData {
  errorType: 'ssh_host_key_unknown' | 'ssh_host_key_mismatch' | 'ssh_host_key_revoked';
  hostKey: HostKeyTrustPayload;
}

async trustHostKey(id: string, payload: HostKeyTrustPayload & { replaceExisting?: boolean }) {
  return apiService.post(`/hosts/${id}/trust-host-key`, {
    host: payload.host,
    port: payload.port,
    algorithm: payload.algorithm,
    fingerprint_sha256: payload.fingerprintSha256,
    public_key: payload.publicKey,
    replace_existing: !!payload.replaceExisting,
  });
}
```

- [ ] **Step 4: Implement the shared hook**

```ts
export function useHostKeyTrust(hostId: string) {
  const [pendingTrust, setPendingTrust] = useState<HostKeyTrustErrorData | null>(null);

  const runWithTrustRetry = useCallback(async <T,>(operation: () => Promise<T>) => {
    try {
      return await operation();
    } catch (error) {
      const trust = parseHostKeyTrustError(error);
      if (!trust) {
        throw error;
      }
      setPendingTrust(trust);
      throw error;
    }
  }, []);

  const confirmTrustAndRetry = useCallback(async (retry: () => Promise<void>) => {
    // call hostApi.trustHostKey, clear pending state, retry once
  }, [hostId]);

  return { pendingTrust, setPendingTrust, runWithTrustRetry, confirmTrustAndRetry };
}
```

- [ ] **Step 5: Run frontend hook tests**

Run: `npm test -- src/hooks/useHostKeyTrust.test.tsx --silent`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/api/modules/hosts.ts web/src/hooks/useHostKeyTrust.ts web/src/hooks/useHostKeyTrust.test.tsx
git commit -m "feat(web): add shared host-key trust retry hook"
```

### Task 7: Add shared trust modal and wire host pages

**Files:**
- Create: `web/src/components/Hosts/HostKeyTrustModal.tsx`
- Modify: `web/src/pages/Hosts/HostOnboardingPage.tsx`
- Modify: `web/src/pages/Hosts/HostDetailPage.tsx`
- Modify: `web/src/pages/Hosts/HostTerminalPage.tsx`
- Test: `web/src/pages/Hosts/HostDetailPage.test.tsx`

- [ ] **Step 1: Write the failing HostDetailPage test**

```tsx
it('offers host-key trust confirmation when health check returns unknown host key', async () => {
  mockApi.hosts.runHealthCheck.mockResolvedValueOnce({
    data: {
      state: 'critical',
      connectivity_status: 'critical',
      error_message: 'ssh host key verification failed',
      host_key: {
        host: '118.193.38.89',
        port: 13012,
        algorithm: 'ssh-ed25519',
        fingerprint_sha256: 'SHA256:test',
        public_key: 'ssh-ed25519 AAAATEST',
      },
    },
  });

  render(...);
  fireEvent.click(screen.getByRole('button', { name: '健康检查' }));
  expect(await screen.findByText('信任此主机指纹？')).toBeInTheDocument();
})
```

- [ ] **Step 2: Run the page test and verify failure**

Run: `npm test -- src/pages/Hosts/HostDetailPage.test.tsx --silent`  
Expected: FAIL because no trust modal exists yet.

- [ ] **Step 3: Implement the modal**

```tsx
export default function HostKeyTrustModal(props: {
  open: boolean;
  loading?: boolean;
  mode: 'create' | 'rotate';
  hostKey: HostKeyTrustPayload | null;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <Modal open={props.open} onCancel={props.onCancel} onOk={props.onConfirm} okText={props.mode === 'rotate' ? '确认替换' : '确认信任'}>
      <Descriptions bordered column={1} size="small">
        <Descriptions.Item label="主机">{props.hostKey?.host}:{props.hostKey?.port}</Descriptions.Item>
        <Descriptions.Item label="算法">{props.hostKey?.algorithm || '-'}</Descriptions.Item>
        <Descriptions.Item label="SHA256 指纹">{props.hostKey?.fingerprintSha256 || '-'}</Descriptions.Item>
      </Descriptions>
    </Modal>
  );
}
```

- [ ] **Step 4: Wire `HostDetailPage` health check and credential update through the hook**

```tsx
const { pendingTrust, runWithTrustRetry, confirmTrustAndRetry, setPendingTrust } = useHostKeyTrust(id);

const runHealthCheck = async () => {
  await runWithTrustRetry(async () => {
    const res = await Api.hosts.runHealthCheck(id, true);
    // existing modal rendering
  });
};
```

- [ ] **Step 5: Wire onboarding probe and terminal connect paths**

```tsx
// HostOnboardingPage
await runWithTrustRetry(async () => {
  const result = await Api.hosts.probeHost(...);
  setProbeResult(result.data);
});

// HostTerminalPage
await runWithTrustRetry(async () => {
  const sessResp = await Api.hosts.createTerminalSession(id);
  // existing websocket setup
});
```

- [ ] **Step 6: Render the modal in each page and retry once after confirm**

```tsx
<HostKeyTrustModal
  open={Boolean(pendingTrust)}
  mode={pendingTrust?.errorType === 'ssh_host_key_mismatch' ? 'rotate' : 'create'}
  hostKey={pendingTrust?.hostKey || null}
  onCancel={() => setPendingTrust(null)}
  onConfirm={() => void confirmTrustAndRetry(lastOperationRef.current)}
/>;
```

- [ ] **Step 7: Run affected frontend tests**

Run: `npm test -- src/pages/Hosts/HostDetailPage.test.tsx src/pages/Hosts/HostTerminalPage.test.tsx --silent`  
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add web/src/components/Hosts/HostKeyTrustModal.tsx web/src/pages/Hosts/HostOnboardingPage.tsx web/src/pages/Hosts/HostDetailPage.tsx web/src/pages/Hosts/HostTerminalPage.tsx web/src/pages/Hosts/HostDetailPage.test.tsx
git commit -m "feat(web): add explicit host-key trust confirmation flow"
```

### Task 8: Full verification and docs

**Files:**
- Modify: `README.md`
- Modify: `docs/reviews/2026-04-14-full-architecture-security-review.md`

- [ ] **Step 1: Document operator workflow**

```md
### SSH Host Key Trust

- OpsPilot keeps strict SSH host key verification enabled.
- First-seen host keys must be explicitly trusted in the product before SSH-backed host actions proceed.
- Trusted host keys are stored in OpsPilot and synchronized to the runtime `known_hosts` file.
```

- [ ] **Step 2: Mark the review follow-up evidence**

```md
- Host key verification now uses explicit trust confirmation instead of manual-only `known_hosts` maintenance.
- Coverage: onboarding probe, health check, credential update, SSH check, terminal, and file flows.
```

- [ ] **Step 3: Run backend focused/full verification**

Run: `go test ./internal/client/ssh ./internal/modules/user/handler ./internal/modules/host/logic ./internal/modules/host/handler -v && go test ./...`  
Expected: PASS

- [ ] **Step 4: Run frontend focused/full verification**

Run: `cd web && npm test -- src/hooks/useHostKeyTrust.test.tsx src/pages/Hosts/HostDetailPage.test.tsx src/pages/Hosts/HostTerminalPage.test.tsx --silent && npm test -- --silent`  
Expected: PASS

- [ ] **Step 5: Run build verification**

Run: `make web-build && make build`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add README.md docs/reviews/2026-04-14-full-architecture-security-review.md
git commit -m "docs(host): record host-key trust workflow and verification evidence"
```
