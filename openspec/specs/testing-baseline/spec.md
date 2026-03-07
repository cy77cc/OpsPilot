# Spec: Testing Baseline

## Capability

定义项目测试标准和最佳实践，确保代码质量和可维护性。

## Requirements

### REQ-TEST-001: 测试覆盖率门禁

**As** 开发团队
**I want** 代码合并前自动检查测试覆盖率
**So that** 确保测试质量不会下降

**Acceptance Criteria:**
- 后端测试覆盖率 >= 60%
- 前端测试覆盖率 >= 50%
- CI 流程中覆盖率低于阈值则失败

---

### REQ-TEST-002: 单元测试标准

**As** 开发者
**I want** 有清晰的单元测试编写指南
**So that** 测试代码风格一致且易于维护

**Acceptance Criteria:**
- 每个测试用例独立，不依赖执行顺序
- 使用表驱动测试 (table-driven tests) 覆盖多场景
- Mock 外部依赖 (SSH, K8s, AI, DB)
- 测试函数命名: `Test<FunctionName>_<Scenario>`

---

### REQ-TEST-003: 集成测试标准

**As** 开发者
**I want** 有集成测试工具包
**So that** 可以快速编写端到端的 API 测试

**Acceptance Criteria:**
- 提供 `IntegrationSuite` 测试套件
- 内置 SQLite 内存数据库
- 内置 Mock SSH/K8s/AI 客户端
- 提供测试数据工厂函数

---

### REQ-TEST-004: E2E 测试标准

**As** QA 工程师
**I want** 有关键用户旅程的 E2E 测试
**So that** 可以验证完整用户流程

**Acceptance Criteria:**
- 使用 Playwright 框架
- 覆盖登录、部署、AI 对话等关键旅程
- 测试失败时保存截图和录像
- 支持多浏览器测试

---

### REQ-TEST-005: CI/CD 测试集成

**As** DevOps 工程师
**I want** 测试在 CI 中自动运行
**So that** 代码质量得到持续保障

**Acceptance Criteria:**
- PR 提交时运行单元测试和集成测试
- Main 分支合并时运行 E2E 测试
- 生成并存储覆盖率报告
- 覆盖率低于阈值则 CI 失败

## Test Data Management

### Fixtures Factory Pattern

```go
// 后端示例
func (s *IntegrationSuite) SeedUser(overrides ...func(*model.User)) *model.User {
    user := &model.User{
        Username: "testuser",
        Email:    "test@example.com",
        Status:   "active",
    }
    for _, fn := range overrides {
        fn(user)
    }
    s.DB.Create(user)
    return user
}

// 使用
user := s.SeedUser(func(u *model.User) {
    u.Role = "admin"
})
```

```typescript
// 前端示例
export function createDeployTarget(overrides?: Partial<DeployTarget>): DeployTarget {
  return {
    id: 1,
    name: 'test-target',
    target_type: 'k8s',
    runtime_type: 'k8s',
    env: 'staging',
    status: 'active',
    ...overrides,
  };
}
```

## Mock Guidelines

### When to Mock

```
必须 Mock:
├── 外部网络调用 (SSH, K8s API, AI API)
├── 文件系统操作
├── 时间相关函数
└── 随机数生成

不应 Mock:
├── 纯逻辑函数
├── 数据库 (使用内存 SQLite)
└── 标准库函数
```

### Mock Interface Design

```go
// 定义接口
type SSHExecutor interface {
    Exec(cmd string) (stdout, stderr string, err error)
}

// 真实实现
type RealSSHClient struct { ... }

// Mock 实现
type MockSSHClient struct {
    ExecFunc func(cmd string) (string, string, error)
}
```

## Anti-Patterns to Avoid

1. **测试私有函数** - 通过公共接口测试
2. **过度 Mock** - Mock 应该简单可控
3. **测试实现细节** - 测试行为，不是实现
4. **共享测试状态** - 每个测试应该独立
5. **断言不足** - 验证所有重要输出

---

## Frontend Runtime Smoke Verification

### Requirement: Frontend production runtime smoke verification MUST detect white-screen regressions

The system MUST include a frontend runtime smoke verification step that exercises the production build in a browser-like environment and detects fatal initialization failures before release.

#### Scenario: Fatal production runtime error

- **WHEN** the frontend production build is loaded in the smoke verification environment
- **THEN** the verification fails if a fatal page or module initialization error occurs
- **AND** the failure is reported before release

### Requirement: Frontend smoke verification MUST prove shell availability independent of AI surface health

The system MUST verify that the main application shell renders even when the AI assistant surface cannot initialize successfully.

#### Scenario: AI surface fails during smoke verification

- **WHEN** the smoke verification simulates or encounters an AI surface initialization failure
- **THEN** the application shell still renders
- **AND** the verification confirms that the failure remains local to the AI surface

## References

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Testing Library Principles](https://testing-library.com/docs/guiding-principles)
- [Playwright Best Practices](https://playwright.dev/docs/best-practices)
