# Design: 系统测试体系建设

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           测试架构                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      E2E Tests (Playwright)                      │   │
│  │  e2e/tests/                                                       │   │
│  │  ├── auth.spec.ts          # 认证流程                             │   │
│  │  ├── deployment.spec.ts    # 部署流程                             │   │
│  │  └── cluster.spec.ts       # 集群管理流程                          │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                    │
│  ┌─────────────────────────────────┴───────────────────────────────┐   │
│  │                   API Contract Tests                             │   │
│  │  internal/testutil/contract/                                     │   │
│  │  ├── auth_contract_test.go                                      │   │
│  │  ├── cluster_contract_test.go                                   │   │
│  │  └── deployment_contract_test.go                                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                    │
│  ┌─────────────────────────────────┴───────────────────────────────┐   │
│  │                   Integration Tests                              │   │
│  │  internal/service/*/                                            │   │
│  │  ├── IntegrationSuite (SQLite + Mocks)                          │   │
│  │  └── Test fixtures & assertions                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                    │
│  ┌─────────────────────────────────┴───────────────────────────────┐   │
│  │                      Unit Tests                                  │   │
│  │  internal/service/user/logic/auth_test.go                       │   │
│  │  internal/service/rbac/handler/*_test.go                        │   │
│  │  internal/service/cluster/*_test.go                             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      Test Infrastructure                         │   │
│  │  internal/testutil/                                              │   │
│  │  ├── integration.go    # IntegrationSuite                        │   │
│  │  ├── fixtures.go       # Test data factories                     │   │
│  │  ├── assertions.go     # Custom assertions                       │   │
│  │  ├── mock_k8s.go       # K8s client mock                         │   │
│  │  ├── mock_ssh.go       # SSH client mock                         │   │
│  │  └── mock_cache.go     # Cache mock (NEW)                        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. 用户认证测试 (user/auth)

#### 文件结构
```
internal/service/user/logic/
├── auth.go
└── auth_test.go    # 新增
```

#### 测试用例设计

```go
// auth_test.go
func TestLogin(t *testing.T) {
    tests := []struct {
        name      string
        setup     func(*IntegrationSuite)
        req       v1.LoginReq
        wantErr   bool
        errCode   int
    }{
        {
            name:    "success with valid credentials",
            setup:   seedValidUser,
            req:     v1.LoginReq{Username: "testuser", Password: "password123"},
            wantErr: false,
        },
        {
            name:    "fail with non-existent user",
            setup:   func(s *IntegrationSuite) {},
            req:     v1.LoginReq{Username: "nonexistent", Password: "password"},
            wantErr: true,
            errCode: xcode.UserNotExist,
        },
        {
            name:    "fail with wrong password",
            setup:   seedValidUser,
            req:     v1.LoginReq{Username: "testuser", Password: "wrongpassword"},
            wantErr: true,
            errCode: xcode.PasswordError,
        },
        {
            name:    "fail with empty username",
            req:     v1.LoginReq{Username: "", Password: "password"},
            wantErr: true,
        },
    }
    // ...
}

func TestRegister(t *testing.T) {
    // 注册成功、用户已存在、密码哈希、默认角色分配
}

func TestRefresh(t *testing.T) {
    // Token刷新成功、无效Token、过期Token
}

func TestLogout(t *testing.T) {
    // 登出成功、白名单清理
}
```

#### Mock 依赖

```go
// internal/testutil/mock_cache.go
type MockWhitelistDAO struct {
    whitelist map[string]time.Time
}

func (m *MockWhitelistDAO) AddToWhitelist(ctx context.Context, token string, exp time.Time) error
func (m *MockWhitelistDAO) IsWhitelisted(ctx context.Context, token string) (bool, error)
func (m *MockWhitelistDAO) DeleteToken(ctx context.Context, token string) error
```

### 2. RBAC 测试

#### 文件结构
```
internal/service/rbac/
├── handler/
│   └── *_test.go    # 扩展现有测试
└── logic/
    └── logic_test.go    # 新增
```

#### 测试场景

```go
// 权限组合测试
func TestPermissionCombination(t *testing.T) {
    // 多角色用户权限合并
    // 角色继承场景
    // 权限冲突处理
}

// 边界条件测试
func TestRBACBoundaryConditions(t *testing.T) {
    // 空权限用户
    // 无效角色
    // 循环角色继承
}
```

### 3. 集群管理测试

#### 文件结构
```
internal/service/cluster/
├── logic_bootstrap_test.go    # 已存在，扩展
├── logic_import_test.go       # 新增
├── logic_sync_test.go         # 新增
└── handler_test.go            # 新增
```

#### 测试场景

```go
// 集群导入测试
func TestClusterImport(t *testing.T) {
    // Kubeconfig 导入
    // Token 导入
    // 证书导入
    // 连接验证
    // 导入失败回滚
}

// 集群状态同步测试
func TestClusterSync(t *testing.T) {
    // 节点状态同步
    // 资源状态同步
    // 异常处理
}
```

### 4. E2E 测试改造

#### 认证支持

```typescript
// e2e/support/auth.ts
import { test as base } from '@playwright/test';

type TestFixtures = {
  authenticatedPage: Page;
};

export const test = base.extend<TestFixtures>({
  authenticatedPage: async ({ page }, use) => {
    // 自动登录
    await page.goto('/login');
    await page.fill('input[name="username"]', process.env.TEST_USER || 'admin');
    await page.fill('input[name="password"]', process.env.TEST_PASS || 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard**');
    await use(page);
  },
});
```

#### 测试环境配置

```yaml
# e2e/env.yaml
test_user: admin
test_pass: admin123
api_base: http://localhost:8080
```

## Test Data Management

### 数据工厂模式

```go
// internal/testutil/fixtures.go (扩展现有)

// UserFactory 用户数据工厂
type UserFactory struct {
    db *gorm.DB
}

func (f *UserFactory) Create(overrides ...func(*model.User)) *model.User {
    user := &model.User{
        Username:     "testuser_" + randomString(6),
        PasswordHash: hashPassword("password123"),
        Email:        "test@example.com",
        Status:       1,
        CreateTime:   time.Now().Unix(),
    }
    for _, fn := range overrides {
        fn(user)
    }
    f.db.Create(user)
    return user
}

// ClusterFactory 集群数据工厂
type ClusterFactory struct { /* ... */ }

// RoleFactory 角色数据工厂
type RoleFactory struct { /* ... */ }
```

## Coverage Strategy

### 目标覆盖率分配

```
┌────────────────────────────────────────────────────────────────┐
│ 模块          │ 当前    │ 目标    │ 优先级  │ 增量测试         │
├────────────────────────────────────────────────────────────────┤
│ user/auth     │ 0%      │ 80%     │ P0      │ ~15 个测试用例   │
│ rbac          │ 14.5%   │ 70%     │ P0      │ ~10 个测试用例   │
│ cluster       │ 12.8%   │ 50%     │ P1      │ ~12 个测试用例   │
│ deployment    │ 18.5%   │ 50%     │ P1      │ ~10 个测试用例   │
│ service       │ 15.2%   │ 40%     │ P2      │ ~8 个测试用例    │
└────────────────────────────────────────────────────────────────┘
```

## Error Handling

### 测试错误分类

```go
// internal/testutil/assertions.go

// AssertErrorCode 断言业务错误码
func AssertErrorCode(t *testing.T, err error, expectedCode int)

// AssertDBRecord 断言数据库记录存在
func AssertDBRecord(t *testing.T, db *gorm.DB, model interface{}, conditions map[string]interface{})

// AssertAuditLog 断言审计日志
func AssertAuditLog(t *testing.T, db *gorm.DB, action string, resourceID uint64)
```

## Dependencies

### 现有依赖 (无需新增)

- `github.com/stretchr/testify` - 断言库
- `gorm.io/driver/sqlite` - 内存数据库
- `github.com/gin-gonic/gin` - HTTP 测试

### 需要完善

- `internal/testutil/mock_cache.go` - 缓存 Mock
- `internal/testutil/mock_k8s.go` - 扩展 K8s Mock 方法

## Migration & Compatibility

### 无破坏性变更

所有测试新增，不修改现有业务代码。

### 测试隔离

- 使用 SQLite 内存数据库
- 每个测试独立数据库实例
- Mock 所有外部依赖 (K8s, SSH, Redis)

## Testing Strategy

### 单元测试

```bash
# 运行所有测试
make test

# 运行特定模块
make test-ai
make test-cluster

# 覆盖率报告
make test-coverage
```

### 集成测试

```bash
# 需要 Docker 环境
make test-integration
```

### E2E 测试

```bash
# 需要 Running 服务
cd e2e && npx playwright test
```
