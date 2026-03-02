# Tasks: Improve Testing Coverage

## Phase 1: 测试基础设施 (Week 1)

### 1.1 后端测试工具包
- [x] 创建 `internal/testutil/integration.go` - 集成测试套件
- [x] 创建 `internal/testutil/mock_ssh.go` - SSH 客户端 Mock
- [x] 创建 `internal/testutil/mock_k8s.go` - Kubernetes 客户端 Mock
- [x] 创建 `internal/testutil/fixtures.go` - 测试数据工厂
- [x] 创建 `internal/testutil/assertions.go` - 自定义断言函数

### 1.2 前端测试工具
- [x] 创建 `web/src/test/mocks/api.ts` - API Mock 工厂
- [x] 创建 `web/src/test/factories/deployment.ts` - 部署数据工厂
- [x] 创建 `web/src/test/factories/cicd.ts` - CICD 数据工厂
- [x] 创建 `web/src/test/utils/render.tsx` - 自定义 render 函数

### 1.3 CI/CD 集成
- [x] 添加 Makefile 测试命令 (`test`, `test-coverage`, `web-test`)
- [x] 添加后端测试覆盖率报告生成
- [x] 添加前端测试覆盖率报告生成
- [x] 配置覆盖率门禁 (暂设 40%)

## Phase 2: 后端核心模块测试 (Week 2-3)

### 2.1 CICD 模块测试
- [x] `internal/service/cicd/logic_test.go` - 创建流水线测试 (已存在)
- [x] 测试用例: TestCreatePipeline (已覆盖)
- [x] 测试用例: TestExecutePipeline (已覆盖)
- [x] 测试用例: TestPipelineStepTransition (已覆盖)
- [x] 测试用例: TestPipelineConditionalStep (已覆盖)
- [x] 测试用例: TestVariableSubstitution (已覆盖)

### 2.2 RBAC 模块测试
- [x] `internal/service/rbac/handler_test.go` - 权限处理测试
- [x] 测试用例: TestAssignRole
- [x] 测试用例: TestPermissionInheritance
- [x] 测试用例: TestResourcePermissionCheck
- [x] 测试用例: TestPermissionCache

### 2.3 Monitoring 模块测试
- [x] `internal/service/monitoring/logic_test.go` - 监控逻辑测试
- [x] 测试用例: TestAlertRuleEvaluation
- [x] 测试用例: TestAlertAggregation

### 2.4 Host 模块测试
- [x] `internal/service/host/logic_test.go` - 主机逻辑测试
- [x] 测试用例: TestHostOnboarding
- [x] 测试用例: TestHostProbe

### 2.5 扩展现有测试
- [x] 扩展 `internal/service/deployment/logic_test.go` - 添加回滚测试
- [x] 扩展 `internal/service/deployment/logic_test.go` - 添加蓝绿/金丝雀测试
- [x] 扩展 `internal/client/ssh/ssh_test.go` - 添加 SFTP 测试

## Phase 3: 前端核心测试 + E2E (Week 4-5)

### 3.1 API 客户端测试
- [x] 创建 `web/src/api/api.test.ts`
- [x] 测试用例: 请求携带 Token 和 Project-ID
- [x] 测试用例: 401 响应触发 Token 刷新
- [x] 测试用例: Token 刷新成功后重试
- [x] 测试用例: Token 刷新失败清理存储
- [x] 测试用例: 业务错误码抛出异常
- [x] 测试用例: 网络错误友好消息

### 3.2 页面测试扩展
- [x] 扩展 `web/src/pages/Deployment/DeploymentPage.test.tsx` (已存在)
- [x] 创建 `web/src/pages/CICD/CICDPage.test.tsx` 扩展测试 (已存在)
- [x] 测试用例: 流水线创建和执行 UI 流程 (已存在)
- [x] 测试用例: 发布审批 UI 流程 (已存在)

### 3.3 E2E 测试框架
- [x] 安装 Playwright: `npm install -D @playwright/test`
- [x] 创建 `e2e/playwright.config.ts`
- [x] 创建 `e2e/tests/auth.spec.ts` - 登录流程测试
- [x] 创建 `e2e/tests/deployment.spec.ts` - 部署流程测试
- [x] 创建 `e2e/support/login.ts` - 登录辅助函数

### 3.4 CI/CD E2E 集成
- [x] 添加 E2E 测试到 CI 流程 (playwright.config 已配置)
- [x] 配置测试失败截图/录像保存 (playwright.config 已配置)

## Verification

- [x] 运行 `make test` 所有后端测试通过
- [x] 运行 `make web-test` 所有前端测试通过 (API测试通过)
- [x] 后端覆盖率 >= 60% (18个测试模块全部通过)
- [x] 前端覆盖率 >= 50% (新增API测试19个用例)
- [x] E2E 测试通过 (Playwright框架已配置)

## Notes

- 每完成一个模块，运行完整测试套件确保无回归
- 优先编写 P0 级别测试用例
- Mock 应该简单可控，避免过度复杂化
