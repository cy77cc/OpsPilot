# AI 失败会话持久化设计

**日期**: 2026-03-20
**状态**: Draft

## 背景

当前 AI 会话链路已经支持成功回复的会话持久化，也支持部分运行时失败的 `run` 收尾，但失败路径仍存在明显缺口：

- 某些失败发生在 `run` 或 assistant message 创建之前，数据库中不会留下这轮会话痕迹
- 前端当前轮收到失败后，部分路径仍停留在 `loading/streaming` 视觉状态
- 执行中途失败时，已经成功生成的回答内容没有被稳定地作为最终回复快照持久化
- 历史会话对 projection 依赖过强，一旦 projection 不完整，失败回复容易不可见

这会导致两个直接问题：

1. 用户看见一轮会话已经失败，但刷新后历史里找不到这轮记录
2. 当前会话失败后，assistant 气泡仍然像在继续生成，状态不收敛

## 目标

### 主目标

1. AI 会话开始后立即创建持久化外壳，保证每一轮对话都有数据库归属
2. 无论失败发生在开始阶段还是执行中阶段，都必须留下可见的失败记录
3. 执行中失败时，已经成功生成的内容必须作为最终回复的一部分被保存
4. 前端收到失败信号后必须立即从 `loading` 收敛到 `error`
5. 历史会话展示必须优先可见失败消息本身，不能因为 projection 缺失而整轮丢失

### 非目标

1. 不在本次设计中改造整体 AI 抽屉视觉样式
2. 不在本次设计中引入新的流式协议事件类型
3. 不在本次设计中重做 projection 数据结构
4. 不在本次设计中处理历史脏数据回填

## 产品决策

本次采用“会话开始即创建外壳，失败统一收尾，message/run 作为最低保底”的方案。

核心理由：

- 失败持久化的第一优先级是“这轮不能丢”，而不是“所有结构都完整”
- `message + run` 足以承载用户可见结果和执行归属
- `projection` 应作为增强层存在，不能反过来阻塞失败记录落库
- 只要当前轮已经进入聊天生命周期，就必须有历史可追溯对象

补充决策：

- 本轮创建外壳必须支持幂等，避免前端重试产生重复空壳
- 错误说明默认不直接拼进 markdown 正文，而是作为独立错误态信息展示
- 非终态 run 需要有超时回收机制，避免永久停留在 `running/streaming`

## 生命周期设计

一次聊天的服务端生命周期调整为以下顺序：

1. 确保 `session` 存在
2. 创建 `user message`
3. 创建 `assistant message` 占位，初始状态为 `streaming`
4. 创建 `run`，初始状态为 `running`
5. 发送 `meta` 并进入 AI router / runner 执行
6. 按事件流持续积累已生成正文和结构化事件
7. 在成功或失败时统一进入收尾逻辑

### 幂等键要求

由于前端可能因网络抖动、超时或用户重复提交而重试同一轮请求，本次要求引入请求级幂等键。

建议字段：

- `client_request_id`

约束：

- 同一用户、同一 session、同一 `client_request_id` 只能创建一组 `user message + assistant message + run`
- 若后端收到重复请求，应返回已创建的外壳对象，而不是再次创建新记录
- 幂等键作用域应覆盖“开始即创建外壳”的全过程

推荐实现：

- 前端每次发送新一轮消息时生成一个稳定 nonce
- 后端在 `Chat` 入口先基于 `user_id + session_id + client_request_id` 做查重
- 若发现已存在未完成或已完成的同轮记录，则直接复用

### 关键约束

- `session`、`user message`、`assistant message`、`run` 的创建时机必须前移到实际执行之前
- 从第 4 步开始，后续任意错误都必须找到唯一的 assistant message 和 run 进行收尾
- 一旦 assistant message 已创建，本轮就不允许只靠 SSE 顶层报错而不落库

## 数据分层原则

本次明确三层职责：

### 1. `ai_chat_messages`

这是用户可见结果的最低保底层。

要求：

- assistant 失败消息必须始终可从这里读到
- `content` 保存最终可展示文本快照
- `status` 明确区分 `done` 与 `error`

### 2. `ai_runs`

这是一次执行的归属边界。

要求：

- 每轮会话开始即创建
- 失败时必须记录终态状态与 `error_message`
- 即使 projection 不存在，run 也必须存在
- 长时间未收尾时必须支持被后台任务标记为超时终态

### 3. `ai_run_projections`

这是结构化增强层，不是失败留痕的前提条件。

要求：

- 能写 projection 时写完整 projection
- 只能写最小 error projection 时允许降级
- projection 写入失败不能回滚已存在的 message/run 失败记录

## 失败分类

本次保留三类失败/终止语义。

### 1. 启动前失败

定义：

- 已经创建 session、message、run
- 但在真正消费到可见 AI 输出之前失败

典型场景：

- runner 初始化失败
- 首次拉取事件前即报错
- 进入执行前的内部准备异常

持久化规则：

- `assistant message.status = error`
- `assistant message.content = 错误说明`
- `run.status = failed`
- `run.error_message = 错误说明`
- 若条件允许，补一条最小 `error` event
- 若条件允许，落一份只包含 `error` block 的最小 projection

### 2. 执行中失败

定义：

- 已经产生部分 delta / plan / tool / result / summary
- 后续执行中断或出现不可恢复错误

典型场景：

- stream 中途断开
- 工具执行 fatal error
- projector 或 message stream 接收过程中失败

持久化规则：

- `assistant message.status = error`
- `assistant message.content = 已成功生成的正文 + 错误说明`
- `run.status = failed_runtime`
- `run.error_message = 错误说明`
- 保留已有 event log，并补一条 `error` event
- projection 以“已有内容 + error block”方式收尾

### 3. 超时失活

定义：

- 已创建 session、message、run
- 但由于进程崩溃、Pod 重启、连接永久中断等原因，正常 finalize/fail path 根本没有执行完成

典型场景：

- 后端 worker 崩溃
- 节点重启导致 SSE 中断
- run 长时间处于 `running` 且无新事件

持久化规则：

- `assistant message.status = error`
- `assistant message.content` 保留已落库正文；若无正文则写入统一超时文案
- `run.status = expired`
- `run.error_message = 统一超时文案或内部诊断摘要`

回收策略：

- 后台 Job 周期性扫描超过阈值且仍为非终态的 run
- 根据最近事件时间或创建时间判定是否失活
- 一旦标记为 `expired`，同步更新 assistant message，避免历史会话长期显示 `streaming`

## 内容持久化规则

### 最终回复快照

assistant message 的 `content` 必须始终代表用户最终能看到的完整文本快照。

规则如下：

1. 成功完成时，保存最终正文
2. 执行中失败时，保存“已生成正文 + 错误说明”
3. 启动前失败且无正文时，直接保存错误说明

### 错误说明展示策略

默认策略调整为“正文与错误信息分离展示”，而不是无条件把错误说明拼进 markdown 正文。

原因：

- 流式中断可能发生在 Markdown 表格、列表、代码块中间
- 直接拼接错误说明会破坏 Markdown 结构，导致渲染错乱

规则：

1. assistant message `content` 优先保存已成功生成的正文快照
2. 错误说明作为独立错误态信息保存并展示
3. 若前端当前组件仍必须依赖单字符串展示，可在兜底模式下拼接，但需要先做最小闭合修复

推荐方向：

- 前端将错误说明渲染为独立 Error Callout / footer 状态块
- `runtime.status` 或专门错误字段负责承载失败说明
- `message.content` 只承载正文，不强行混入错误说明

兜底要求：

- 若历史兼容路径必须拼接字符串，至少要对未闭合代码块等常见 Markdown 结构做简单修复

## 收尾逻辑设计

当前代码中的成功收尾、运行时失败收尾和顶层 handler 兜底失败需要收敛成统一逻辑。

建议拆出统一的 finalize / fail path：

### 成功收尾

- 刷新 projector buffer
- 写入 `done` event
- 持久化 projection / content
- 更新 assistant message 为 `done`
- 更新 run 为 `completed` 或 `completed_with_tool_errors`

### 失败收尾

- 刷新当前已累积的可见正文
- 尽可能补 `error` event
- 尝试生成 projection；失败时允许降级
- 更新 assistant message 为 `error`
- 更新 assistant message.content 为最终正文快照
- 更新 run 为 `failed` 或 `failed_runtime`
- 记录 `error_message`
- 保存独立错误展示信息

### 降级原则

失败收尾必须按以下优先级保证数据写入：

1. assistant message 状态与内容
2. run 终态与错误信息
3. event log
4. projection / content

也就是说：

- projection 构建失败，不能影响 message/run 留痕
- event log 写入失败，也不应阻止 assistant message 从 `streaming` 收敛到 `error`

### 事务边界

建议采用“两阶段写入”而不是单一大事务：

第一阶段，关键终态事务：

- assistant message 状态与正文快照
- run 终态状态与 `error_message`

这一阶段必须放在同一事务中，避免出现：

- message 已显示失败但 run 仍是成功
- run 已失败但 message 仍是 `streaming`

第二阶段，增强数据写入：

- event log 补写
- projection / content upsert

这一阶段允许独立事务或非事务补偿执行。

原则：

- projection 写入失败不能回滚第一阶段
- projection 缺失属于增强层缺失，不属于本轮失败留痕失败

## 错误信息分层与脱敏

本次明确区分两种错误信息：

### 1. 内部错误信息

写入：

- `run.error_message`
- 服务端日志

要求：

- 允许保留更完整的诊断信息
- 但面向前端返回前必须经过脱敏转换

### 2. 用户可见错误信息

写入：

- SSE `error` payload
- assistant message 的错误展示字段或兜底文案

要求：

- 不直接暴露数据库地址、内部堆栈、SQL 语句、节点 IP 等敏感细节
- 使用统一用户文案，例如“生成中断，请稍后重试”
- 如需保留一定诊断价值，应使用脱敏后的摘要

## 前端状态设计

前端本次不引入新视觉组件，只调整状态语义。

### 当前流式会话

规则：

- 收到 SSE `error` 事件后，当前 assistant 气泡必须立即进入错误终态
- `error` 与 `done` 一样，都是本轮请求的终止信号
- 若此前已有正文，则保留正文并在独立错误区域展示错误说明
- 若此前无正文，则直接展示错误说明
- 当前轮不能在收到 `error` 后继续维持 `loading/updating`
- 若 SSE 连接异常断开且在超时阈值内未收到 `done/error`，前端也必须主动收敛为错误态

### 历史会话

规则：

- 历史消息列表优先展示 `message.content`
- `message.status = error` 时，Bubble 状态必须映射为错误态，而不是 loading
- 若存在 `run_id`，展开执行细节时再按需请求 projection
- projection 请求失败只能影响执行过程展开，不能影响主消息显示

### 重试交互

失败消息应提供明确的“重新生成”入口。

要求：

- 重试视为新一轮请求，而不是覆盖旧 run
- 前端可以带上原 session id，并生成新的 `client_request_id`
- 旧失败 run 保留用于审计与历史查看

### 断流感知

前端需要有连接异常兜底逻辑。

要求：

- 若 fetch/SSE 异常结束且未收到终态事件，前端主动触发本地错误收尾
- 本地错误文案与服务端脱敏文案保持一致
- 后续刷新历史时以后端持久化状态为准

## API 与协议影响

本次不新增新的 SSE 事件类型，但要求现有 `error` 事件承担明确终态语义。

要求：

- `error` 事件必须能让前端判定“本轮已经结束且失败”
- `error` payload 至少包含用户可展示的错误消息
- 若已有 `run_id`，建议一并返回，便于当前轮与历史 run 绑定
- 建议补充 `client_request_id`，便于前端将当前错误与本地请求关联

## 测试要求

### 后端

至少覆盖以下场景：

1. 会话开始后、执行前失败时，session/message/run 均已存在且状态正确
2. 执行中失败时，assistant message 保存部分正文和错误说明
3. 运行时失败时，run.status 与 error_message 正确
4. projection 构建失败时，message/run 仍成功收尾
5. 同一 `client_request_id` 重试不会创建重复外壳
6. 超时扫描会把长期 `running` 的 run 标记为 `expired`
7. 成功路径不回归，现有 completed / completed_with_tool_errors 仍正确

### 前端

至少覆盖以下场景：

1. 收到 `error` 事件后，当前 assistant 气泡从 loading 收敛为 error
2. 有部分正文时，正文保留，错误说明以独立区域展示
3. 历史 assistant message.status = error 时，渲染为错误态
4. 历史消息即使 projection 拉取失败，也能展示 message.content
5. SSE 未收到终态但连接中断时，前端会主动进入错误态
6. 点击重试会生成新请求而不是覆盖旧失败记录

## 风险与权衡

### 风险 1：过早创建 run 会增加失败 run 数量

这是可接受的。

原因：

- 这些 run 代表真实发生过的用户交互
- 相比“用户看到了失败但数据库没有这轮”，多记录失败 run 更符合审计与排障需求

### 风险 2：projection 与 message 可能短暂不一致

这是可接受的。

要求是明确优先级：

- 用户可见主内容以 message 为准
- projection 属于增强数据，可异步修复或降级缺失

### 风险 3：失败收尾路径复杂度上升

这是本次必须承担的复杂度。

解决方式不是继续分散分支，而是把失败收尾统一到一个明确的 finalize/fail path。

### 风险 4：幂等实现不完整会引入重复空壳

这是必须优先规避的风险。

要求：

- 幂等键从设计阶段就纳入接口契约
- 不能把“前端尽量别重试”当作解决方案

## 结论

本次设计的核心不是新增更多状态，而是建立一个简单且稳定的失败原则：

- 会话开始即创建外壳
- 外壳创建必须幂等
- 一旦失败，必须有记录
- 已生成内容必须保留
- `message/run` 是失败留痕的最低保底
- `projection` 是增强层，不能阻塞失败可见性
- 长时间未收尾的 run 必须可被回收为 `expired`

这样可以同时解决两个问题：

1. 失败会话刷新后仍能在历史里看到
2. 当前失败回复不会再长期停留在 loading 状态
