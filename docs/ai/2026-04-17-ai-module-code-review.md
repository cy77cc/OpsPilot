# AI 模块 Code Review 记录

日期：2026-04-17

范围：`internal/modules/ai` 及其直接 HTTP/logic/dao/runtime/tool 边界

已执行验证：

- `go test ./internal/modules/ai/...`
- `go test -race ./internal/modules/ai/logic/... ./internal/modules/ai/dao/... ./internal/modules/ai/handler/... ./internal/modules/ai/agent/runtime ./internal/modules/ai/agent/shared/middleware ./internal/modules/ai/infra/...`

补充说明：

- `go test -race` 在 `internal/modules/ai/handler/chat` 失败，触发点位于 `TestChatHandler_ReconnectReplaysOnlyEventsAfterLastEventID` 对 `httptest.ResponseRecorder` 的并发读写。该信号更偏向测试代码竞争，不作为本文的主缺陷结论，但说明该场景的并发验证仍不扎实。

当前已确认问题如下。

## 修复状态

- 2026-04-18：本轮已在当前工作区修复文档中的 1-7 项问题，并补充了对应回归测试。
- 验证结果：
  - `go test ./internal/modules/ai/...`
  - `go test -race ./internal/modules/ai/logic/... ./internal/modules/ai/handler/chat ./internal/modules/ai/dao/chat ./internal/modules/ai/dao/approval ./internal/modules/ai/agent/tools ./internal/modules/ai/logic/metrics`

## 1. 审批通过后并不会真正恢复执行

- 缺陷级别：致命
- 定位：
  - `internal/modules/ai/logic/approval/worker.go:64`
  - `internal/modules/ai/logic/approval/worker.go:70`
  - `internal/modules/ai/logic/approval/worker.go:268`
  - `internal/modules/ai/logic/approval/worker.go:310`

问题分析：

`Worker` 暴露了 `resume ResumeFunc` 和 `WithWorkerResume(...)` 扩展点，但在 `processEvent -> resumeApproved` 路径里从未调用该恢复函数。当前实现只是把 run 状态更新成 `resuming`，随后 `RunOnce` 会把 outbox 事件标记为 `done`。这意味着审批通过后，检查点恢复并未发生，运行会永久停留在 `resuming`，而且事件已经被消费，后台不会再重试。

重构方案：

在 `resumeApproved` 中真正执行恢复逻辑，并把恢复成功/失败显式写回状态和生命周期事件；只有在恢复动作完成或明确判定失败后才允许 outbox 进入完成态。建议把“拿租约 -> 调用 resume -> 根据结果发出 resumed/resume_failed 事件 -> 释放/续约租约”做成单一事务边界外的明确状态机。

## 2. 审批过期不会传播到 run 生命周期，运行可能永久卡死

- 缺陷级别：致命
- 定位：
  - `internal/modules/ai/logic/approval/expirer.go:57`
  - `internal/modules/ai/logic/approval/expirer.go:81`
  - `internal/modules/ai/logic/approval/write_model.go:206`
  - `internal/modules/ai/logic/approval/write_model.go:246`

问题分析：

`Expirer.RunOnce` 扫描到超时审批后，只在 `expireTask` 里把任务状态更新成 `expired`，没有发出任何 outbox 事件，也没有更新关联 run。另一方面，`Worker` 只消费 `approval_decided` 事件，不会主动扫描已过期任务。结果是审批任务虽然过期了，但 run 仍可能一直停留在 `waiting_approval`，前端和下游状态机都拿不到终结信号。

重构方案：

过期扫描必须和 lifecycle 事件绑定。最低要求是在过期时调用统一写模型发出 `ai.approval.expired` / `ai.run.completed|cancelled` 类事件，并原子更新 run 状态；更好的做法是完全复用 `WriteModel.emitLifecycle(...)` 路径，避免再维护一套“只改 task 不改 run”的旁路状态机。

## 3. 审批终结路径吞掉持久化错误，可能导致事件丢失

- 缺陷级别：警告
- 定位：
  - `internal/modules/ai/logic/approval/worker.go:294`
  - `internal/modules/ai/logic/approval/worker.go:298`
  - `internal/modules/ai/logic/approval/worker.go:302`
  - `internal/modules/ai/logic/approval/worker.go:306`

问题分析：

`expireAndFinalize` 和 `finalize` 对数据库更新全部使用 `_ = ...` 丢弃错误。上层 `RunOnce` 只要收到 `nil` 就会把 outbox 标记为 `done`。一旦 run 状态更新失败，审批事件会被错误地视为已完成，但 run 仍停留在旧状态，例如继续显示 `waiting_approval`。

重构方案：

这些分支必须把真实错误向上返回，让 `RunOnce` 进入 `MarkRetry` 分支。否则 outbox 的“至少一次”语义被破坏，状态机无法自愈。

## 4. Metrics 回调为每次请求创建脱离上下文的 goroutine

- 缺陷级别：警告
- 定位：
  - `internal/modules/ai/logic/metrics/handler.go:75`
  - `internal/modules/ai/logic/metrics/handler.go:76`
  - `internal/modules/ai/logic/metrics/handler.go:84`
  - `internal/modules/ai/logic/metrics/handler.go:87`
  - `internal/modules/ai/logic/metrics/handler.go:136`

问题分析：

`OnEndFn`、`OnEndWithStreamOutputFn`、`OnErrorFn` 都直接启动 goroutine 写库，既没有并发上限，也没有超时，也没有错误记录，还忽略了原始 `ctx` 的取消信号。高并发流式请求下，这会把“每次模型调用”放大成“每次调用至少一个后台 goroutine + 两次 DB 写入”，容易把连接池和 goroutine 数量顶高，同时默默丢指标。

重构方案：

改为显式的异步指标队列或有界 worker pool，写入时使用短超时上下文并记录失败原因。流式指标收尾逻辑也应避免无限期等待私有流消费结束。

## 5. 工具构造失败会直接 panic，导致 AI 服务整体崩溃

- 缺陷级别：警告
- 定位：
  - `internal/modules/ai/agent/tools/host/tools.go:167`
  - `internal/modules/ai/agent/tools/host/tools.go:193`
  - `internal/modules/ai/agent/tools/orchestrator/platform_discovery.go:47`
  - `internal/modules/ai/agent/tools/orchestrator/platform_discovery.go:83`

问题分析：

多个 tool factory 在 `InferOptionableTool(...)` 返回错误时直接 `panic(err)`。这类函数会在运行期按场景构建工具集，只要某个 schema 推导、依赖注入或未来字段调整出错，整个进程就会因为一次 AI 请求直接崩溃，而不是降级成“该工具不可用”。

重构方案：

把 tool 构造改成显式返回错误，由工具注册层统一决定是跳过该工具、记录错误，还是返回受控失败。进程级 panic 不应作为业务输入校验的错误通道。

## 6. 顺序号分配使用 `MAX(...) + 1`，并发下会产生冲突

- 缺陷级别：警告
- 定位：
  - `internal/modules/ai/dao/chat/dao.go:200`
  - `internal/modules/ai/dao/chat/dao.go:204`
  - `internal/modules/ai/dao/run/dao.go:228`
  - `internal/modules/ai/dao/run/dao.go:260`
  - `internal/modules/ai/dao/approval/outbox_dao.go:34`
  - `internal/modules/ai/logic/approval/write_model.go:270`
  - `internal/modules/ai/logic/approval/write_model.go:274`

问题分析：

消息 `session_id_num` 和审批 outbox `sequence` 都通过“查当前最大值再加一”分配，没有行级锁，也没有重复键重试。并发请求打到同一 session 或同一 run 时，两条事务可能拿到相同的下一个序号，最终触发唯一键冲突，或者导致顺序依赖逻辑出现抖动。

重构方案：

把顺序号分配改成数据库原子递增模型，例如：

- 单独的 sequence 表 + `SELECT ... FOR UPDATE`
- 数据库 sequence / auto increment + 逻辑映射
- 在唯一键冲突时做有限次重试

当前实现更像“单线程假设”，对真实并发负载不稳。

## 7. Service 构造器允许空逻辑对象，但大多数方法会直接空指针崩溃

- 缺陷级别：建议
- 定位：
  - `internal/modules/ai/handler/chat/service.go:29`
  - `internal/modules/ai/handler/chat/service.go:30`
  - `internal/modules/ai/handler/chat/service.go:50`
  - `internal/modules/ai/handler/chat/service.go:54`
  - `internal/modules/ai/handler/chat/service.go:58`

问题分析：

`NewServiceWithLogic(nil)` 会返回 `&Service{}`，但除 `Chat`/`BuildResumableCredentials` 外，大多数方法都直接调用 `s.logic...`。这让构造器语义和方法安全性不一致，属于典型的“表面可空，实际不可空”接口设计。

重构方案：

二选一即可：

- 构造器禁止 `nil`，直接 panic 或返回错误
- 所有方法统一做 `nil` 防御并返回显式错误

当前这种半防御状态会把初始化错误延后成运行时 panic。
