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

## 生命周期设计

一次聊天的服务端生命周期调整为以下顺序：

1. 确保 `session` 存在
2. 创建 `user message`
3. 创建 `assistant message` 占位，初始状态为 `streaming`
4. 创建 `run`，初始状态为 `running`
5. 发送 `meta` 并进入 AI router / runner 执行
6. 按事件流持续积累已生成正文和结构化事件
7. 在成功或失败时统一进入收尾逻辑

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

### 3. `ai_run_projections`

这是结构化增强层，不是失败留痕的前提条件。

要求：

- 能写 projection 时写完整 projection
- 只能写最小 error projection 时允许降级
- projection 写入失败不能回滚已存在的 message/run 失败记录

## 失败分类

本次只保留两类失败语义，避免过度细分。

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

## 内容持久化规则

### 最终回复快照

assistant message 的 `content` 必须始终代表用户最终能看到的完整文本快照。

规则如下：

1. 成功完成时，保存最终正文
2. 执行中失败时，保存“已生成正文 + 错误说明”
3. 启动前失败且无正文时，直接保存错误说明

### 错误说明拼接策略

建议采用稳定拼接形式：

- 若已有正文：`正文 + 分隔符 + 错误说明`
- 若无正文：`错误说明`

目标不是追求精致文案，而是保证：

- 用户能看到已成功生成的部分
- 用户能明确知道本轮最终失败
- 历史回复不会因为错误覆盖掉已生成内容

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
- 更新 assistant message.content 为最终失败快照
- 更新 run 为 `failed` 或 `failed_runtime`
- 记录 `error_message`

### 降级原则

失败收尾必须按以下优先级保证数据写入：

1. assistant message 状态与内容
2. run 终态与错误信息
3. event log
4. projection / content

也就是说：

- projection 构建失败，不能影响 message/run 留痕
- event log 写入失败，也不应阻止 assistant message 从 `streaming` 收敛到 `error`

## 前端状态设计

前端本次不引入新视觉组件，只调整状态语义。

### 当前流式会话

规则：

- 收到 SSE `error` 事件后，当前 assistant 气泡必须立即进入错误终态
- `error` 与 `done` 一样，都是本轮请求的终止信号
- 若此前已有内容，则保留内容并附加错误说明
- 若此前无内容，则直接展示错误文本
- 当前轮不能在收到 `error` 后继续维持 `loading/updating`

### 历史会话

规则：

- 历史消息列表优先展示 `message.content`
- `message.status = error` 时，Bubble 状态必须映射为错误态，而不是 loading
- 若存在 `run_id`，展开执行细节时再按需请求 projection
- projection 请求失败只能影响执行过程展开，不能影响主消息显示

## API 与协议影响

本次不新增新的 SSE 事件类型，但要求现有 `error` 事件承担明确终态语义。

要求：

- `error` 事件必须能让前端判定“本轮已经结束且失败”
- `error` payload 至少包含用户可展示的错误消息
- 若已有 `run_id`，建议一并返回，便于当前轮与历史 run 绑定

## 测试要求

### 后端

至少覆盖以下场景：

1. 会话开始后、执行前失败时，session/message/run 均已存在且状态正确
2. 执行中失败时，assistant message 保存部分正文和错误说明
3. 运行时失败时，run.status 与 error_message 正确
4. projection 构建失败时，message/run 仍成功收尾
5. 成功路径不回归，现有 completed / completed_with_tool_errors 仍正确

### 前端

至少覆盖以下场景：

1. 收到 `error` 事件后，当前 assistant 气泡从 loading 收敛为 error
2. 有部分正文时，错误说明会附加而非覆盖正文
3. 历史 assistant message.status = error 时，渲染为错误态
4. 历史消息即使 projection 拉取失败，也能展示 message.content

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

## 结论

本次设计的核心不是新增更多状态，而是建立一个简单且稳定的失败原则：

- 会话开始即创建外壳
- 一旦失败，必须有记录
- 已生成内容必须保留
- `message/run` 是失败留痕的最低保底
- `projection` 是增强层，不能阻塞失败可见性

这样可以同时解决两个问题：

1. 失败会话刷新后仍能在历史里看到
2. 当前失败回复不会再长期停留在 loading 状态
