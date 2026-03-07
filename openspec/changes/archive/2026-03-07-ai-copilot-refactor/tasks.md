# 实施任务

## Phase 1: 后端清理与重组

### 1.1 删除冗余代码

- [x] 删除 `internal/service/ai/approval_notifier.go`
- [x] 删除 `internal/service/ai/approval_notifier_test.go`
- [x] 删除 `internal/service/ai/permission_checker.go`
- [x] 删除 `internal/service/ai/permission_checker_test.go`
- [x] 删除空目录 `internal/service/ai/handler/`
- [x] 删除空目录 `internal/service/ai/logic/`
- [x] 运行测试确保编译通过

### 1.2 合并工具分类文件

- [x] 创建 `internal/ai/tools/category.go`
- [x] 合并以下文件内容到 category.go:
  - cicd.go → buildCICDTools()
  - config.go → buildConfigTools()
  - deployment.go → buildDeploymentTools()
  - governance.go → buildGovernanceTools()
  - inventory.go → buildInventoryTools()
  - k8s.go → buildK8sTools()
  - monitor.go → buildMonitorTools()
  - ops.go → buildOpsTools()
  - service.go → buildServiceTools()
- [x] 删除原 9 个文件
- [x] 更新相关 import

### 1.3 重组 internal/ai/ 目录

- [x] 创建 `internal/ai/agent/` 目录
- [x] 移动 agent.go → agent/agent.go
- [x] 移动 runner.go → agent/runner.go
- [x] 移动 tool_registry.go → agent/registry.go
- [x] 创建 `internal/ai/modes/` 目录
- [x] 移动 agentic_mode.go → modes/agentic.go
- [x] 移动 simple_chat_mode.go → modes/simple.go
- [x] 创建 `internal/ai/store/` 目录
- [x] 移动 store.go → store/store.go
- [x] 移动 checkpoint_store.go → store/checkpoint.go
- [x] 创建 `internal/ai/classifier/` 目录
- [x] 移动 classifier.go → classifier/classifier.go
- [x] 创建 `internal/ai/types/` 目录 (新增，解决循环依赖)
- [x] 移动 HybridAgent → internal/ai/hybrid.go (主包，避免循环依赖)
- [x] 更新所有 import 路径
- [x] 更新测试文件
- [x] 运行测试验证

### 1.4 重组 internal/service/ai/ 目录

- [x] 创建 `internal/service/ai/handler/` 目录
- [x] 移动 handler.go → handler/handler.go
- [x] 移动 chat_handler.go → handler/chat_handler.go
- [x] 移动 capability_handler.go → handler/capability_handler.go
- [x] 移动 misc_handler.go → handler/misc_handler.go
- [x] 移动 scene_handler.go → handler/scene_handler.go
- [x] 移动 http_handler.go → handler/http_handler.go
- [x] 移动 events.go → handler/events.go
- [x] 移动 policy.go → handler/policy.go
- [x] 移动 scene_context.go → handler/scene_context.go
- [x] 创建 `internal/service/ai/logic/` 目录
- [x] 移动 confirmation_service.go → logic/confirmation.go
- [x] 移动 preview_builder.go → logic/preview.go
- [x] 创建 logic/runtime.go (RuntimeStore)
- [x] 创建 logic/session.go (SessionStore)
- [x] 创建 `internal/service/ai/events/` 目录
- [x] 移动 events_sse.go → events/sse.go
- [x] 创建 `internal/service/ai/knowledge/` 目录
- [x] 移动 faq_knowledge.go → knowledge/faq.go
- [x] 移动 help_knowledge.go → knowledge/help.go
- [x] 更新 routes.go 导入 handler 子包
- [x] 删除旧的 types.go, util.go, store.go
- [x] 更新所有 import 路径
- [x] 运行测试验证

### 1.5 更新主入口

- [x] 修改 `internal/svc/svc.go`
- [ ] 使用 HybridAgent 替代 PlatformRunner (待前端配合)
- [x] 运行测试验证

---

## Phase 2: 前端重构

### 2.1 安装依赖

- [x] 确认 @ant-design/x-sdk 已安装 (v2.3.0)
- [x] 如需更新: `pnpm add @ant-design/x-sdk`

### 2.2 创建新组件

- [x] 创建 `web/src/components/AI/AICopilotButton.tsx`
  - 单一入口按钮
  - AI Copilot 标识
- [x] 创建 `web/src/components/AI/AICopilotDrawer.tsx` (重构)
  - 场景选择器
  - 保留现有 useAIChat hook
- [x] 创建 `web/src/components/AI/providers/PlatformChatProvider.ts`
  - 继承 AbstractChatProvider
  - 适配现有 SSE API 格式
  - 处理事件类型: meta, delta, tool_call, tool_result, approval_required, done, error
- [x] 创建 `web/src/components/AI/hooks/useXPlatformChat.ts`
  - 封装 useXChat + PlatformChatProvider
  - 提供简洁的 API 供组件使用

### 2.3 创建场景管理

- [x] 创建 `web/src/components/AI/hooks/useAutoScene.ts`
  - 自动检测当前路由对应场景
  - 支持手动切换场景
  - 持久化场景选择

### 2.4 重构现有组件

- [x] 更新 `web/src/components/AI/index.ts`
  - 导出 AICopilotButton 替代 AIAssistantButton
- [x] 更新 `web/src/components/Layout/AppLayout.tsx`
  - 使用 AICopilotButton
- [x] 保留 `web/src/components/AI/AIAssistantButton.tsx` (兼容)
- [x] 保留 `web/src/components/AI/AIAssistantDrawer.tsx` (复用部分逻辑)

### 2.5 样式更新

- [x] 现有样式已兼容
- [x] 添加场景选择器样式 (使用 Select 组件)

---

## Phase 3: 测试与验证

### 3.1 后端测试

- [x] 运行所有单元测试: `go test ./internal/ai/... ./internal/service/ai/...`
- [x] 确保测试通过
- [x] 验证 API 端点正常工作 (需启动服务)

### 3.2 前端测试

- [x] 前端编译通过: `pnpm build`
- [x] PlatformChatProvider 类型检查通过
- [ ] 为新组件编写单元测试
- [ ] 为 PlatformChatProvider 编写测试

### 3.3 集成测试

- [x] 手动测试 AI Copilot 完整流程
- [x] 验证场景自动感知
- [x] 验证场景切换
- [ ] 验证会话管理
- [x] 验证消息发送接收

### 3.4 文档更新

- [ ] 更新 README.md (如有必要)
- [x] 更新内存文件 `.claude/projects/-root-project-k8s-manage/memory/MEMORY.md`
