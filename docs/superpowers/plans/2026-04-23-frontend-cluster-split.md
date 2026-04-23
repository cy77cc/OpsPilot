# Frontend cluster.ts Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split `web/src/api/modules/cluster.ts` (1662 lines) into functional domain modules, remove normalizers, and update all import paths.

**Architecture:** Create `types/` and `operations/` subdirectories under `cluster/`, move type definitions to types files, move API functions to operations files, delete ~550 lines of normalizers (handled by backend now).

**Tech Stack:** TypeScript, React, Vite

---

## File Structure

### Files to Create

| File | Purpose | Est. Lines |
|------|---------|------------|
| `web/src/api/modules/cluster/types/index.ts` | Unified type exports | ~20 |
| `web/src/api/modules/cluster/types/bootstrap.types.ts` | Bootstrap types | ~80 |
| `web/src/api/modules/cluster/types/node.types.ts` | Node operation types | ~60 |
| `web/src/api/modules/cluster/types/workload.types.ts` | Workload types | ~50 |
| `web/src/api/modules/cluster/types/network.types.ts` | Service/Ingress types | ~40 |
| `web/src/api/modules/cluster/types/policy.types.ts` | Network policy types | ~80 |
| `web/src/api/modules/cluster/types/operation.types.ts` | Operation/approval types | ~70 |
| `web/src/api/modules/cluster/types/resource.types.ts` | K8s resource types | ~60 |
| `web/src/api/modules/cluster/operations/index.ts` | Unified API exports | ~20 |
| `web/src/api/modules/cluster/operations/bootstrap.api.ts` | Bootstrap API functions | ~100 |
| `web/src/api/modules/cluster/operations/node.api.ts` | Node API functions | ~150 |
| `web/src/api/modules/cluster/operations/workload.api.ts` | Workload API functions | ~120 |
| `web/src/api/modules/cluster/operations/network.api.ts` | Network API functions | ~100 |
| `web/src/api/modules/cluster/operations/policy.api.ts` | Policy API functions | ~150 |
| `web/src/api/modules/cluster/operations/operation.api.ts` | Operation history API | ~80 |
| `web/src/api/modules/cluster/operations/resource.api.ts` | Resource query API | ~150 |
| `web/src/api/modules/cluster/index.ts` | Module entry point | ~30 |

### Files to Modify

| File | Change |
|------|--------|
| `web/src/api/modules/cluster.ts` | Delete entire file after migration |
| `web/src/api/index.ts` | Update import path |
| `web/src/pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx` | Update import path |
| `web/src/pages/Deployment/Infrastructure/hooks/useClusterDetailPageOperations.tsx` | Update import path |
| `web/src/pages/Deployment/Infrastructure/hooks/useClusterResources.ts` | Update import path |
| `web/src/components/K8s/RolloutPanel.tsx` | Update import path |
| `web/src/components/K8s/HPAEditor.tsx` | Update import path |
| `web/src/components/K8s/QuotaEditor.tsx` | Update import path |
| `web/src/components/K8s/NamespacePolicyPanel.tsx` | Update import path |
| `web/src/pages/Services/ServiceDetailPage.tsx` | Update import path |

---

## Task 1: Create Types Directory Structure

**Files:**
- Create: `web/src/api/modules/cluster/types/`

- [ ] **Step 1: Create types directory**

```bash
mkdir -p web/src/api/modules/cluster/types
```

- [ ] **Step 2: Commit directory structure**

```bash
git add web/src/api/modules/cluster/types/
git commit -m "feat(web): create cluster types directory structure"
```

---

## Task 2: Create bootstrap.types.ts

**Files:**
- Create: `web/src/api/modules/cluster/types/bootstrap.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 64-185

- [ ] **Step 1: Extract Bootstrap types from cluster.ts**

Read cluster.ts lines 64-185 and create the types file:

```typescript
// web/src/api/modules/cluster/types/bootstrap.types.ts

export interface BootstrapPreviewReq {
  name: string;
  profile_id?: number;
  control_plane_host_id: number;
  worker_host_ids?: number[];
  k8s_version?: string;
  version_channel?: string;
  cni?: string;
  pod_cidr?: string;
  service_cidr?: string;
  repo_mode?: 'online' | 'mirror';
  repo_url?: string;
  image_repository?: string;
  endpoint_mode?: 'nodeIP' | 'vip' | 'lbDNS';
  control_plane_endpoint?: string;
  vip_provider?: 'kube-vip' | 'keepalived';
  etcd_mode?: 'stacked' | 'external';
  external_etcd?: {
    endpoints?: string[];
    ca_cert?: string;
    cert?: string;
    key?: string;
  };
}

export interface BootstrapValidationIssue {
  field: string;
  code: string;
  domain?: string;
  message: string;
  remediation?: string;
}

export interface BootstrapProfile {
  id: number;
  name: string;
  description?: string;
  version_channel: string;
  k8s_version: string;
  repo_mode: 'online' | 'mirror';
  repo_url?: string;
  image_repository?: string;
  endpoint_mode: 'nodeIP' | 'vip' | 'lbDNS';
  control_plane_endpoint?: string;
  vip_provider?: 'kube-vip' | 'keepalived';
  etcd_mode: 'stacked' | 'external';
  external_etcd?: {
    endpoints?: string[];
    ca_cert?: string;
    cert?: string;
    key?: string;
  };
  created_at: string;
  updated_at: string;
}

export interface BootstrapVersionItem {
  version: string;
  channel: string;
  status: 'supported' | 'blocked';
  reason?: string;
}

export interface BootstrapPreviewResp {
  name: string;
  control_plane_host_id: number;
  worker_host_ids: number[];
  k8s_version: string;
  version_channel: string;
  cni: string;
  pod_cidr: string;
  service_cidr: string;
  repo_mode: string;
  repo_url: string;
  image_repository: string;
  endpoint_mode: string;
  control_plane_endpoint: string;
  vip_provider: string;
  etcd_mode: string;
  steps: string[];
  expected_endpoint: string;
  warnings?: string[];
  validation_issues?: BootstrapValidationIssue[];
  diagnostics?: Record<string, unknown>;
}

export interface BootstrapStepStatus {
  name: string;
  status: string;
  message?: string;
  started_at?: string;
  finished_at?: string;
  host_id?: number;
  output?: string;
}

export interface BootstrapTask {
  id: string;
  name: string;
  cluster_id?: number;
  k8s_version: string;
  version_channel: string;
  repo_mode: string;
  repo_url: string;
  image_repository: string;
  endpoint_mode: string;
  control_plane_endpoint: string;
  vip_provider: string;
  etcd_mode: string;
  cni: string;
  pod_cidr: string;
  service_cidr: string;
  status: string;
  steps: BootstrapStepStatus[];
  current_step: number;
  error_message?: string;
  resolved_config_json?: string;
  diagnostics_json?: string;
  created_at: string;
  updated_at: string;
}

export interface BootstrapApplyResp {
  task_id: string;
  status: string;
  cluster_id?: number;
}
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

Expected: No errors (file is not imported yet)

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/bootstrap.types.ts
git commit -m "feat(web): extract bootstrap types from cluster.ts"
```

---

## Task 3: Create node.types.ts

**Files:**
- Create: `web/src/api/modules/cluster/types/node.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 24-62, 247-278

- [ ] **Step 1: Extract Node types from cluster.ts**

```typescript
// web/src/api/modules/cluster/types/node.types.ts

export interface ClusterNode {
  id: number;
  cluster_id: number;
  host_id?: number;
  host_name?: string;
  name: string;
  ip: string;
  role: string;
  status: string;
  kubelet_version?: string;
  kube_proxy_version?: string;
  container_runtime?: string;
  os_image?: string;
  kernel_version?: string;
  allocatable_cpu?: string;
  allocatable_mem?: string;
  allocatable_pods?: number;
  labels?: Record<string, string>;
  taints?: Taint[];
  conditions?: NodeCondition[];
  joined_at?: string;
  last_seen_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Taint {
  key: string;
  value: string;
  effect: string;
}

export interface NodeCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  last_transition_time?: string;
}

export interface ClusterNodeApprovalPayload {
  approval_token?: string;
}

export interface ClusterNodeDrainPayload extends ClusterNodeApprovalPayload {
  delete_emptydir_data?: boolean;
  force?: boolean;
  ignore_daemonsets?: boolean;
  grace_period_seconds?: number;
  timeout_seconds?: number;
}

export interface ClusterNodeTaintPayload extends ClusterNodeApprovalPayload {
  key: string;
  value?: string;
  effect?: string;
}

export interface ClusterNodeLabelPayload extends ClusterNodeApprovalPayload {
  key: string;
  value?: string;
}

export interface AddNodeReq {
  host_ids: number[];
  role?: string;
}
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/node.types.ts
git commit -m "feat(web): extract node types from cluster.ts"
```

---

## Task 4: Create operation.types.ts

**Files:**
- Create: `web/src/api/modules/cluster/types/operation.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 212-364

- [ ] **Step 1: Extract Operation types from cluster.ts**

```typescript
// web/src/api/modules/cluster/types/operation.types.ts

export type ClusterOperationState = 'completed' | 'approval_required' | 'rejected' | 'failed';

export interface ClusterOperationApproval {
  required: boolean;
  ticket?: string;
  cluster_id?: number;
  namespace?: string;
  action?: string;
  resource?: string;
  resource_id?: string;
  expires_at?: string;
  reason?: string;
  status?: string;
  consumed_at?: string;
  consumed_by?: number;
  replay_count?: number;
  replay_at?: string;
  replay_by?: number;
  replay_code?: string;
  replay_message?: string;
}

export interface ClusterOperationResponse<T = unknown> {
  state: ClusterOperationState;
  success: boolean;
  code: string;
  message: string;
  audit_id?: string | number;
  approval?: ClusterOperationApproval;
  error_code?: string;
  diagnostics?: string[];
  result?: T;
  raw?: Record<string, unknown>;
}

export interface ClusterOperationHistoryQuery {
  page?: number;
  page_size?: number;
  resource?: string;
  status?: string;
  operator?: string;
  from?: string;
  to?: string;
}

export interface ClusterOperationHistoryItem {
  audit_id: string | number;
  resource_type?: string;
  resource_name?: string;
  resource?: string;
  action: string;
  status: string;
  operator?: string;
  message?: string;
  approval_required?: boolean;
  approval_ticket?: string;
  created_at: string;
  updated_at?: string;
  namespace?: string;
  target?: string;
}

export interface ClusterOperationDetail extends ClusterOperationHistoryItem {
  approval?: ClusterOperationApproval & {
    approver?: string;
    approved_at?: string;
    rejected_by?: string;
    rejected_at?: string;
    ticket?: string;
  };
  request?: Record<string, unknown>;
  response?: Record<string, unknown>;
  diagnostics?: unknown[];
  timeline?: Array<{
    at?: string;
    message?: string;
    status?: string;
    level?: string;
  }>;
}
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/operation.types.ts
git commit -m "feat(web): extract operation types from cluster.ts"
```

---

## Task 5: Create policy.types.ts

**Files:**
- Create: `web/src/api/modules/cluster/types/policy.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 600-700

- [ ] **Step 1: Extract Policy types from cluster.ts**

```typescript
// web/src/api/modules/cluster/types/policy.types.ts

export interface ClusterPolicyIssue {
  code?: string;
  message: string;
  severity?: string;
  suggestion?: string;
}

export interface ClusterPolicyWarning {
  code?: string;
  message: string;
}

export interface ClusterPolicyImpactSummary {
  affected_pods: number;
  affected_namespaces: string[];
  new_denied_flows: string[];
}

export interface ClusterPolicySimulationStatus {
  job_id?: string;
  passed_at?: string;
  blocking_issues: ClusterPolicyIssue[];
  warnings: ClusterPolicyWarning[];
  impact_summary?: ClusterPolicyImpactSummary;
}

export interface ClusterPolicySimulationResult extends ClusterPolicySimulationStatus {
  passed: boolean;
  risk_score?: number;
  risk_level?: string;
}

export interface ClusterPolicyReference {
  api_version?: string;
  kind?: string;
  name?: string;
  namespace?: string;
}

export interface ClusterPolicyTargetCluster {
  cluster_id?: number;
  cni_type?: string;
  cni_version?: string;
}

export interface ClusterPolicyReleaseStatus {
  phase?: string;
  risk_score?: number;
  risk_level?: string;
}

export interface ClusterPolicyApprovalStatus {
  required?: boolean;
  approvers?: string[];
  approved_at?: string;
  approval_token?: string;
}

export interface ClusterPolicyAuditStatus {
  created_at?: string;
  created_by?: number;
  applied_at?: string;
  rollback_at?: string;
}

export interface ClusterPolicyRelease {
  release_id: number;
  version: string;
  previous_stable_version?: string;
  rollback_target_version?: string;
  policy?: ClusterPolicyReference;
  target_cluster?: ClusterPolicyTargetCluster;
  status?: ClusterPolicyReleaseStatus;
  simulation?: ClusterPolicySimulationStatus;
  approval?: ClusterPolicyApprovalStatus;
  audit?: ClusterPolicyAuditStatus;
  last_error_code?: string;
  last_error_message?: string;
}

export interface ClusterPolicySimulationPayload {
  base_version?: string;
  candidate_version: string;
  cluster?: {
    cni_type?: string;
    namespaces?: string[];
  };
}

export interface ClusterPolicyReleaseCreatePayload {
  version: string;
  previous_stable_version?: string;
}

export interface ClusterPolicyReleaseActionPayload {
  version?: string;
  approval_token?: string;
  rollback_target_version?: string;
}
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/policy.types.ts
git commit -m "feat(web): extract policy types from cluster.ts"
```

---

## Task 6: Create resource.types.ts

**Files:**
- Create: `web/src/api/modules/cluster/types/resource.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 365-595

- [ ] **Step 1: Extract Resource types from cluster.ts**

```typescript
// web/src/api/modules/cluster/types/resource.types.ts

export interface NamespaceInfo {
  name: string;
  status: string;
  labels?: Record<string, string>;
  created_at: string;
}

export interface PodInfo {
  name: string;
  namespace: string;
  status: string;
  pod_ip: string;
  node_name: string;
  ready: string;
  restarts: number;
  age: string;
  labels?: Record<string, string>;
  created_at: string;
}

export interface DeploymentInfo {
  name: string;
  namespace: string;
  replicas: number;
  ready: number;
  updated: number;
  available: number;
  age: string;
  created_at: string;
}

export interface StatefulSetInfo {
  name: string;
  namespace: string;
  replicas: number;
  ready: number;
  age: string;
  created_at: string;
}

export interface DaemonSetInfo {
  name: string;
  namespace: string;
  desired: number;
  ready: number;
  age: string;
  created_at: string;
}

export interface JobInfo {
  name: string;
  namespace: string;
  completions: number;
  succeeded: number;
  failed: number;
  status: string;
  age: string;
  created_at: string;
}

export interface ServiceInfo {
  name: string;
  namespace: string;
  type: string;
  cluster_ip: string;
  ports: ServicePort[];
  selector?: Record<string, string>;
  age: string;
  created_at: string;
}

export interface ServicePort {
  name: string;
  port: number;
  target_port: string;
  protocol: string;
}

export interface IngressInfo {
  name: string;
  namespace: string;
  hosts: IngressHost[];
  age: string;
  created_at: string;
}

export interface IngressHost {
  host: string;
  paths: string[];
}

export interface ConfigMapInfo {
  name: string;
  namespace: string;
  data_keys: string[];
  age: string;
  created_at: string;
}

export interface SecretInfo {
  name: string;
  namespace: string;
  type: string;
  data_keys: string[];
  age: string;
  created_at: string;
}

export interface PVCInfo {
  name: string;
  namespace: string;
  status: string;
  capacity: string;
  access_modes: string;
  storage_class: string;
  volume_name: string;
  age: string;
  created_at: string;
}

export interface PVInfo {
  name: string;
  status: string;
  capacity: string;
  access_modes: string;
  storage_class: string;
  claim_ref: string;
  age: string;
  created_at: string;
}

export interface ClusterServiceInfo {
  id: number;
  name: string;
  project_name: string;
  team_name: string;
  env: string;
  last_deploy_at: string;
  status: string;
}

export interface EventInfo {
  name: string;
  namespace: string;
  type: string;
  reason: string;
  message: string;
  source: string;
  count: number;
  age: string;
  first_seen: string;
  last_seen: string;
}

export interface HPAInfo {
  name: string;
  namespace: string;
  reference: string;
  min_replicas: number;
  max_replicas: number;
  current_cpu: string;
  target_cpu: string;
  current_mem: string;
  target_mem: string;
  replicas: number;
  metrics: HPAMetricInfo[];
  age: string;
  created_at: string;
}

export interface HPAMetricInfo {
  name: string;
  type: string;
  current: string;
  target: string;
}

export interface ResourceQuotaInfo {
  name: string;
  namespace: string;
  hard: Record<string, string>;
  used: Record<string, string>;
  age: string;
  created_at: string;
}

export interface LimitRangeInfo {
  name: string;
  namespace: string;
  type: string;
  limits: LimitRangeItem[];
  age: string;
  created_at: string;
}

export interface LimitRangeItem {
  type: string;
  max: Record<string, string>;
  min: Record<string, string>;
  default: Record<string, string>;
  default_request: Record<string, string>;
}

export interface ClusterVersionInfo {
  kubernetes_version: string;
  git_version: string;
  platform: string;
  go_version: string;
}

export interface ClusterUpgradePlan {
  current_version: string;
  target_version: string;
  upgradable: boolean;
  steps: string[];
  warnings: string[];
}

export interface CertificateInfo {
  name: string;
  expires_at: string;
  days_left: number;
  ca: boolean;
  alternate_names: string[];
}

export interface ClusterCNIInfo {
  cluster_id: number;
  cni_type?: string;
  cni_version?: string;
  capabilities: Record<string, boolean>;
  constraints?: Record<string, unknown>;
}
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/resource.types.ts
git commit -m "feat(web): extract resource types from cluster.ts"
```

---

## Task 7: Create workload.types.ts

**Files:**
- Create: `web/src/api/modules/cluster/types/workload.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 276-285

- [ ] **Step 1: Extract Workload types**

```typescript
// web/src/api/modules/cluster/types/workload.types.ts

import type { ClusterNodeApprovalPayload } from './node.types';

export interface ClusterWorkloadScalePayload extends ClusterNodeApprovalPayload {
  replicas: number;
}
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/workload.types.ts
git commit -m "feat(web): extract workload types from cluster.ts"
```

---

## Task 8: Create network.types.ts

**Files:**
- Create: `web/src/api/modules/cluster/types/network.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 280-317

- [ ] **Step 1: Extract Network types**

```typescript
// web/src/api/modules/cluster/types/network.types.ts

import type { ClusterNodeApprovalPayload } from './node.types';

export interface ClusterServiceMutationPort {
  name?: string;
  port: number;
  target_port: string;
  protocol?: string;
  node_port?: number;
}

export interface ClusterServiceMutationPayload extends ClusterNodeApprovalPayload {
  name: string;
  type: string;
  selector: Record<string, string>;
  ports: ClusterServiceMutationPort[];
}

export interface ClusterIngressMutationPath {
  path: string;
  path_type?: string;
  service_name: string;
  service_port: number;
}

export interface ClusterIngressMutationRule {
  host: string;
  paths: ClusterIngressMutationPath[];
}

export interface ClusterIngressMutationTLS {
  secret_name?: string;
  hosts: string[];
}

export interface ClusterIngressMutationPayload extends ClusterNodeApprovalPayload {
  name: string;
  ingress_class_name?: string;
  rules: ClusterIngressMutationRule[];
  tls?: ClusterIngressMutationTLS[];
}
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/network.types.ts
git commit -m "feat(web): extract network types from cluster.ts"
```

---

## Task 9: Create Cluster base types

**Files:**
- Create: `web/src/api/modules/cluster/types/cluster.types.ts`
- Source: `web/src/api/modules/cluster.ts` lines 4-22, 186-210

- [ ] **Step 1: Extract Cluster base types**

```typescript
// web/src/api/modules/cluster/types/cluster.types.ts

export interface Cluster {
  id: number;
  name: string;
  description?: string;
  version?: string;
  k8s_version?: string;
  status: string;
  source: string;
  type: string;
  node_count: number;
  endpoint?: string;
  pod_cidr?: string;
  service_cidr?: string;
  management_mode?: string;
  credential_id?: number;
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ClusterImportReq {
  name: string;
  description?: string;
  kubeconfig?: string;
  endpoint?: string;
  ca_cert?: string;
  cert?: string;
  key?: string;
  token?: string;
  skip_tls_verify?: boolean;
  auth_method?: string;
}

export interface ClusterTestResp {
  cluster_id: number;
  connected: boolean;
  message: string;
  version?: string;
  latency_ms?: number;
}

export interface ClusterUpgradePayload extends ClusterNodeApprovalPayload {
  target_version: string;
}

export interface ClusterCertificateRenewPayload extends ClusterNodeApprovalPayload {}
```

Note: `ClusterNodeApprovalPayload` is imported from node.types.ts, add that import after types/index.ts is created.

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/types/cluster.types.ts
git commit -m "feat(web): extract cluster base types"
```

---

## Task 10: Create types/index.ts unified export

**Files:**
- Create: `web/src/api/modules/cluster/types/index.ts`

- [ ] **Step 1: Create unified type exports**

```typescript
// web/src/api/modules/cluster/types/index.ts

// Base cluster types
export type { Cluster, ClusterImportReq, ClusterTestResp } from './cluster.types';

// Bootstrap types
export type {
  BootstrapPreviewReq,
  BootstrapValidationIssue,
  BootstrapProfile,
  BootstrapVersionItem,
  BootstrapPreviewResp,
  BootstrapStepStatus,
  BootstrapTask,
  BootstrapApplyResp,
} from './bootstrap.types';

// Node types
export type {
  ClusterNode,
  Taint,
  NodeCondition,
  ClusterNodeApprovalPayload,
  ClusterNodeDrainPayload,
  ClusterNodeTaintPayload,
  ClusterNodeLabelPayload,
  AddNodeReq,
} from './node.types';

// Workload types
export type { ClusterWorkloadScalePayload } from './workload.types';

// Network types
export type {
  ClusterServiceMutationPort,
  ClusterServiceMutationPayload,
  ClusterIngressMutationPath,
  ClusterIngressMutationRule,
  ClusterIngressMutationTLS,
  ClusterIngressMutationPayload,
} from './network.types';

// Policy types
export type {
  ClusterPolicyIssue,
  ClusterPolicyWarning,
  ClusterPolicyImpactSummary,
  ClusterPolicySimulationStatus,
  ClusterPolicySimulationResult,
  ClusterPolicyReference,
  ClusterPolicyTargetCluster,
  ClusterPolicyReleaseStatus,
  ClusterPolicyApprovalStatus,
  ClusterPolicyAuditStatus,
  ClusterPolicyRelease,
  ClusterPolicySimulationPayload,
  ClusterPolicyReleaseCreatePayload,
  ClusterPolicyReleaseActionPayload,
} from './policy.types';

// Operation types
export type {
  ClusterOperationState,
  ClusterOperationApproval,
  ClusterOperationResponse,
  ClusterOperationHistoryQuery,
  ClusterOperationHistoryItem,
  ClusterOperationDetail,
} from './operation.types';

// Resource types
export type {
  NamespaceInfo,
  PodInfo,
  DeploymentInfo,
  StatefulSetInfo,
  DaemonSetInfo,
  JobInfo,
  ServiceInfo,
  ServicePort,
  IngressInfo,
  IngressHost,
  ConfigMapInfo,
  SecretInfo,
  PVCInfo,
  PVInfo,
  ClusterServiceInfo,
  EventInfo,
  HPAInfo,
  HPAMetricInfo,
  ResourceQuotaInfo,
  LimitRangeInfo,
  LimitRangeItem,
  ClusterVersionInfo,
  ClusterUpgradePlan,
  CertificateInfo,
  ClusterCNIInfo,
} from './resource.types';
```

- [ ] **Step 2: Fix cluster.types.ts import**

Update cluster.types.ts to import ClusterNodeApprovalPayload:

```typescript
// web/src/api/modules/cluster/types/cluster.types.ts

import type { ClusterNodeApprovalPayload } from './node.types';

export interface ClusterUpgradePayload extends ClusterNodeApprovalPayload {
  target_version: string;
}

export interface ClusterCertificateRenewPayload extends ClusterNodeApprovalPayload {}
```

- [ ] **Step 3: Run TypeScript check**

```bash
cd web && npm run typecheck
```

Expected: All types resolve correctly

- [ ] **Step 4: Commit**

```bash
git add web/src/api/modules/cluster/types/
git commit -m "feat(web): create unified type exports for cluster module"
```

---

## Task 11: Create operations directory structure

**Files:**
- Create: `web/src/api/modules/cluster/operations/`

- [ ] **Step 1: Create operations directory**

```bash
mkdir -p web/src/api/modules/cluster/operations
```

- [ ] **Step 2: Commit**

```bash
git add web/src/api/modules/cluster/operations/
git commit -m "feat(web): create cluster operations directory structure"
```

---

## Task 12: Create bootstrap.api.ts

**Files:**
- Create: `web/src/api/modules/cluster/operations/bootstrap.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1543-1578

- [ ] **Step 1: Extract Bootstrap API functions**

```typescript
// web/src/api/modules/cluster/operations/bootstrap.api.ts

import apiService from '../../api';
import type { ApiResponse } from '../../api';
import type {
  BootstrapPreviewReq,
  BootstrapPreviewResp,
  BootstrapTask,
  BootstrapProfile,
  BootstrapVersionItem,
} from '../types';

export const bootstrapApi = {
  getVersions(): Promise<ApiResponse<{ default_channel: string; list: BootstrapVersionItem[] }>> {
    return apiService.get('/clusters/bootstrap/versions');
  },

  getProfiles(): Promise<ApiResponse<{ list: BootstrapProfile[]; total: number }>> {
    return apiService.get('/clusters/bootstrap/profiles');
  },

  createProfile(
    data: Omit<BootstrapProfile, 'id' | 'created_at' | 'updated_at'>
  ): Promise<ApiResponse<BootstrapProfile>> {
    return apiService.post('/clusters/bootstrap/profiles', data);
  },

  updateProfile(
    id: number,
    data: Partial<Omit<BootstrapProfile, 'id' | 'name' | 'created_at' | 'updated_at'>>
  ): Promise<ApiResponse<BootstrapProfile>> {
    return apiService.put(`/clusters/bootstrap/profiles/${id}`, data);
  },

  deleteProfile(id: number): Promise<ApiResponse<{ id: number; deleted: boolean }>> {
    return apiService.delete(`/clusters/bootstrap/profiles/${id}`);
  },

  preview(data: BootstrapPreviewReq): Promise<ApiResponse<BootstrapPreviewResp>> {
    return apiService.post('/clusters/bootstrap/preview', data);
  },

  apply(data: BootstrapPreviewReq): Promise<ApiResponse<{ task_id: string; status: string }>> {
    return apiService.post('/clusters/bootstrap/apply', data);
  },

  getTask(taskId: string): Promise<ApiResponse<BootstrapTask>> {
    return apiService.get(`/clusters/bootstrap/${encodeURIComponent(taskId)}`);
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/bootstrap.api.ts
git commit -m "feat(web): extract bootstrap API functions"
```

---

## Task 13: Create node.api.ts

**Files:**
- Create: `web/src/api/modules/cluster/operations/node.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1294-1341

- [ ] **Step 1: Extract Node API functions**

Note: These functions no longer need wrapOperationResponse since backend now returns standardized format.

```typescript
// web/src/api/modules/cluster/operations/node.api.ts

import apiService from '../../api';
import type { ApiResponse, PaginatedResponse } from '../../api';
import type {
  ClusterNode,
  AddNodeReq,
  ClusterNodeApprovalPayload,
  ClusterNodeDrainPayload,
  ClusterNodeTaintPayload,
  ClusterNodeLabelPayload,
  ClusterOperationResponse,
} from '../types';

export const nodeApi = {
  getNodes(id: number): Promise<ApiResponse<PaginatedResponse<ClusterNode>>> {
    return apiService.get(`/clusters/${id}/nodes`);
  },

  syncNodes(id: number): Promise<ApiResponse<PaginatedResponse<ClusterNode>>> {
    return apiService.post(`/clusters/${id}/nodes/sync`);
  },

  getNodeDetail(clusterId: number, nodeName: string): Promise<ApiResponse<ClusterNode>> {
    return apiService.get(`/clusters/${clusterId}/nodes/${encodeURIComponent(nodeName)}`);
  },

  addNodes(id: number, data: AddNodeReq): Promise<ApiResponse<{ results: unknown[]; message: string }>> {
    return apiService.post(`/clusters/${id}/nodes`, data);
  },

  removeNode(
    id: number,
    nodeName: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}`, { data: payload || {} });
  },

  cordonNode(
    id: number,
    nodeName: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/cordon`, payload || {});
  },

  uncordonNode(
    id: number,
    nodeName: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/uncordon`, payload || {});
  },

  drainNode(
    id: number,
    nodeName: string,
    payload?: ClusterNodeDrainPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/drain`, payload || {});
  },

  upsertTaint(
    id: number,
    nodeName: string,
    payload: ClusterNodeTaintPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/taints`, payload);
  },

  removeTaint(
    id: number,
    nodeName: string,
    payload: ClusterNodeTaintPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/taints`, { data: payload });
  },

  upsertLabel(
    id: number,
    nodeName: string,
    payload: ClusterNodeLabelPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/labels`, payload);
  },

  removeLabel(
    id: number,
    nodeName: string,
    payload: ClusterNodeLabelPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/labels`, { data: payload });
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/node.api.ts
git commit -m "feat(web): extract node API functions (simplified without normalizers)"
```

---

## Task 14: Create workload.api.ts

**Files:**
- Create: `web/src/api/modules/cluster/operations/workload.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1342-1394

- [ ] **Step 1: Extract Workload API functions**

```typescript
// web/src/api/modules/cluster/operations/workload.api.ts

import apiService from '../../api';
import type { ApiResponse, PaginatedResponse } from '../../api';
import type {
  PodInfo,
  DeploymentInfo,
  StatefulSetInfo,
  DaemonSetInfo,
  JobInfo,
  ClusterNodeApprovalPayload,
  ClusterWorkloadScalePayload,
  ClusterOperationResponse,
} from '../types';

export const workloadApi = {
  getPods(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<PodInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/pods`);
  },

  deletePod(
    id: number,
    namespace: string,
    name: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/pods/${encodeURIComponent(name)}`,
      { data: payload || {} }
    );
  },

  getDeployments(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<DeploymentInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/deployments`);
  },

  restartDeployment(
    id: number,
    namespace: string,
    name: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/deployments/${encodeURIComponent(name)}/restart`,
      payload || {}
    );
  },

  scaleDeployment(
    id: number,
    namespace: string,
    name: string,
    payload: ClusterWorkloadScalePayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/deployments/${encodeURIComponent(name)}/scale`,
      payload
    );
  },

  deleteDeployment(
    id: number,
    namespace: string,
    name: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/deployments/${encodeURIComponent(name)}`,
      { data: payload || {} }
    );
  },

  getStatefulSets(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<StatefulSetInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/statefulsets`);
  },

  restartStatefulSet(
    id: number,
    namespace: string,
    name: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/statefulsets/${encodeURIComponent(name)}/restart`,
      payload || {}
    );
  },

  scaleStatefulSet(
    id: number,
    namespace: string,
    name: string,
    payload: ClusterWorkloadScalePayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/statefulsets/${encodeURIComponent(name)}/scale`,
      payload
    );
  },

  deleteStatefulSet(
    id: number,
    namespace: string,
    name: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/statefulsets/${encodeURIComponent(name)}`,
      { data: payload || {} }
    );
  },

  getDaemonSets(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<DaemonSetInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/daemonsets`);
  },

  getJobs(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<JobInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/jobs`);
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/workload.api.ts
git commit -m "feat(web): extract workload API functions"
```

---

## Task 15: Create network.api.ts

**Files:**
- Create: `web/src/api/modules/cluster/operations/network.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1396-1427

- [ ] **Step 1: Extract Network API functions**

```typescript
// web/src/api/modules/cluster/operations/network.api.ts

import apiService from '../../api';
import type { ApiResponse, PaginatedResponse } from '../../api';
import type {
  ServiceInfo,
  IngressInfo,
  ClusterNodeApprovalPayload,
  ClusterServiceMutationPayload,
  ClusterIngressMutationPayload,
  ClusterOperationResponse,
} from '../types';

export const networkApi = {
  getServices(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<ServiceInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/services`);
  },

  createService(
    id: number,
    namespace: string,
    payload: ClusterServiceMutationPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/services`, payload);
  },

  updateService(
    id: number,
    namespace: string,
    name: string,
    payload: ClusterServiceMutationPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.put(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/services/${encodeURIComponent(name)}`,
      payload
    );
  },

  deleteService(
    id: number,
    namespace: string,
    name: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/services/${encodeURIComponent(name)}`,
      { data: payload || {} }
    );
  },

  getIngresses(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<IngressInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/ingresses`);
  },

  createIngress(
    id: number,
    namespace: string,
    payload: ClusterIngressMutationPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.post(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/ingresses`, payload);
  },

  updateIngress(
    id: number,
    namespace: string,
    name: string,
    payload: ClusterIngressMutationPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.put(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/ingresses/${encodeURIComponent(name)}`,
      payload
    );
  },

  deleteIngress(
    id: number,
    namespace: string,
    name: string,
    payload?: ClusterNodeApprovalPayload
  ): Promise<ApiResponse<ClusterOperationResponse>> {
    return apiService.delete(
      `/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/ingresses/${encodeURIComponent(name)}`,
      { data: payload || {} }
    );
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/network.api.ts
git commit -m "feat(web): extract network API functions"
```

---

## Task 16: Create policy.api.ts

**Files:**
- Create: `web/src/api/modules/cluster/operations/policy.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1429-1517

- [ ] **Step 1: Extract Policy API functions**

```typescript
// web/src/api/modules/cluster/operations/policy.api.ts

import apiService from '../../api';
import type { ApiResponse } from '../../api';
import type {
  ClusterCNIInfo,
  ClusterPolicySimulationPayload,
  ClusterPolicySimulationResult,
  ClusterPolicyReleaseCreatePayload,
  ClusterPolicyReleaseActionPayload,
  ClusterPolicyRelease,
  ClusterOperationResponse,
} from '../types';

export const policyApi = {
  getCNIInfo(id: number): Promise<ApiResponse<ClusterCNIInfo>> {
    return apiService.get(`/clusters/${id}/cni-info`);
  },

  simulatePolicy(
    id: number,
    namespace: string,
    name: string,
    payload: ClusterPolicySimulationPayload
  ): Promise<ApiResponse<ClusterPolicySimulationResult>> {
    return apiService.post(
      `/clusters/${id}/policies/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/simulate`,
      payload
    );
  },

  createPolicyRelease(
    id: number,
    namespace: string,
    name: string,
    payload: ClusterPolicyReleaseCreatePayload
  ): Promise<ApiResponse<ClusterOperationResponse<{ release?: ClusterPolicyRelease }>>> {
    return apiService.post(
      `/clusters/${id}/policies/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/releases`,
      payload
    );
  },

  getPolicyRelease(id: number, releaseId: number): Promise<ApiResponse<ClusterPolicyRelease>> {
    return apiService.get(`/clusters/${id}/releases/${releaseId}`);
  },

  applyPolicyRelease(
    id: number,
    releaseId: number,
    payload?: ClusterPolicyReleaseActionPayload
  ): Promise<ApiResponse<ClusterOperationResponse<{ release?: ClusterPolicyRelease }>>> {
    return apiService.post(`/clusters/${id}/releases/${releaseId}/apply`, payload || {});
  },

  rollbackPolicyRelease(
    id: number,
    releaseId: number,
    payload?: ClusterPolicyReleaseActionPayload
  ): Promise<ApiResponse<ClusterOperationResponse<{ release?: ClusterPolicyRelease }>>> {
    return apiService.post(`/clusters/${id}/releases/${releaseId}/rollback`, payload || {});
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/policy.api.ts
git commit -m "feat(web): extract policy API functions (simplified)"
```

---

## Task 17: Create operation.api.ts

**Files:**
- Create: `web/src/api/modules/cluster/operations/operation.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1655-1661

- [ ] **Step 1: Extract Operation history API functions**

```typescript
// web/src/api/modules/cluster/operations/operation.api.ts

import apiService from '../../api';
import type { ApiResponse, PaginatedResponse } from '../../api';
import type { ClusterOperationHistoryQuery, ClusterOperationHistoryItem, ClusterOperationDetail } from '../types';

export const operationApi = {
  getHistory(
    id: number,
    query?: ClusterOperationHistoryQuery
  ): Promise<ApiResponse<PaginatedResponse<ClusterOperationHistoryItem>>> {
    return apiService.get(`/clusters/${id}/operations/history`, { params: query });
  },

  getDetail(id: number, auditId: string | number): Promise<ApiResponse<ClusterOperationDetail>> {
    return apiService.get(`/clusters/${id}/operations/${encodeURIComponent(String(auditId))}`);
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/operation.api.ts
git commit -m "feat(web): extract operation history API functions"
```

---

## Task 18: Create resource.api.ts

**Files:**
- Create: `web/src/api/modules/cluster/operations/resource.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1342-1427, 1519-1654

- [ ] **Step 1: Extract Resource query API functions**

```typescript
// web/src/api/modules/cluster/operations/resource.api.ts

import apiService from '../../api';
import type { ApiResponse, PaginatedResponse } from '../../api';
import type {
  NamespaceInfo,
  ConfigMapInfo,
  SecretInfo,
  PVInfo,
  PVCInfo,
  ClusterServiceInfo,
  EventInfo,
  HPAInfo,
  ResourceQuotaInfo,
  LimitRangeInfo,
  ClusterVersionInfo,
  ClusterUpgradePlan,
  CertificateInfo,
  ClusterUpgradePayload,
  ClusterCertificateRenewPayload,
  ClusterOperationResponse,
} from '../types';

export const resourceApi = {
  getNamespaces(id: number): Promise<ApiResponse<PaginatedResponse<NamespaceInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces`);
  },

  getConfigMaps(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<ConfigMapInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/configmaps`);
  },

  getSecrets(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<SecretInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/secrets`);
  },

  getPVs(id: number): Promise<ApiResponse<PaginatedResponse<PVInfo>>> {
    return apiService.get(`/clusters/${id}/pvs`);
  },

  getPVCs(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<PVCInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/pvcs`);
  },

  getClusterServices(id: number): Promise<ApiResponse<PaginatedResponse<ClusterServiceInfo>>> {
    return apiService.get(`/clusters/${id}/services`);
  },

  getEvents(id: number, namespace?: string): Promise<ApiResponse<PaginatedResponse<EventInfo>>> {
    const params: Record<string, string> = {};
    if (namespace) params.namespace = namespace;
    return apiService.get(`/clusters/${id}/events`, { params });
  },

  getHPAs(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<HPAInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/hpas`);
  },

  getResourceQuotas(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<ResourceQuotaInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/resourcequotas`);
  },

  getLimitRanges(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<LimitRangeInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/limitranges`);
  },

  getVersion(id: number): Promise<ApiResponse<ClusterVersionInfo>> {
    return apiService.get(`/clusters/${id}/version`);
  },

  getUpgradePlan(id: number): Promise<ApiResponse<ClusterUpgradePlan>> {
    return apiService.get(`/clusters/${id}/upgrade-plan`);
  },

  upgrade(
    id: number,
    payload: ClusterUpgradePayload | string
  ): Promise<ApiResponse<ClusterOperationResponse<{
    cluster_id?: number;
    from_version?: string;
    to_version?: string;
    status?: string;
    message?: string;
    upgrade_steps?: string[];
  }>>> {
    const requestBody = typeof payload === 'string' ? { target_version: payload } : payload;
    return apiService.post(`/clusters/${id}/upgrade`, requestBody);
  },

  renewCertificates(
    id: number,
    payload?: ClusterCertificateRenewPayload
  ): Promise<ApiResponse<ClusterOperationResponse<{
    cluster_id?: number;
    results?: Array<{
      node_name: string;
      host_name?: string;
      success: boolean;
      message: string;
    }>;
    message?: string;
  }>>> {
    return apiService.post(`/clusters/${id}/certificates/renew`, payload || {});
  },

  getCertificates(id: number): Promise<ApiResponse<PaginatedResponse<CertificateInfo>>> {
    return apiService.get(`/clusters/${id}/certificates`);
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/resource.api.ts
git commit -m "feat(web): extract resource query API functions"
```

---

## Task 19: Create cluster CRUD API (cluster.api.ts)

**Files:**
- Create: `web/src/api/modules/cluster/operations/cluster.api.ts`
- Source: `web/src/api/modules/cluster.ts` lines 1267-1292, 1576-1598

- [ ] **Step 1: Extract Cluster CRUD API functions**

```typescript
// web/src/api/modules/cluster/operations/cluster.api.ts

import apiService from '../../api';
import type { ApiResponse, PaginatedResponse } from '../../api';
import type { Cluster, ClusterImportReq } from '../types';

export const clusterApi = {
  getClusters(params?: { status?: string; source?: string }): Promise<ApiResponse<PaginatedResponse<Cluster>>> {
    return apiService.get('/clusters', { params });
  },

  getClusterDetail(id: number): Promise<ApiResponse<Cluster>> {
    return apiService.get(`/clusters/${id}`);
  },

  createCluster(data: ClusterImportReq): Promise<ApiResponse<Cluster>> {
    return apiService.post('/clusters', data);
  },

  updateCluster(
    id: number,
    data: { name?: string; description?: string }
  ): Promise<ApiResponse<{ id: number; message: string }>> {
    return apiService.put(`/clusters/${id}`, data);
  },

  deleteCluster(id: number): Promise<ApiResponse<{ id: number; message: string }>> {
    return apiService.delete(`/clusters/${id}`);
  },

  testCluster(id: number): Promise<ApiResponse<{
    cluster_id: number;
    connected: boolean;
    message: string;
    version?: string;
    latency_ms?: number;
  }>> {
    return apiService.post(`/clusters/${id}/test`);
  },

  importCluster(data: ClusterImportReq): Promise<ApiResponse<Cluster>> {
    return apiService.post('/clusters/import', data);
  },

  validateImport(data: {
    name?: string;
    kubeconfig?: string;
    endpoint?: string;
    ca_cert?: string;
    cert?: string;
    key?: string;
    token?: string;
    skip_tls_verify?: boolean;
  }): Promise<ApiResponse<{
    valid: boolean;
    message: string;
    endpoint?: string;
    version?: string;
    auth_method?: string;
  }>> {
    return apiService.post('/clusters/import/validate', data);
  },
};
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/cluster.api.ts
git commit -m "feat(web): extract cluster CRUD API functions"
```

---

## Task 20: Create operations/index.ts unified export

**Files:**
- Create: `web/src/api/modules/cluster/operations/index.ts`

- [ ] **Step 1: Create unified API exports**

```typescript
// web/src/api/modules/cluster/operations/index.ts

import { clusterApi } from './cluster.api';
import { bootstrapApi } from './bootstrap.api';
import { nodeApi } from './node.api';
import { workloadApi } from './workload.api';
import { networkApi } from './network.api';
import { policyApi } from './policy.api';
import { operationApi } from './operation.api';
import { resourceApi } from './resource.api';

export const clusterOperations = {
  ...clusterApi,
  ...bootstrapApi,
  ...nodeApi,
  ...workloadApi,
  ...networkApi,
  ...policyApi,
  ...operationApi,
  ...resourceApi,
};

// Also export individual APIs for selective imports
export { clusterApi } from './cluster.api';
export { bootstrapApi } from './bootstrap.api';
export { nodeApi } from './node.api';
export { workloadApi } from './workload.api';
export { networkApi } from './network.api';
export { policyApi } from './policy.api';
export { operationApi } from './operation.api';
export { resourceApi } from './resource.api';
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/operations/index.ts
git commit -m "feat(web): create unified API exports"
```

---

## Task 21: Create cluster module index.ts

**Files:**
- Create: `web/src/api/modules/cluster/index.ts`

- [ ] **Step 1: Create module entry point**

```typescript
// web/src/api/modules/cluster/index.ts

// Re-export all types
export * from './types';

// Re-export all API functions
export * from './operations';

// Export combined clusterApi for backward compatibility
import { clusterOperations } from './operations';
export const clusterApi = clusterOperations;
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster/index.ts
git commit -m "feat(web): create cluster module entry point"
```

---

## Task 22: Delete old cluster.ts file

**Files:**
- Delete: `web/src/api/modules/cluster.ts`

- [ ] **Step 1: Delete old file**

```bash
rm web/src/api/modules/cluster.ts
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

Expected: Errors in files importing from old path

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/cluster.ts
git commit -m "refactor(web): delete old cluster.ts after migration"
```

---

## Task 23: Update api/index.ts import path

**Files:**
- Modify: `web/src/api/index.ts`

- [ ] **Step 1: Read current api/index.ts**

```bash
cat web/src/api/index.ts
```

- [ ] **Step 2: Update import path**

Replace the cluster import:

```typescript
// web/src/api/index.ts

// Change this line:
// export * from './modules/cluster';
// To:
export * from './modules/cluster';
```

Note: The import should work since `cluster/index.ts` exports everything.

- [ ] **Step 3: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 4: Commit**

```bash
git add web/src/api/index.ts
git commit -m "refactor(web): update api index for cluster module"
```

---

## Task 24: Update useClusterDetailPageOperations.tsx imports

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/hooks/useClusterDetailPageOperations.tsx`

- [ ] **Step 1: Read current imports**

```bash
head -35 web/src/pages/Deployment/Infrastructure/hooks/useClusterDetailPageOperations.tsx
```

- [ ] **Step 2: Update import statements**

Replace:

```typescript
// OLD (lines 15-30):
import type {
  ClusterNode,
  DeploymentInfo,
  StatefulSetInfo,
  PodInfo,
  ServiceInfo,
  IngressInfo,
  ConfigMapInfo,
  SecretInfo,
  ClusterOperationApproval,
  ClusterOperationResponse,
  ClusterOperationState,
  ClusterServiceMutationPayload,
  ClusterIngressMutationPayload,
} from '../../../../api/modules/cluster';
```

With:

```typescript
// NEW:
import type {
  ClusterNode,
  DeploymentInfo,
  StatefulSetInfo,
  PodInfo,
  ServiceInfo,
  IngressInfo,
  ConfigMapInfo,
  SecretInfo,
  ClusterOperationApproval,
  ClusterOperationResponse,
  ClusterOperationState,
  ClusterServiceMutationPayload,
  ClusterIngressMutationPayload,
} from '../../../../api/modules/cluster/types';
```

- [ ] **Step 3: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/hooks/useClusterDetailPageOperations.tsx
git commit -m "refactor(web): update useClusterDetailPageOperations imports"
```

---

## Task 25: Update ClusterBootstrapWizard.tsx imports

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx`

- [ ] **Step 1: Read current imports**

```bash
head -30 web/src/pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx
```

- [ ] **Step 2: Update import statements**

Replace any imports from `../../api/modules/cluster` to use the new structure.

- [ ] **Step 3: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx
git commit -m "refactor(web): update ClusterBootstrapWizard imports"
```

---

## Task 26: Update useClusterResources.ts imports

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/hooks/useClusterResources.ts`

- [ ] **Step 1: Read current imports**

```bash
head -30 web/src/pages/Deployment/Infrastructure/hooks/useClusterResources.ts
```

- [ ] **Step 2: Update import statements**

- [ ] **Step 3: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/hooks/useClusterResources.ts
git commit -m "refactor(web): update useClusterResources imports"
```

---

## Task 27: Update K8s components imports

**Files:**
- Modify: `web/src/components/K8s/RolloutPanel.tsx`
- Modify: `web/src/components/K8s/HPAEditor.tsx`
- Modify: `web/src/components/K8s/QuotaEditor.tsx`
- Modify: `web/src/components/K8s/NamespacePolicyPanel.tsx`

- [ ] **Step 1: Update all K8s component imports**

For each file, update import paths from `../../../api/modules/cluster` to use new structure.

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/components/K8s/
git commit -m "refactor(web): update K8s component imports"
```

---

## Task 28: Update ServiceDetailPage.tsx imports

**Files:**
- Modify: `web/src/pages/Services/ServiceDetailPage.tsx`

- [ ] **Step 1: Update imports**

- [ ] **Step 2: Run TypeScript check**

```bash
cd web && npm run typecheck
```

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/Services/ServiceDetailPage.tsx
git commit -m "refactor(web): update ServiceDetailPage imports"
```

---

## Task 29: Find and update remaining imports

**Files:**
- Multiple files with imports from cluster.ts

- [ ] **Step 1: Search for remaining imports**

```bash
grep -r "from.*api/modules/cluster'" web/src --include="*.ts" --include="*.tsx" | grep -v "cluster/" | grep -v "cluster.types" | grep -v "__tests__"
```

- [ ] **Step 2: Update each file found**

For each file:
1. Read current imports
2. Update to use new import paths
3. Verify TypeScript

- [ ] **Step 3: Run full TypeScript check**

```bash
cd web && npm run typecheck
```

Expected: All errors resolved

- [ ] **Step 4: Commit**

```bash
git add web/src/
git commit -m "refactor(web): update all remaining cluster imports"
```

---

## Task 30: Run tests to verify no regressions

**Files:**
- Test: All test files

- [ ] **Step 1: Run frontend tests**

```bash
cd web && npm run test
```

- [ ] **Step 2: Fix any test failures**

If tests fail, investigate and fix import issues or mock issues.

- [ ] **Step 3: Run lint**

```bash
cd web && npm run lint
```

- [ ] **Step 4: Commit any fixes**

```bash
git add web/src/
git commit -m "fix(web): resolve test failures after cluster split"
```

---

## Task 31: Final verification and cleanup

- [ ] **Step 1: Verify file sizes**

```bash
wc -l web/src/api/modules/cluster/types/*.ts
wc -l web/src/api/modules/cluster/operations/*.ts
```

Expected: Each file < 200 lines

- [ ] **Step 2: Verify no normalizers remain**

```bash
grep -r "normalizeCluster" web/src --include="*.ts" --include="*.tsx"
grep -r "coerceNumber" web/src --include="*.ts" --include="*.tsx"
```

Expected: No matches

- [ ] **Step 3: Final TypeScript check**

```bash
cd web && npm run typecheck
```

Expected: PASS

- [ ] **Step 4: Final commit**

```bash
git add web/src/
git commit -m "refactor(web): complete cluster.ts split - types and operations separated, normalizers removed"
```

---

## Success Criteria

1. ✅ Frontend cluster.ts split into 8 type files + 9 API files
2. ✅ Total entry file < 50 lines
3. ✅ No normalizer functions in frontend
4. ✅ TypeScript compilation passes
5. ✅ All tests pass
6. ✅ Import paths updated correctly