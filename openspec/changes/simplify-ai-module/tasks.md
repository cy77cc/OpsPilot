# Tasks: simplify-ai-module

## Phase 1 — 合并 EventMeta（B）

- [x] 1.1 在 `executor/executor.go` 中删除 `EventMeta` struct 和 `EventEmitter` type，改为 import `events` 包，`EventEmitter` 函数签名改为 `func(name string, meta events.EventMeta, data map[string]any) bool`
- [x] 1.2 更新 `executor/events.go` 中四个 emit 函数（`emitStepUpdate`、`emitToolCall`、`emitToolResult`、`emitApprovalRequired`）的 `EventMeta` 类型引用，改用 `events.EventMeta`
- [x] 1.3 更新 `executor/executor.go` 中 `Request.EventMeta` 字段类型为 `events.EventMeta`，更新 `WithDefaults()` 调用
- [x] 1.4 更新 `orchestrator.go`：删除 `EmitEvent` 回调中手工构造 `executor.EventMeta` 的转换块（约 15 行），改为直接传递 `events.EventMeta`；更新 `executor.Request` 中的 `EventMeta` 赋值
- [x] 1.5 更新 `executor/executor_test.go`，将测试中的 `executor.EventMeta{...}` 改为 `events.EventMeta{...}`
- [x] 1.6 验证：`go build ./internal/ai/...` 通过，`go test ./internal/ai/executor/...` 通过

## Phase 2 — 拆分 planner.go（C）

- [x] 2.1 创建 `planner/types.go`：将 `Decision`、`DecisionType`、`ExecutionPlan`、`PlanStep`、`ResourceRef`、`PodRef`、`ResourceScope`、`ResolvedResources`、`PlanningError` 等所有类型定义移入，保留 `PlanningError` 的方法
- [x] 2.2 创建 `planner/parse.go`：将 `ParseDecision`、`parseExecutionPlan`、`parseResolvedResources`、`parsePlanSteps`、`parseResourceRefs`、`parsePodRefs`、`parseResourceScope`、`looseStringValue`、`looseIntValue`、`stringSliceValue`、`intSliceValue`、`mapSliceValue` 移入
- [x] 2.3 创建 `planner/normalize.go`：将 `normalizeDecision`、`canonicalizeDecision`、`canonicalizePlan`、`populateStepInput`、`validatePlanPrerequisites`、`baseStepInput`、`mergeStepInput`、`normalizeModeRisk`、`normalizedRisk`、所有 `requires*`、`resolved*`、`hasResolvedScope`、`hasTargetType`、`targetNameForType`、`isSupportedExpert`、`isSupportedMode`、`isSupportedRisk`、`cloneInput`、`cloneResourceRefs`、`clonePodRefs`、`cloneScope`、`scopeToMap`、`resourceIDs`、`podRefsFromInput`、`podRefsToAny` 移入
- [x] 2.4 创建 `planner/collect.go`：将 `buildBasePlanContext`、`buildPromptInput`、`collectHostNames`、`collectHostIDs`、`collectPodName`、`collectServices`、`collectClusters`、`collectHosts`、`collectPods`、`detectScope`、`dedupeResourceRefs`、`dedupePodRefs`、`dedupe`、`firstNonEmpty`、`isAllKeyword` 移入
- [x] 2.5 精简 `planner/planner.go`：仅保留 `Planner` struct、`New`、`NewWithFunc`、`Plan`、`PlanStream`、`plan` 方法，以及必要的 import
- [x] 2.6 确认 `planner/support.go` 中的内容是否已被覆盖，若有重叠则合并或删除重复函数
- [x] 2.7 验证：`go build ./internal/ai/planner/...` 通过，`go test ./internal/ai/planner/...` 通过

## Phase 3 — 简化 ResolvedResources（A）

- [ ] 3.1 在 `planner/types.go` 中修改 `ResolvedResources`：删除平铺字段 `ServiceName`、`ServiceID`、`ClusterName`、`ClusterID`、`HostNames []string`、`HostIDs []int`、`PodName`；保留 `Services []ResourceRef`、`Clusters []ResourceRef`、`Hosts []ResourceRef`、`Pods []PodRef`、`Namespace string`、`Scope *ResourceScope`
- [ ] 3.2 在 `planner/normalize.go`（或新建 `planner/helpers.go`）中添加包级 helper：`primaryID(refs []ResourceRef) int`、`primaryName(refs []ResourceRef) string`、`allIDs(refs []ResourceRef) []int`
- [ ] 3.3 更新 `planner/parse.go` 的 `parseResolvedResources`：继续从 LLM JSON 读取平铺字段（`service_id`、`service_name`、`cluster_id` 等），但将其归并到 `Services`/`Clusters`/`Hosts`/`Pods` 列表字段中，不再赋值到已删除的平铺字段
- [ ] 3.4 更新 `planner/normalize.go` 的 `canonicalizePlan`：将原来的 11 个 fallback 块替换为 5 个（`Services`、`Clusters`、`Hosts`、`Pods`、`Namespace`），删除平铺字段的 fallback
- [ ] 3.5 更新 `planner/normalize.go` 的 `populateStepInput`：用 `primaryID(resolved.Services)`、`primaryID(resolved.Clusters)` 等替换 `resolved.ServiceID`、`resolved.ClusterID` 等直接引用
- [ ] 3.6 更新 `planner/normalize.go` 的所有 `resolved*` helper 函数（`resolvedServiceID`、`resolvedClusterID`、`resolvedHostIDs`、`resolvedPodName`）：使用列表字段和新 helper 函数
- [ ] 3.7 更新 `planner/normalize.go` 的 `requiresHostTarget`：移除对 `resolved.HostIDs`、`resolved.HostNames` 的引用，改用 `resolved.Hosts`
- [ ] 3.8 更新 `planner/collect.go` 的 `buildBasePlanContext`：构造 `ResolvedResources` 时只赋值列表字段
- [ ] 3.9 检索全项目是否有其他地方访问 `ResolvedResources` 的平铺字段（`grep -r "\.ServiceID\|\.ClusterID\|\.HostIDs\|\.HostNames\|\.PodName\|\.ServiceName\|\.ClusterName" internal/ai/`），逐一更新
- [ ] 3.10 更新相关测试（`planner_test.go`、`support_test.go`）中构造 `ResolvedResources` 的地方，移除平铺字段赋值
- [ ] 3.11 验证：`go build ./internal/ai/...` 通过，`go test ./internal/ai/planner/... ./internal/ai/executor/...` 通过

## Phase 4 — 最终验证

- [ ] 4.1 运行 `go build ./internal/... ./api/...` 确认整体编译通过
- [ ] 4.2 运行 `go test ./internal/ai/...` 确认所有 AI 模块测试通过
- [ ] 4.3 检查 `planner/planner.go` 行数 ≤ 120，`planner/parse.go`、`planner/normalize.go`、`planner/collect.go` 各自 ≤ 450 行
- [ ] 4.4 运行 `grep -r "executor\.EventMeta" internal/` 确认无残留引用
- [ ] 4.5 运行 `grep -r "\.ServiceID\|\.ClusterID\b" internal/ai/` 确认 `planner` 包内无平铺字段残留（orchestrator 等外层不持有 `ResolvedResources` 故无影响）
