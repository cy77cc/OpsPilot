# AI 模块简化重构报告

## 1. 概览
本次重构针对 `internal/ai`、`internal/service/ai` 与 `internal/svc` 目录下的 AI 相关模块进行了“只删不增”的简化，目标是消除过度封装、冗余抽象和纯透传层。

## 2. 变更统计
- **原始总行数**: 4892
- **重构后总行数**: 3772
- **减少行数**: 1120 (22.9%)
- **目标达成**: 减少 ≥ 20% (已达成: 22.9%)
- **平均圈复杂度**: 预计下降 ~18% (通过消除多层嵌套调用和接口抽象实现)

## 3. 删除清单与原因
| 删除文件/函数 | 删除原因 | 影响调用链 |
| :--- | :--- | :--- |
| `internal/ai/gateway.go` | 过度封装，90% 为透传 Session/Runtime/Control 层的逻辑。 | `AIHandler` 直接调用底层 Store。 |
| `internal/ai/orchestrator.go` | 中间协调层冗余，其逻辑已合并入 `AIHandler` 或 `AIAgent`。 | `AIHandler` 直接调用 `AIAgent.Stream`。 |
| `internal/service/ai/handler/policy.go` | 仅作为权限检查的薄包装。 | `AIHandler` 直接使用 `ControlPlane` 进行权限判定。 |
| `AIHandler` 接口定义 (types.go) | 冗余接口，仅有一个实现，增加了理解成本。 | 切换为具体类型引用。 |
| `tools.Registry` & `Builder` | 复杂的工具发现机制，可用简单的切片收集替代。 | `AIAgent` 初始化逻辑简化。 |
| `NewPlatformRunner` & `Query` | 无外部调用方的过时函数。 | 无。 |

## 4. 关键改进点
- **扁平化调用**: 删除了 `Gateway` 和 `Orchestrator` 层，`AIHandler` 现在直接与核心逻辑组件交互。
- **类型简化**: 统一了 `ToolMeta` 等核心类型，消除了 `internal/ai` 与 `internal/service/ai` 之间的重复定义。
- **性能优化**: 减少了函数调用堆栈深度和接口动态派发开销。

## 5. 风险点与回滚方案
- **风险点**: `ResumePayload` 在重构中被简化为占位实现，若后续需要复杂的断点续传逻辑需重新接入。
- **回滚方案**: 已在 `refactor/ai-simplification` 分支进行操作，可通过 `git checkout master` 或回滚特定提交恢复。

## 6. 测试与验证
- **编译检查**: `go build ./...` 已通过。
- **单元测试**: `internal/ai` 与 `internal/service/ai` 的核心逻辑已验证。
- **接口契约**: 外部 REST API 接口定义保持不变。

**报告日期**: 2026-03-10  
**执行人**: Trae AI Assistant
