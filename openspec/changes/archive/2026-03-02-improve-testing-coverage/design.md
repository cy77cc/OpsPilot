# Design: Improve Testing Coverage

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           测试架构                                           │
│                                                                              │
│                           ┌─────────┐                                       │
│                          │   E2E   │  Playwright                            │
│                         └───────────┘                                       │
│                      ┌─────────────────┐                                    │
│                     │   集成测试       │  httptest + SQLite                  │
│                    └───────────────────┘                                    │
│                 ┌───────────────────────────┐                               │
│                │        单元测试            │  go test + vitest              │
│               └───────────────────────────────┘                             │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. 后端测试工具包 (internal/testutil)

```
internal/testutil/
├── integration.go      # 集成测试套件
├── mock_ssh.go         # SSH 客户端 Mock
├── mock_k8s.go         # Kubernetes 客户端 Mock
├── mock_ai.go          # AI 客户端 Mock
├── fixtures.go         # 测试数据工厂
└── assertions.go       # 自定义断言
```

**IntegrationSuite 结构**：

```go
type IntegrationSuite struct {
    DB      *gorm.DB              // SQLite 内存数据库
    SvcCtx  *svc.ServiceContext   // 服务上下文
    MockSSH *MockSSHClient        // SSH Mock
    MockK8s *MockK8sClient        // K8s Mock
    HTTP    *httptest.Server      // 测试服务器
}

func NewIntegrationSuite(t *testing.T) *IntegrationSuite
func (s *IntegrationSuite) SeedUser(overrides ...func(*model.User)) *model.User
func (s *IntegrationSuite) SeedNode(overrides ...func(*model.Node)) *model.Node
func (s *IntegrationSuite) SeedCluster(overrides ...func(*model.Cluster)) *model.Cluster
```

### 2. Mock 接口设计

**SSH Mock**：

```go
type MockSSHClient struct {
    ExecFunc    func(cmd string) (stdout, stderr string, err error)
    UploadFunc  func(remotePath string, content []byte) error
    calls       []string
}

func (m *MockSSHClient) Exec(cmd string) (string, string, error)
func (m *MockSSHClient) AssertCalled(t *testing.T, expectedCmd string)
```

**K8s Mock**：

```go
type MockK8sClient struct {
    clientset *fake.Clientset
}

func (m *MockK8sClient) CreateNamespace(name string) error
func (m *MockK8sClient) ApplyManifest(manifest string) error
func (m *MockK8sClient) GetPods(namespace string) ([]corev1.Pod, error)
```

### 3. 前端测试工具

```
web/src/test/
├── setupTests.ts       # 已有，需要扩展
├── mocks/
│   ├── api.ts          # API Mock 工厂
│   ├── server.ts       # MSW Server (可选)
│   └── handlers/       # MSW Handlers
├── factories/          # 测试数据工厂
│   ├── deployment.ts
│   ├── cicd.ts
│   └── user.ts
└── utils/
    ├── render.tsx      # 自定义 render (带 Provider)
    └── assertions.ts   # 自定义断言
```

### 4. E2E 测试结构

```
e2e/
├── playwright.config.ts
├── tests/
│   ├── auth.spec.ts
│   ├── deployment.spec.ts
│   ├── cicd.spec.ts
│   └── ai-chat.spec.ts
├── fixtures/
│   └── test-data.ts
└── support/
    ├── login.ts
    └── api-helpers.ts
```

## Test Cases by Module

### 后端: CICD 模块

| 测试用例 | 描述 | 优先级 |
|---------|------|--------|
| TestCreatePipeline | 创建流水线 | P0 |
| TestExecutePipeline | 执行流水线完整流程 | P0 |
| TestPipelineStepTransition | 步骤状态转换 | P0 |
| TestPipelineConditionalStep | 条件步骤跳过 | P1 |
| TestPipelineManualApproval | 手动审批步骤 | P1 |
| TestVariableSubstitution | 变量替换 | P1 |

### 后端: RBAC 模块

| 测试用例 | 描述 | 优先级 |
|---------|------|--------|
| TestAssignRole | 用户角色分配 | P0 |
| TestPermissionInheritance | 角色权限继承 | P0 |
| TestResourcePermissionCheck | 资源级权限校验 | P0 |
| TestPermissionCache | 权限缓存更新 | P1 |
| TestCasbinPolicyLoad | 策略加载 | P1 |
| TestPermissionSync | 权限同步 | P1 |

### 后端: Monitoring 模块

| 测试用例 | 描述 | 优先级 |
|---------|------|--------|
| TestAlertRuleEvaluation | 告警规则评估 | P0 |
| TestAlertAggregation | 告警聚合 | P1 |
| TestMetricCollection | 指标采集 | P1 |

### 前端: API 客户端

| 测试用例 | 描述 | 优先级 |
|---------|------|--------|
| 请求携带 Token | 请求头包含 Authorization | P0 |
| 请求携带 Project-ID | 请求头包含 X-Project-ID | P0 |
| 401 触发 Token 刷新 | code 4005/4006 触发刷新 | P0 |
| Token 刷新后重试 | 刷新成功后重新发起请求 | P0 |
| Token 刷新失败清理 | 清除本地存储 | P0 |
| 并发请求单次刷新 | 多请求只刷新一次 | P1 |
| 业务错误抛出异常 | code !== 1000 时抛错 | P0 |
| 网络错误友好消息 | 网络异常返回友好提示 | P1 |

### E2E: 关键旅程

| 测试场景 | 描述 | 优先级 |
|---------|------|--------|
| 用户登录流程 | 登录成功/失败场景 | P0 |
| 部署发布流程 | 创建目标→预览→发布 | P0 |
| AI 对话流程 | 发送消息→接收响应 | P1 |

## Coverage Targets

```
┌────────────────────┬─────────────┬─────────────┐
│ 模块               │ 当前        │ 目标        │
├────────────────────┼─────────────┼─────────────┤
│ 后端 deployment    │ ~30%        │ 70%         │
│ 后端 cicd          │ 0%          │ 60%         │
│ 后端 rbac          │ ~5%         │ 60%         │
│ 后端 monitoring    │ 0%          │ 50%         │
│ 后端 utils         │ ~30%        │ 85%         │
│ 前端 api           │ 0%          │ 80%         │
│ 前端 pages         │ ~10%        │ 50%         │
└────────────────────┴─────────────┴─────────────┘
```

## Dependencies

- Go 1.25+ (已有)
- Vitest 4.0+ (已有)
- Playwright (新增)
- k8s.io/client-go (已有)
- fake client-go (新增)
