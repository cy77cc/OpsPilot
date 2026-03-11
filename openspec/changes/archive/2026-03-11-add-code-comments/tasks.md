# Tasks: 添加代码注释

## Phase 1: AI 模块核心文件

### 1.1 编排核心
- [x] internal/ai/orchestrator.go - AI 编排核心
- [x] internal/ai/gateway_contract.go - 网关契约定义
- [x] internal/ai/config.go - 配置管理
- [x] internal/ai/model.go - 模型初始化
- [x] internal/ai/metrics.go - 指标收集
- [x] internal/ai/errors.go - 错误定义
- [x] internal/ai/final_answer_renderer.go - 最终答案渲染

### 1.2 执行器模块
- [x] internal/ai/executor/executor.go - 执行器核心
- [x] internal/ai/executor/expert_runner.go - 专家运行器
- [x] internal/ai/executor/scheduler.go - 调度器
- [x] internal/ai/executor/events.go - 事件定义

### 1.3 规划器模块
- [x] internal/ai/planner/planner.go - 规划器核心
- [x] internal/ai/planner/adk.go - ADK 集成
- [x] internal/ai/planner/decision_tools.go - 决策工具
- [x] internal/ai/planner/prompt.go - 提示词
- [x] internal/ai/planner/support.go - 辅助函数

### 1.4 改写器模块
- [x] internal/ai/rewrite/rewrite.go - 改写器核心
- [x] internal/ai/rewrite/adk.go - ADK 集成
- [x] internal/ai/rewrite/prompt.go - 提示词

### 1.5 总结器模块
- [x] internal/ai/summarizer/summarizer.go - 总结器核心
- [x] internal/ai/summarizer/adk.go - ADK 集成
- [x] internal/ai/summarizer/decision_tools.go - 决策工具
- [x] internal/ai/summarizer/prompt.go - 提示词

### 1.6 状态与运行时
- [x] internal/ai/state/state.go - 状态管理
- [x] internal/ai/state/chat_store.go - 聊天存储
- [x] internal/ai/runtime/execution_state.go - 执行状态

### 1.7 专家模块
- [x] internal/ai/experts/registry.go - 专家注册表
- [x] internal/ai/experts/spec/spec.go - 专家规范
- [x] internal/ai/experts/hostops/expert.go - 主机运维专家
- [x] internal/ai/experts/k8s/expert.go - K8s 专家
- [x] internal/ai/experts/service/expert.go - 服务专家
- [x] internal/ai/experts/observability/expert.go - 可观测性专家
- [x] internal/ai/experts/delivery/expert.go - 交付专家

### 1.8 工具模块
- [x] internal/ai/tools/tools.go - 工具入口
- [x] internal/ai/tools/common/common.go - 通用工具
- [x] internal/ai/tools/host/tools.go - 主机工具
- [x] internal/ai/tools/host/runtime.go - 主机运行时
- [x] internal/ai/tools/kubernetes/tools.go - K8s 工具
- [x] internal/ai/tools/service/tools.go - 服务工具
- [x] internal/ai/tools/deployment/tools.go - 部署工具
- [x] internal/ai/tools/cicd/tools.go - CI/CD 工具
- [x] internal/ai/tools/monitor/tools.go - 监控工具
- [x] internal/ai/tools/infrastructure/tools.go - 基础设施工具
- [x] internal/ai/tools/governance/tools.go - 治理工具

### 1.9 事件与 RAG
- [x] internal/ai/events/events.go - 事件定义
- [x] internal/ai/events/payloads.go - 事件载荷
- [x] internal/ai/availability/messages.go - 可用性消息
- [x] internal/ai/rag/feedback.go - RAG 反馈
- [x] internal/ai/rag/indexer.go - RAG 索引
- [x] internal/ai/rag/retriever.go - RAG 检索
- [x] internal/ai/rag/rewrite_bridge.go - 改写桥接

---

## Phase 2: Service 层

### 2.1 AI 服务
- [x] internal/service/ai/routes.go - 路由注册
- [x] internal/service/ai/handler.go - HTTP 处理器
- [x] internal/service/ai/session_recorder.go - 会话记录

### 2.2 主机服务
- [x] internal/service/host/routes.go - 路由注册

### 2.3 集群服务
- [x] internal/service/cluster/routes.go - 路由注册

### 2.4 部署服务
- [x] internal/service/deployment/routes.go - 路由注册

### 2.5 监控服务
- [x] internal/service/monitoring/routes.go - 路由注册

### 2.6 其他服务
- [x] internal/service/user/routes.go
- [x] internal/service/rbac/routes.go
- [x] internal/service/cicd/routes.go
- [x] internal/service/jobs/routes.go
- [x] internal/service/cmdb/routes.go
- [x] internal/service/topology/routes.go
- [x] internal/service/project/routes.go
- [x] internal/service/dashboard/routes.go
- [x] internal/service/automation/routes.go
- [x] internal/service/service/routes.go
- [x] internal/service/node/routes.go

---

## Phase 3: 基础设施

### 3.1 中间件
- [x] internal/middleware/jwt.go - JWT 认证
- [x] internal/middleware/casbin.go - 权限控制
- [x] internal/middleware/cors.go - CORS 处理
- [x] internal/middleware/logger.go - 日志记录
- [x] internal/middleware/context.go - 上下文处理

### 3.2 工具函数
- [x] internal/utils/jwt.go - JWT 工具
- [x] internal/utils/crypt.go - 加密工具
- [x] internal/utils/password.go - 密码工具
- [x] internal/utils/secret.go - 密钥工具
- [x] internal/utils/utils.go - 通用工具

### 3.3 DAO 层
- [x] internal/dao/user/*.go - 用户 DAO
- [x] internal/dao/node/*.go - 节点 DAO

### 3.4 其他基础设施
- [x] internal/config/config.go - 配置加载
- [x] internal/cache/facade.go - 缓存门面
- [x] internal/cache/l2_redis.go - Redis 缓存
- [x] internal/httpx/*.go - HTTP 工具
- [x] internal/xcode/*.go - 错误码
- [x] internal/logger/*.go - 日志
- [x] internal/svc/*.go - 服务上下文
- [x] internal/server/*.go - 服务器
- [x] internal/cmd/*.go - 命令入口
- [x] internal/client/ssh/*.go - SSH 客户端
- [x] internal/client/http/*.go - HTTP 客户端 (目录不存在)
- [x] internal/component/casbin/*.go - Casbin 组件
- [x] internal/websocket/*.go - WebSocket
- [x] internal/pki/*.go - PKI
- [x] internal/infra/prometheus/*.go - Prometheus
- [x] internal/constants/*.go - 常量

---

## Phase 4: Model 层

- [x] internal/model/ai_chat.go
- [x] internal/model/ai_approval_task.go
- [x] internal/model/ai_checkpoint.go
- [x] internal/model/ai_command.go
- [x] internal/model/ai_confirmation.go
- [x] internal/model/ai_scene_prompt.go
- [x] internal/model/aiops.go
- [x] internal/model/user.go
- [x] internal/model/role.go (定义在 user.go 中)
- [x] internal/model/permission.go (定义在 user.go 中)
- [x] 其他 model 文件 (cluster.go, node.go, notification.go, deployment.go, project.go 等)

---

## 统计

| Phase | 文件数 | 代码行数 |
|-------|--------|----------|
| Phase 1: AI 模块 | 51 | 11,371 |
| Phase 2: Service 层 | ~120 | 24,655 |
| Phase 3: 基础设施 | ~40 | ~4,000 |
| Phase 4: Model 层 | ~26 | 1,687 |
| **总计** | **~237** | **~41,713** |
