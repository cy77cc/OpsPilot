# OpsPilot 大文件拆分与架构重构设计文档

**日期**: 2026-04-23
**范围**: 前端 cluster.ts (1662行) + 后端 bootstrap.go (1335行)
**执行策略**: 两端并行

---

## 1. 背景

根据 Code Review 报告，项目中存在多处超大文件：

| 文件 | 行数 | 问题 |
|------|------|------|
| 前端 `cluster.ts` | 1662 | 类型定义、normalization 函数、API 函数混杂 |
| 后端 `bootstrap.go` | 1335 | Handler 承担过多职责，SSH 执行、状态管理混在一起 |

这些大文件导致：
- 维护困难，查找特定功能需要大量滚动
- 职责不清，难以定位问题
- 可测试性差
- 新开发者难以理解代码组织

---

## 2. 目标

1. **前端 cluster.ts 拆分**：按功能域拆分为独立模块
2. **后端 bootstrap.go 拆分**：引入 Logic 层，职责分离
3. **移除前端 normalizers**：后端统一响应格式，前端删除约 550 行兼容代码
4. **引入 asynq 任务队列**：修复并发安全问题，任务持久化、可重试、可限流

---

## 3. 前端 cluster.ts 拆分设计

### 3.1 目标目录结构

```
web/src/api/modules/cluster/
├── types/
│   ├── bootstrap.types.ts    # Bootstrap 相关类型 (约 80 行)
│   ├── node.types.ts         # 节点操作类型 (约 60 行)
│   ├── workload.types.ts     # Deployment/StatefulSet/DaemonSet/Job 类型 (约 50 行)
│   ├── network.types.ts      # Service/Ingress 类型 (约 40 行)
│   ├── policy.types.ts       # 网络策略类型 (约 80 行)
│   ├── operation.types.ts    # 操作审批类型 (约 70 行)
│   ├── resource.types.ts     # Namespace/Pod/PVC/PV/ConfigMap/Secret 类型 (约 60 行)
│   └── index.ts              # 统一导出 (约 20 行)
├── operations/
│   ├── bootstrap.api.ts      # Bootstrap CRUD API (约 100 行)
│   ├── node.api.ts           # 节点 cordon/uncordon/drain/taint/label API (约 150 行)
│   ├── workload.api.ts       # Workload restart/scale/delete API (约 120 行)
│   ├── network.api.ts        # Service/Ingress CRUD API (约 100 行)
│   ├── policy.api.ts         # 网络策略 simulate/release API (约 150 行)
│   ├── operation.api.ts      # 操作历史查询 API (约 80 行)
│   ├── resource.api.ts       # 资源查询 API (约 150 行)
│   └── index.ts              # 统一导出 (约 20 行)
└── index.ts                  # 模块入口导出 clusterApi (约 30 行)
```

### 3.2 类型拆分详情

#### bootstrap.types.ts

```typescript
// 从 cluster.ts 第 64-185 行提取

export interface BootstrapPreviewReq {
  name: string;
  profile_id?: number;
  control_plane_host_id: number;
  worker_host_ids?: number[];
  k8s_version?: string;
  // ...
}

export interface BootstrapPreviewResp {
  name: string;
  control_plane_host_id: number;
  worker_host_ids: number[];
  k8s_version: string;
  steps: string[];
  expected_endpoint: string;
  warnings?: string[];
  validation_issues?: BootstrapValidationIssue[];
  // ...
}

export interface BootstrapTask {
  id: string;
  name: string;
  cluster_id?: number;
  k8s_version: string;
  status: string;
  steps: BootstrapStepStatus[];
  current_step: number;
  error_message?: string;
  // ...
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

export interface BootstrapProfile {
  id: number;
  name: string;
  description?: string;
  version_channel: string;
  k8s_version: string;
  repo_mode: 'online' | 'mirror';
  // ...
}
```

#### node.types.ts

```typescript
// 从 cluster.ts 第 24-62 行提取

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
  // ...
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
```

#### operation.types.ts

```typescript
// 从 cluster.ts 第 212-364 行提取

export type ClusterOperationState = 'completed' | 'approval_required' | 'rejected' | 'failed';

export interface ClusterOperationApproval {
  required: boolean;
  ticket?: string;
  cluster_id?: number;
  namespace?: string;
  action?: string;
  resource?: string;
  // ...
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
```

#### policy.types.ts

```typescript
// 从 cluster.ts 第 600-700 行提取

export interface ClusterPolicySimulationResult {
  passed: boolean;
  blocking_issues: ClusterPolicyIssue[];
  warnings: ClusterPolicyWarning[];
  impact_summary?: ClusterPolicyImpactSummary;
  risk_score?: number;
  risk_level?: string;
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
```

### 3.3 API 拆分详情

#### bootstrap.api.ts

```typescript
// 从 cluster.ts 第 1543-1578 行提取

import apiService from '../../api';
import type { ApiResponse } from '../../api';
import type {
  BootstrapPreviewReq,
  BootstrapPreviewResp,
  BootstrapTask,
  BootstrapProfile,
  BootstrapVersionItem,
} from '../types/bootstrap.types';

export const bootstrapApi = {
  getVersions(): Promise<ApiResponse<{ default_channel: string; list: BootstrapVersionItem[] }>> {
    return apiService.get('/clusters/bootstrap/versions');
  },

  getProfiles(): Promise<ApiResponse<{ list: BootstrapProfile[]; total: number }>> {
    return apiService.get('/clusters/bootstrap/profiles');
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

#### node.api.ts

```typescript
// 从 cluster.ts 第 1294-1341 行提取

import apiService from '../../api';
import type { ApiResponse, PaginatedResponse } from '../../api';
import type {
  ClusterNode,
  ClusterNodeDrainPayload,
  ClusterNodeTaintPayload,
  ClusterNodeLabelPayload,
  ClusterOperationResponse,
} from '../types';

export const nodeApi = {
  getNodes(clusterId: number): Promise<ApiResponse<PaginatedResponse<ClusterNode>>> {
    return apiService.get(`/clusters/${clusterId}/nodes`);
  },

  syncNodes(clusterId: number): Promise<ApiResponse<PaginatedResponse<ClusterNode>>> {
    return apiService.post(`/clusters/${clusterId}/nodes/sync`);
  },

  getNodeDetail(clusterId: number, nodeName: string): Promise<ApiResponse<ClusterNode>> {
    return apiService.get(`/clusters/${clusterId}/nodes/${encodeURIComponent(nodeName)}`);
  },

  cordonNode(clusterId: number, nodeName: string, payload?: ClusterNodeApprovalPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${clusterId}/nodes/${encodeURIComponent(nodeName)}/cordon`, payload || {}));
  },

  uncordonNode(clusterId: number, nodeName: string, payload?: ClusterNodeApprovalPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${clusterId}/nodes/${encodeURIComponent(nodeName)}/uncordon`, payload || {}));
  },

  drainNode(clusterId: number, nodeName: string, payload?: ClusterNodeDrainPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${clusterId}/nodes/${encodeURIComponent(nodeName)}/drain`, payload || {}));
  },

  // ...
};
```

### 3.4 删除内容

**删除约 550 行 normalization 函数**：

从 cluster.ts 删除以下函数（移至后端处理）：
- `isPlainObject` (第 724-727 行)
- `coerceNumber` (第 728-739 行)
- `coerceStringArray` (第 741-759 行)
- `coerceBoolean` (第 761-771 行)
- `coerceObject` (第 773-779 行)
- `coerceString` (第 780-783 行)
- `normalizePolicyWarning` (第 784-803 行)
- `normalizePolicyWarnings` (第 805-814 行)
- `normalizePolicyIssue` (第 816-838 行)
- `normalizePolicyIssues` (第 840-850 行)
- `normalizePolicyImpactSummary` (第 851-861 行)
- `normalizeClusterPolicySimulationStatus` (第 867-904 行)
- `normalizeClusterPolicySimulationResult` (第 906-927 行)
- `normalizeClusterPolicyRelease` (第 929-1019 行)
- `normalizeClusterCNIInfo` (第 1021-1033 行)
- `normalizeApprovalPayload` (第 1035-1063 行)
- `normalizeOperationState` (第 1065-1127 行)
- `normalizeOperationCode` (第 1129-1152 行)
- `normalizeAuditID` (第 1154-1160 行)
- `normalizeOperationMessage` (第 1162-1181 行)
- `normalizeOperationApproval` (第 1183-1201 行)
- `normalizeClusterOperationResponse` (第 1203-1250 行)
- `wrapOperationResponse` (第 1252-1265 行)
- 以及所有相关的常量定义（第 701-723 行）

### 3.5 导入路径变更

所有使用 cluster.ts 的文件需要更新导入路径：

```typescript
// 之前
import type { Cluster, ClusterNode, ClusterOperationResponse } from '../../api/modules/cluster';
import { clusterApi } from '../../api/modules/cluster';

// 之后
import type { Cluster } from '../../api/modules/cluster/types';
import type { ClusterNode } from '../../api/modules/cluster/types/node.types';
import type { ClusterOperationResponse } from '../../api/modules/cluster/types/operation.types';
import { clusterApi } from '../../api/modules/cluster';
```

---

## 4. 后端 bootstrap.go 拆分设计

### 4.1 目标目录结构

```
internal/modules/cluster/
├── handler/
│   └── bootstrap_handler.go   # HTTP Handler 方法（约 150 行）
├── logic/bootstrap/
│   ├── types.go               # 类型定义（约 120 行）
│   ├── steps.go               # 步骤构建（约 100 行）
│   ├── executor.go            # 任务执行（约 300 行）
│   ├── ssh.go                 # SSH 操作封装（约 200 行）
│   ├── validator.go           # 预检查验证（约 150 行）
│   ├── normalizer.go          # 响应数据标准化（约 100 行）
│   └── queue.go               # asynq 任务队列（约 150 行）
└── model/
│   └── bootstrap_task.go      # BootstrapTask 模型（已有）
```

### 4.2 职责划分

| 文件 | 职责 | 估计行数 | 原位置 |
|------|------|----------|--------|
| `handler/bootstrap_handler.go` | HTTP Handler 方法，参数验证，调用 Logic | ~150 | bootstrap.go 第 136-450 行 |
| `logic/bootstrap/types.go` | 所有类型定义 | ~120 | bootstrap.go 第 39-116 行 |
| `logic/bootstrap/steps.go` | buildBootstrapSteps，版本目录加载 | ~100 | bootstrap.go 第 117-150 行 |
| `logic/bootstrap/executor.go` | executeBootstrap 任务执行，状态更新 | ~300 | bootstrap.go 第 492-650 行 |
| `logic/bootstrap/ssh.go` | SSH 命令执行封装，超时处理 | ~200 | bootstrap.go 第 650-850 行 |
| `logic/bootstrap/validator.go` | 预检查，版本验证 | ~150 | bootstrap.go 第 183-350 行 |
| `logic/bootstrap/normalizer.go` | 响应数据标准化，替代前端 normalizers | ~100 | 新增 |
| `logic/bootstrap/queue.go` | asynq 任务队列集成 | ~150 | 新增 |

### 4.3 各文件详细设计

#### handler/bootstrap_handler.go

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/cy77cc/OpsPilot/internal/core/httpx"
    "github.com/cy77cc/OpsPilot/internal/modules/cluster/logic/bootstrap"
)

// BootstrapHandler 处理 Bootstrap 相关 HTTP 请求
type BootstrapHandler struct {
    logic *bootstrap.Logic
    queue *bootstrap.Queue
}

func NewBootstrapHandler(logic *bootstrap.Logic, queue *bootstrap.Queue) *BootstrapHandler {
    return &BootstrapHandler{logic: logic, queue: queue}
}

// GetBootstrapVersions 获取可用版本列表
// GET /clusters/bootstrap/versions
func (h *BootstrapHandler) GetBootstrapVersions(c *gin.Context) {
    channel, items := h.logic.LoadVersionCatalog(c.Request.Context())
    httpx.OK(c, gin.H{
        "default_channel": channel,
        "list":            items,
    })
}

// PreviewBootstrap 预览 Bootstrap 配置
// POST /clusters/bootstrap/preview
func (h *BootstrapHandler) PreviewBootstrap(c *gin.Context) {
    var req bootstrap.PreviewReq
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BadRequest(c, err)
        return
    }
    
    preview, issues := h.logic.Preview(c.Request.Context(), &req)
    if len(issues) > 0 {
        httpx.OK(c, gin.H{
            "code":              2000,
            "msg":               "bootstrap profile validation failed",
            "validation_issues": issues,
            "preview":           preview,
        })
        return
    }
    
    httpx.OK(c, preview)
}

// ApplyBootstrap 执行 Bootstrap
// POST /clusters/bootstrap/apply
func (h *BootstrapHandler) ApplyBootstrap(c *gin.Context) {
    var req bootstrap.PreviewReq
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BadRequest(c, err)
        return
    }
    
    // 创建任务并入队
    taskID, err := h.queue.Enqueue(c.Request.Context(), &req)
    if err != nil {
        httpx.ServerErr(c, err)
        return
    }
    
    httpx.OK(c, gin.H{
        "task_id": taskID,
        "status":  "pending",
    })
}

// GetBootstrapTask 获取任务状态
// GET /clusters/bootstrap/:taskId
func (h *BootstrapHandler) GetBootstrapTask(c *gin.Context) {
    taskID := c.Param("taskId")
    task, err := h.logic.GetTask(c.Request.Context(), taskID)
    if err != nil {
        httpx.ServerErr(c, err)
        return
    }
    
    // 使用 normalizer 标准化响应
    response := bootstrap.NormalizeTaskResponse(task)
    httpx.OK(c, response)
}
```

#### logic/bootstrap/types.go

```go
package bootstrap

import "time"

// PreviewReq Bootstrap 预览请求
type PreviewReq struct {
    Name                 string                 `json:"name" binding:"required"`
    ProfileID            *uint                  `json:"profile_id,omitempty"`
    ControlPlaneID       uint                   `json:"control_plane_host_id" binding:"required"`
    WorkerIDs            []uint                 `json:"worker_host_ids"`
    K8sVersion           string                 `json:"k8s_version"`
    VersionChannel       string                 `json:"version_channel"`
    CNI                  string                 `json:"cni"`
    PodCIDR              string                 `json:"pod_cidr"`
    ServiceCIDR          string                 `json:"service_cidr"`
    RepoMode             string                 `json:"repo_mode"`
    RepoURL              string                 `json:"repo_url"`
    ImageRepository      string                 `json:"image_repository"`
    EndpointMode         string                 `json:"endpoint_mode"`
    ControlPlaneEndpoint string                 `json:"control_plane_endpoint"`
    VIPProvider          string                 `json:"vip_provider"`
    EtcdMode             string                 `json:"etcd_mode"`
    ExternalEtcd         map[string]any         `json:"external_etcd"`
}

// PreviewResp Bootstrap 预览响应
type PreviewResp struct {
    Name                 string             `json:"name"`
    ControlPlaneID       uint               `json:"control_plane_host_id"`
    WorkerIDs            []uint             `json:"worker_host_ids"`
    K8sVersion           string             `json:"k8s_version"`
    VersionChannel       string             `json:"version_channel"`
    CNI                  string             `json:"cni"`
    PodCIDR              string             `json:"pod_cidr"`
    ServiceCIDR          string             `json:"service_cidr"`
    RepoMode             string             `json:"repo_mode"`
    RepoURL              string             `json:"repo_url"`
    ImageRepository      string             `json:"image_repository"`
    EndpointMode         string             `json:"endpoint_mode"`
    ControlPlaneEndpoint string             `json:"control_plane_endpoint"`
    VIPProvider          string             `json:"vip_provider"`
    EtcdMode             string             `json:"etcd_mode"`
    Steps                []string           `json:"steps"`
    ExpectedEndpoint     string             `json:"expected_endpoint"`
    Warnings             []string           `json:"warnings,omitempty"`
    ValidationIssues     []ValidationIssue  `json:"validation_issues,omitempty"`
    Diagnostics          map[string]any     `json:"diagnostics,omitempty"`
}

// ValidationIssue 验证问题
type ValidationIssue struct {
    Field       string `json:"field"`
    Code        string `json:"code"`
    Domain      string `json:"domain"`
    Message     string `json:"message"`
    Remediation string `json:"remediation,omitempty"`
}

// Step Bootstrap 步骤定义
type Step struct {
    Name      string
    Hosts     []string // "control-plane", "workers", "all"
    Script    string
    Timeout   time.Duration
    Rollback  string
    OnFailure string // "abort", "continue"
    EnvVars   map[string]string
}

// StepStatus 步骤执行状态
type StepStatus struct {
    Name       string     `json:"name"`
    Status     string     `json:"status"` // pending, running, completed, failed
    Message    string     `json:"message,omitempty"`
    StartedAt  *time.Time `json:"started_at,omitempty"`
    FinishedAt *time.Time `json:"finished_at,omitempty"`
    HostID     *uint      `json:"host_id,omitempty"`
    Output     string     `json:"output,omitempty"`
}

// Task Bootstrap 任务
type Task struct {
    ID              string       `json:"id"`
    Name            string       `json:"name"`
    ClusterID       *uint        `json:"cluster_id,omitempty"`
    K8sVersion      string       `json:"k8s_version"`
    Status          string       `json:"status"` // pending, running, completed, failed
    Steps           []StepStatus `json:"steps"`
    CurrentStep     int          `json:"current_step"`
    ErrorMessage    string       `json:"error_message,omitempty"`
    ResolvedConfig  string       `json:"resolved_config_json,omitempty"`
    Diagnostics     string       `json:"diagnostics_json,omitempty"`
    CreatedAt       time.Time    `json:"created_at"`
    UpdatedAt       time.Time    `json:"updated_at"`
}

// TaskPayload asynq 任务 payload
type TaskPayload struct {
    TaskID    string    `json:"task_id"`
    ClusterID uint      `json:"cluster_id,omitempty"`
    Config    PreviewReq `json:"config"`
}
```

#### logic/bootstrap/queue.go

```go
package bootstrap

import (
    "context"
    "encoding/json"
    "time"

    "github.com/hibiken/asynq"
)

const (
    TaskTypeBootstrap    = "bootstrap:execute"
    DefaultMaxRetry      = 3
    DefaultTimeout       = 30 * time.Minute
    DefaultMaxConcurrency = 10
)

// Queue asynq 任务队列
type Queue struct {
    client   *asynq.Client
    server   *asynq.Server
    executor *Executor
}

// QueueConfig 队列配置
type QueueConfig struct {
    RedisAddr      string
    MaxConcurrency int
    MaxRetry       int
    Timeout        time.Duration
}

// NewQueue 创建任务队列
func NewQueue(cfg QueueConfig, executor *Executor) *Queue {
    client := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
    
    server := asynq.NewServer(
        asynq.RedisClientOpt{Addr: cfg.RedisAddr},
        asynq.Config{
            Concurrency: cfg.MaxConcurrency,
            Queues: map[string]int{
                "critical": 10,
                "default":  5,
                "low":      1,
            },
            RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
                return time.Duration(n) * time.Minute
            },
        },
    )
    
    return &Queue{
        client:   client,
        server:   server,
        executor: executor,
    }
}

// Enqueue 入队任务
func (q *Queue) Enqueue(ctx context.Context, req *PreviewReq) (string, error) {
    taskID := generateTaskID()
    
    payload := TaskPayload{
        TaskID: taskID,
        Config: *req,
    }
    
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return "", err
    }
    
    task := asynq.NewTask(TaskTypeBootstrap, payloadBytes)
    
    info, err := q.client.Enqueue(task,
        asynq.MaxRetry(DefaultMaxRetry),
        asynq.Timeout(DefaultTimeout),
        asynq.TaskID(taskID),
    )
    if err != nil {
        return "", err
    }
    
    return info.ID, nil
}

// RegisterWorker 注册任务处理器
func (q *Queue) RegisterWorker() {
    mux := asynq.NewServeMux()
    
    mux.HandleFunc(TaskTypeBootstrap, func(ctx context.Context, t *asynq.Task) error {
        var payload TaskPayload
        if err := json.Unmarshal(t.Payload(), &payload); err != nil {
            return err
        }
        
        return q.executor.Execute(ctx, &payload)
    })
    
    q.server.Start(mux)
}

// Shutdown 关闭队列
func (q *Queue) Shutdown() {
    q.server.Shutdown()
    q.client.Close()
}

func generateTaskID() string {
    return fmt.Sprintf("bootstrap-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])
}
```

#### logic/bootstrap/executor.go

```go
package bootstrap

import (
    "context"
    "fmt"
    "time"

    "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
    "gorm.io/gorm"
)

// Executor Bootstrap 任务执行器
type Executor struct {
    db        *gorm.DB
    ssh       *SSHClient
    validator *Validator
}

// NewExecutor 创建执行器
func NewExecutor(db *gorm.DB, ssh *SSHClient) *Executor {
    return &Executor{
        db:        db,
        ssh:       ssh,
        validator: NewValidator(db),
    }
}

// Execute 执行 Bootstrap 任务
func (e *Executor) Execute(ctx context.Context, payload *TaskPayload) error {
    // 1. 创建任务记录
    task := &model.BootstrapTask{
        ID:         payload.TaskID,
        Name:       payload.Config.Name,
        K8sVersion: payload.Config.K8sVersion,
        Status:     "running",
        Steps:      buildStepStatuses(payload.Config.K8sVersion),
        CreatedAt:  time.Now(),
    }
    
    if err := e.db.Create(task).Error; err != nil {
        return fmt.Errorf("failed to create task: %w", err)
    }
    
    // 2. 获取主机信息
    hosts, err := e.validator.LoadHosts(ctx, payload.Config.ControlPlaneID, payload.Config.WorkerIDs)
    if err != nil {
        e.updateTaskStatus(task, "failed", err.Error())
        return err
    }
    
    // 3. 执行步骤
    steps := buildBootstrapSteps(payload.Config.K8sVersion)
    
    for i, step := range steps {
        if ctx.Err() != nil {
            e.updateTaskStatus(task, "aborted", "context cancelled")
            return ctx.Err()
        }
        
        e.updateStepStatus(task, i, "running", "")
        
        err := e.executeStep(ctx, step, hosts, &payload.Config)
        if err != nil {
            if step.OnFailure == "abort" {
                e.updateStepStatus(task, i, "failed", err.Error())
                e.updateTaskStatus(task, "failed", err.Error())
                return err
            }
            e.updateStepStatus(task, i, "failed", err.Error())
            continue
        }
        
        e.updateStepStatus(task, i, "completed", "")
    }
    
    // 4. 更新最终状态
    e.updateTaskStatus(task, "completed", "")
    
    return nil
}

// executeStep 执行单个步骤
func (e *Executor) executeStep(ctx context.Context, step Step, hosts HostsMap, config *PreviewReq) error {
    targetHosts := step.Hosts
    if len(targetHosts) == 0 {
        targetHosts = []string{"all"}
    }
    
    for _, target := range targetHosts {
        hostList := hosts.Get(target)
        for _, host := range hostList {
            timeoutCtx, cancel := context.WithTimeout(ctx, step.Timeout)
            defer cancel()
            
            output, err := e.ssh.ExecuteScript(timeoutCtx, host, step.Script, step.EnvVars, config)
            if err != nil {
                return fmt.Errorf("step %s failed on host %s: %w", step.Name, host.Name, err)
            }
            
            // 记录输出
            e.appendStepOutput(host.ID, output)
        }
    }
    
    return nil
}

// updateTaskStatus 更新任务状态
func (e *Executor) updateTaskStatus(task *model.BootstrapTask, status, message string) {
    task.Status = status
    task.ErrorMessage = message
    task.UpdatedAt = time.Now()
    e.db.Save(task)
}

// updateStepStatus 更新步骤状态
func (e *Executor) updateStepStatus(task *model.BootstrapTask, stepIndex int, status, message string) {
    if stepIndex < len(task.Steps) {
        task.Steps[stepIndex].Status = status
        task.Steps[stepIndex].Message = message
        if status == "running" {
            task.Steps[stepIndex].StartedAt = &time.Time{}
            *task.Steps[stepIndex].StartedAt = time.Now()
        }
        if status == "completed" || status == "failed" {
            task.Steps[stepIndex].FinishedAt = &time.Time{}
            *task.Steps[stepIndex].FinishedAt = time.Now()
        }
        task.CurrentStep = stepIndex
        task.UpdatedAt = time.Now()
        e.db.Save(task)
    }
}
```

### 4.4 normalizer.go 设计（替代前端 normalizers）

```go
package bootstrap

// TaskResponse 标准任务响应格式
type TaskResponse struct {
    ID           string       `json:"id"`
    Name         string       `json:"name"`
    ClusterID    *uint        `json:"cluster_id,omitempty"`
    K8sVersion   string       `json:"k8s_version"`
    Status       string       `json:"status"` // pending, running, completed, failed, aborted
    Steps        []StepStatus `json:"steps"`
    CurrentStep  int          `json:"current_step"`
    ErrorMessage string       `json:"error_message,omitempty"`
    Progress     int          `json:"progress"` // 0-100
    CreatedAt    string       `json:"created_at"`
    UpdatedAt    string       `json:"updated_at"`
}

// NormalizeTaskResponse 标准化任务响应
func NormalizeTaskResponse(task *model.BootstrapTask) TaskResponse {
    progress := calculateProgress(task)
    
    return TaskResponse{
        ID:           task.ID,
        Name:         task.Name,
        ClusterID:    task.ClusterID,
        K8sVersion:   task.K8sVersion,
        Status:       normalizeStatus(task.Status),
        Steps:        normalizeSteps(task.Steps),
        CurrentStep:  task.CurrentStep,
        ErrorMessage: task.ErrorMessage,
        Progress:     progress,
        CreatedAt:    task.CreatedAt.Format(time.RFC3339),
        UpdatedAt:    task.UpdatedAt.Format(time.RFC3339),
    }
}

func normalizeStatus(status string) string {
    // 确保状态为标准值
    validStatuses := map[string]string{
        "pending":   "pending",
        "running":   "running",
        "completed": "completed",
        "failed":    "failed",
        "aborted":   "aborted",
        "success":   "completed",
        "error":     "failed",
    }
    
    if normalized, ok := validStatuses[status]; ok {
        return normalized
    }
    return status
}

func normalizeSteps(steps []model.BootstrapStepStatus) []StepStatus {
    result := make([]StepStatus, len(steps))
    for i, s := range steps {
        result[i] = StepStatus{
            Name:       s.Name,
            Status:     normalizeStatus(s.Status),
            Message:    s.Message,
            StartedAt:  formatTime(s.StartedAt),
            FinishedAt: formatTime(s.FinishedAt),
            HostID:     s.HostID,
            Output:     s.Output,
        }
    }
    return result
}

func calculateProgress(task *model.BootstrapTask) int {
    if len(task.Steps) == 0 {
        return 0
    }
    
    completed := 0
    for _, s := range task.Steps {
        if s.Status == "completed" {
            completed++
        }
    }
    
    return (completed * 100) / len(task.Steps)
}

func formatTime(t *time.Time) string {
    if t == nil {
        return ""
    }
    return t.Format(time.RFC3339)
}
```

---

## 5. asynq 配置集成

### 5.1 配置扩展

```go
// internal/core/config/config.go 新增

type AsynqConfig struct {
    RedisAddr      string `yaml:"redis_addr" env:"ASYNQ_REDIS_ADDR"`
    MaxConcurrency int    `yaml:"max_concurrency" env:"ASYNQ_MAX_CONCURRENCY"`
    MaxRetry       int    `yaml:"max_retry" env:"ASYNQ_MAX_RETRY"`
    TimeoutSeconds int    `yaml:"timeout_seconds" env:"ASYNQ_TIMEOUT_SECONDS"`
}

// 默认值
func defaultAsynqConfig() AsynqConfig {
    return AsynqConfig{
        RedisAddr:      "localhost:6379",
        MaxConcurrency: 10,
        MaxRetry:       3,
        TimeoutSeconds: 1800, // 30 minutes
    }
}
```

### 5.2 ServerContext 扩展

```go
// internal/svc/app_context.go 新增

type ServiceContext struct {
    DB             *gorm.DB
    Rdb            redis.UniversalClient
    CacheFacade    *cache.Facade
    CasbinEnforcer *casbin.Enforcer
    Prometheus     prominfra.Client
    
    // 新增
    BootstrapQueue *bootstrap.Queue
}

// NewServiceContext 更新
func NewServiceContext(cfg *config.Config) (*ServiceContext, error) {
    // ... 现有初始化
    
    // 初始化 Bootstrap Queue
    bootstrapQueue := bootstrap.NewQueue(bootstrap.QueueConfig{
        RedisAddr:      cfg.Asynq.RedisAddr,
        MaxConcurrency: cfg.Asynq.MaxConcurrency,
        MaxRetry:       cfg.Asynq.MaxRetry,
        Timeout:        time.Duration(cfg.Asynq.TimeoutSeconds) * time.Second,
    }, bootstrap.NewExecutor(db, sshClient))
    
    bootstrapQueue.RegisterWorker()
    
    return &ServiceContext{
        // ...
        BootstrapQueue: bootstrapQueue,
    }, nil
}
```

---

## 6. 前端导入路径变更清单

需要更新导入路径的文件列表：

| 文件 | 当前导入 | 新导入 |
|------|----------|--------|
| `pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx` | `from '../../api/modules/cluster'` | `from '../../api/modules/cluster'` (clusterApi 保持不变) |
| `pages/Deployment/Infrastructure/hooks/useClusterDetailPageOperations.tsx` | `from '../../../../api/modules/cluster'` | 类型导入改为 `from '../../../../api/modules/cluster/types'` |
| `pages/Deployment/Infrastructure/hooks/useClusterResources.ts` | 同上 | 同上 |
| `components/K8s/RolloutPanel.tsx` | 同上 | 同上 |
| `components/K8s/HPAEditor.tsx` | 同上 | 同上 |
| `components/K8s/QuotaEditor.tsx` | 同上 | 同上 |

---

## 7. 执行顺序

### Phase 1: 前端拆分 (可并行)

1. 创建 `web/src/api/modules/cluster/types/` 目录和类型文件
2. 创建 `web/src/api/modules/cluster/operations/` 目录和 API 文件
3. 更新 `web/src/api/modules/cluster/index.ts` 导出
4. 删除 cluster.ts 中的 normalizers 函数
5. 更新所有导入路径
6. 运行 TypeScript 类型检查

### Phase 2: 后端拆分 (可并行)

1. 创建 `internal/modules/cluster/logic/bootstrap/` 目录
2. 创建 types.go、steps.go、executor.go、ssh.go、validator.go、normalizer.go
3. 创建 queue.go 并集成 asynq
4. 更新 handler/bootstrap_handler.go（拆分后的精简版）
5. 更新 ServiceContext 初始化
6. 更新 config.go 添加 AsynqConfig
7. 运行 Go 编译检查

### Phase 3: 集成测试

1. 前端启动测试，验证 API 调用
2. 后端启动测试，验证 Bootstrap 流程
3. 测试 asynq 任务队列功能

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 导入路径变更导致编译错误 | 前端无法编译 | 使用 TypeScript 自动导入重构工具 |
| asynq 集成导致启动失败 | 后端无法启动 | 先在本地 Redis 测试，逐步部署 |
| normalizers 移除导致前端数据异常 | 运行时错误 | 后端 normalizer.go 先行测试，确保格式一致 |
| 任务队列迁移导致任务丢失 | 生产故障 | 提供平滑迁移脚本，数据库记录作为备份 |

---

## 9. 成功标准

1. 前端 cluster.ts 拆分为 8 个类型文件 + 8 个 API 文件，总入口文件 < 50 行
2. 后端 bootstrap.go 拆分为 8 个文件，Handler 文件 < 200 行
3. 前端 normalizers 函数完全删除
4. asynq 任务队列正常工作，支持重试和限流
5. TypeScript 和 Go 编译无错误
6. Bootstrap 流程测试通过