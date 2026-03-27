# AI 会话存储重构设计

**日期**: 2026-03-20
**状态**: Draft

## 背景

当前 AI 会话存储采用混合模型：

- `ai_chat_messages.content` 同时承担最终回答文本和历史展示兜底内容
- `ai_chat_messages.runtime_json` 存放 assistant 当前态投影结果
- 前端当前会话依赖 SSE 在内存中持续构建 runtime
- 前端历史会话依赖 message 列表加 `GetMessageRuntime` 懒加载恢复 runtime

这套模型已经出现结构性问题：

- 当前会话和历史会话不是同一种数据来源
- `steps`、`summary`、`tool_call`、`tool_result` 的顺序关系会在历史恢复时丢失
- `runtime_json` 混合了运行时状态、展示结构和局部内容快照，职责不清
- 一旦投影逻辑出错，缺少稳定的原始事实来源用于回放和修复

本次不是兼容性补丁，而是一次完整重构。目标是先清理旧存储方案，再引入新的三层模型。

## 目标

### 主目标

1. 建立单一事实来源，保证事件一致性与可回放
2. 建立稳定的历史渲染结构，确保当前态与历史态读取模型统一
3. 支持长内容懒加载，控制会话首屏载荷
4. 支持按 `session`、`run`、`event_type`、`tool_call_id`、`agent` 等维度查询
5. 明确淘汰旧方案，避免继续在旧模型上叠补丁

### 非目标

1. 不对旧脏数据做强兼容展示优化
2. 不保留 `message + runtime_json` 的双轨读取模式
3. 不继续扩展 `AssistantReplyRuntime` 作为后端持久化协议

## 旧方案清理范围

本次重构要明确删除的是一整套旧思路，而不是局部字段替换。

### 1. 废弃 `AIChatMessage` 作为 assistant 历史渲染真相来源

旧模型中 assistant message 同时承担：

- 对话消息记录
- 结构化运行时展示
- 历史视图兜底文本

这三种职责必须拆开。重构后 `ai_chat_messages` 只保留消息外壳和基础元信息，不再承载历史结构化展示真相。

### 2. 废弃 `ai_chat_messages.runtime_json`

当前 `runtime_json` 混合存放：

- `plan`
- `activities`
- `summary`
- `segments`
- `status`
- 当前投影态快照

这个字段模型整体废弃。重构后不再写入、不再读取、不再演进。

### 3. 删除基于 message 的 runtime 恢复链路

需要删除：

- `GetMessageRuntime`
- 历史消息按 `message_id` 懒加载 runtime 的接口
- 前端 `historyRuntime.ts` 的恢复逻辑与缓存逻辑
- 所有从 `message.content + runtime_json` 反推块结构和工具位置的逻辑

### 4. 停止把前端内存 runtime 当作持久化协议

`replyRuntime.ts` 可继续用于前端当前流式渲染，但它不再代表后端存储协议。

需要明确分离：

- 前端内存态：为流式更新服务
- 后端持久化模型：为事件落库和历史读取服务

### 5. 删除历史兼容性补丁

需要清理所有“为了补救旧模型缺陷”的代码路径，例如：

- 用 `content` 兜底 step 展示
- 从 activities 反推 tool 插入位置
- 历史态与当前态分别走不同渲染协议
- 历史态对 tool/result 做平铺补偿

## 新方案总览

新方案采用三层模型：

1. `event log`
2. `runtime projection`
3. `content store`

职责分离如下：

- `event log` 负责记录原始事件顺序，是唯一事实来源
- `runtime projection` 负责给前端提供结构化渲染索引
- `content store` 负责存放长文本、大 JSON 和懒加载内容

### 设计原则

1. 所有历史展示结构都必须能追溯到事件日志
2. 历史默认读取 projection，不默认重放全部事件
3. 长内容不内嵌到 projection 中，结论类内容可直接内联
4. 一个 `run` 是一次执行和一次投影的主边界
5. `executor` 的文本与工具调用顺序必须被显式建模，不能只合并成单个字符串
6. projection 不是唯一读取前提，缺失或不完整时必须允许从 event log 现场补偿重建

## 新数据模型

### 会话层级

- `session`: 会话容器
- `run`: 一次用户输入触发的一次 AI 执行
- `event`: run 内原始事件
- `projection`: run 的结构化展示索引
- `content`: projection 引用的大内容

### 真相来源

单一事实来源是 `ai_run_events`。

`projection_json` 不是原始事实，只是从事件日志生成的读模型。

`content store` 也不是真相来源，它只是 projection 的内容承载层。

## 表结构设计

### 1. `ai_sessions`

保留现有表，继续作为会话容器。

建议保留字段：

- `id`
- `user_id`
- `scene`
- `title`
- `created_at`
- `updated_at`
- `deleted_at`

### 2. `ai_runs`

保留现有表，但职责明确为一次执行边界。

建议保留或增强字段：

- `id`
- `session_id`
- `user_message_id`
- `assistant_message_id`
- `status`
- `assistant_type`
- `intent_type`
- `progress_summary`
- `trace_id`
- `error_message`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`

### 3. 新增 `ai_run_events`

用于记录一次 run 的完整事件流。

建议字段：

- `id` `varchar(64)` 主键
- `run_id` `varchar(64)` 非空
- `session_id` `varchar(64)` 非空
- `seq` `int` 非空，run 内顺序号
- `event_type` `varchar(32)` 非空
- `agent_name` `varchar(64)` 可空
- `tool_call_id` `varchar(64)` 可空
- `payload_json` `longtext` 非空
- `created_at` `datetime` 非空

建议索引：

- `uk_ai_run_events_run_seq (run_id, seq)` 唯一
- `idx_ai_run_events_session_created (session_id, created_at)`
- `idx_ai_run_events_tool_call_id (tool_call_id)`
- `idx_ai_run_events_run_type (run_id, event_type)`

### 4. 新增 `ai_run_projections`

用于记录 run 对应的结构化展示索引。

建议字段：

- `id` `varchar(64)` 主键
- `run_id` `varchar(64)` 非空且唯一
- `session_id` `varchar(64)` 非空
- `version` `int` 非空
- `status` `varchar(32)` 非空
- `projection_json` `longtext` 非空
- `created_at` `datetime` 非空
- `updated_at` `datetime` 非空

建议索引：

- `uk_ai_run_projections_run_id (run_id)` 唯一
- `idx_ai_run_projections_session_id (session_id)`

### 5. 新增 `ai_run_contents`

用于承载 projection 引用的大内容。

建议字段：

- `id` `varchar(64)` 主键
- `run_id` `varchar(64)` 非空
- `session_id` `varchar(64)` 非空
- `content_kind` `varchar(32)` 非空
- `encoding` `varchar(16)` 非空，建议取值 `text` 或 `json`
- `summary_text` `varchar(500)` 可空
- `body_text` `longtext` 可空
- `body_json` `longtext` 可空
- `size_bytes` `bigint` 非空默认 0
- `created_at` `datetime` 非空

建议索引：

- `idx_ai_run_contents_run_id (run_id)`
- `idx_ai_run_contents_session_id (session_id)`
- `idx_ai_run_contents_kind (content_kind)`

### 6. `ai_chat_messages`

保留但降级。

建议最终角色：

- 记录用户消息和 assistant 外层消息
- 提供基础对话顺序
- assistant `content` 仅保留最终回答摘要或直接展示文本

需要移除字段：

- `runtime_json`

不再允许把结构化历史展示写入该表。

## Projection 结构设计

### 核心思想

`projection_json` 不保存完整大文本，只保存结构、顺序、标题和内容引用。

顶层组织单位为 `blocks`。

推荐块类型：

- `agent_handoff`
- `plan`
- `executor`
- `summary`
- `error`

其中 `executor` 块内部再组织 `items`。

### `executor` 块设计

`executor` 内部的 `items` 是有序数组。

允许的 `item.type`：

- `content`
- `tool_call`

其中：

- `content` 用于承载一段连续合并后的 `delta`
- `tool_call` 表示一次工具调用
- `tool_result` 不再作为顶层 block，也不单独作为平级 item，而是挂在 `tool_call.result`

### 示例 `projection_json`

```json
{
  "version": 1,
  "run_id": "15e1a091-e9c1-4321-8c90-73997453afd5",
  "session_id": "6aa6b90c-5ec5-485c-9b0c-c7e76b503e22",
  "turn": 1,
  "status": "completed",
  "summary": {
    "title": "结论",
    "content_mode": "inline",
    "content": "先定位火山云主机，再查询内存总量与可用量。"
  },
  "blocks": [
    {
      "id": "block_handoff_1",
      "type": "agent_handoff",
      "title": "任务转交",
      "event_ids": ["evt_003"],
      "data": {
        "from": "OpsPilotAgent",
        "to": "DiagnosisAgent",
        "intent": "diagnosis"
      }
    },
    {
      "id": "block_plan_1",
      "type": "plan",
      "title": "处理计划",
      "event_ids": ["evt_004"],
      "steps": [
        "定位主机",
        "查询内存",
        "输出结果"
      ]
    },
    {
      "id": "block_executor_1",
      "type": "executor",
      "title": "定位火山云主机",
      "agent": "executor",
      "start_event_id": "evt_005",
      "end_event_id": "evt_012",
      "lazy": true,
      "items": [
        {
          "id": "item_1",
          "type": "content",
          "content_id": "cnt_exec_1",
          "start_event_id": "evt_005",
          "end_event_id": "evt_007"
        },
        {
          "id": "item_2",
          "type": "tool_call",
          "tool_call_id": "call_a67bae8347e842409fc2378f",
          "tool_name": "host_list_inventory",
          "event_id": "evt_008",
          "arguments_content_id": "cnt_args_1",
          "result": {
            "event_id": "evt_009",
            "status": "done",
            "preview": "{\"total\":0,\"list\":[]}",
            "result_content_id": "cnt_result_1"
          }
        },
        {
          "id": "item_3",
          "type": "content",
          "content_id": "cnt_exec_2",
          "start_event_id": "evt_010",
          "end_event_id": "evt_012"
        }
      ]
    }
  ]
}
```

## 内容存储设计

### 内容类型

建议至少支持：

- `executor_content`
- `tool_arguments`
- `tool_result`
- `summary_content`

### 存储规则

1. `content` item 的长文本写入 `ai_run_contents`
2. `tool_call.arguments` 写入 `ai_run_contents`
3. `tool_result` 原始结果写入 `ai_run_contents`
4. `summary` 默认直接内联在 `projection_json` 中
5. 如果后续 summary 变大，再迁移为 `content_id` 模式

## 事件写入设计

### 写入原则

每次 SSE 事件到达时，后端先写 `ai_run_events`，再更新内存中的投影状态。

执行结束时生成：

- `ai_run_projections`
- `ai_run_contents`
- assistant message 简要内容

### 并发与幂等要求

`ai_run_events.seq` 必须是同一个 `run_id` 内严格递增且不重复的稳定顺序号。

本设计要求：

1. `seq` 由事件写入侧统一分配，不依赖前端生成
2. 同一 `run_id` 下使用单写入源顺序追加事件
3. `uk_ai_run_events_run_seq (run_id, seq)` 作为最终一致性兜底
4. 如果同一事件发生重复写入，必须能够通过 `run_id + seq` 或额外的幂等键安全拒绝重复落库
5. projection 构建只能依赖已持久化的事件顺序，不允许依赖到达时的内存顺序假设

### Projection 生成时机

主路径仍然是在 run 结束时生成并持久化 `ai_run_projections`。

但必须承认以下失败场景：

- 进程崩溃
- 宿主机宕机
- OOM
- run 被异常中断，导致 projection 尚未写入

因此 projection 不是“只会在结束时生成一次”的静态产物，而是“优先在结束时生成，必要时可从事件日志重建”的可补偿读模型。

## Projection 补偿与重建机制

### 触发条件

读取 run projection 时，出现以下任一情况，都必须触发补偿逻辑：

1. `ai_run_projections` 不存在
2. projection 状态不是稳态
3. projection 版本落后于当前协议版本
4. projection 内容损坏或 JSON 解析失败

### 读取降级策略

新读取链路必须支持：

1. 先读取 `ai_run_projections`
2. 如果 projection 可用，则直接返回
3. 如果 projection 不可用，则从 `ai_run_events` 拉取该 run 全量事件
4. 在内存中重建 projection
5. 将重建结果返回前端
6. 异步回写新的 `ai_run_projections`

这条链路是新模型的必备能力，不是可选优化。

### 稳态定义

建议将以下状态视为稳态：

- `completed`
- `completed_with_tool_errors`
- `failed_runtime`
- `interrupted`

非稳态 projection 不应被视为可信最终结果。

### 事件类型映射

原始事件最少保留：

- `meta`
- `tool_call`
- `agent_handoff`
- `plan`
- `replan`
- `delta`
- `tool_result`
- `done`
- `error`

不允许为了 projection 简化而丢弃原始事件。

## `executor` 切块规则

这是本方案的核心规则，必须作为稳定协议固定下来。

### 切块规则

1. 连续的 `delta(agent=executor)` 合并为一个 `content` item
2. 遇到 `tool_call` 时，结束当前 `content` item
3. 生成一个新的 `tool_call` item，挂到当前 `executor` block 下
4. 同一 `tool_call_id` 的 `tool_result` 写入该 `tool_call.result`
5. `tool_call` 结束后，新的连续 `delta` 开始下一个 `content` item
6. 遇到 `plan`、`replan`、`agent_handoff`、`summary`、`done`、`error` 或 agent 切换时，结束当前 `executor` block
7. `executor` block 内部严格保持 item 顺序

### 不采用的做法

1. 不把所有 `delta` 合并成一个大 `content`
2. 不把 `tool_call`、`tool_result` 平铺到 executor 外层
3. 不通过 activities 反推 tool 插入位置

## 读取链路设计

### 历史会话读取

历史会话读取链路改为：

1. 获取 session 列表
2. 获取 session 下的 run 摘要
3. 获取 run projection
4. 如果 projection 缺失或不完整，则即时从 `ai_run_events` 重建 projection
5. 首屏直接渲染 projection 中的 `summary`、`plan`、`executor item` 索引
6. 遇到 `content_id` 时，按需请求内容接口

### 当前会话读取

当前流式会话仍可在前端维护内存态，但最终展示结构要尽量靠近 projection 协议。

目标是：

- 当前态和历史态共享同一套块模型
- 只是数据来源不同

### 旧数据前端兜底策略

本次重构不对旧脏数据做强兼容，但用户侧必须有基础可读兜底。

如果读取到旧数据且满足以下任一情况：

- 没有 event log
- 没有 projection
- 无法可靠恢复结构化块

前端必须退化为最基础的纯文本模式：

1. 只渲染 assistant 外层 `content`
2. 隐藏 `steps` UI
3. 隐藏 `tools` UI
4. 隐藏结构化 runtime 组件
5. 保证页面不白屏、不抛错，至少可读结论

### 需要删除的旧读取接口

- `GET /session/message/:id/runtime`

需要新增的新接口类型：

- `GET /sessions/:id/runs`
- `GET /runs/:id/projection`
- `GET /run-contents/:id`

## 迁移策略

本次迁移按“先停旧、后上新、再删旧”的顺序执行。

### Phase 1: 冻结旧模型

1. 停止对 `AIChatMessage.RuntimeJSON` 添加任何新逻辑
2. 停止新增基于 message runtime 的任何接口或前端逻辑
3. 将现有 `runtime_json` 标记为待删除

### Phase 2: 引入新表

1. 创建 `ai_run_events`
2. 创建 `ai_run_projections`
3. 创建 `ai_run_contents`
4. 在 `ai_runs` 和 `ai_chat_messages` 上补充必要外键和索引

### Phase 3: 新写入链路上线

1. SSE 事件消费过程中写入 `ai_run_events`
2. run 完成时生成 `projection_json`
3. 长内容写入 `ai_run_contents`
4. assistant message 只写简要文本，不写 runtime

### Phase 4: 新读取链路上线

1. 前端历史会话改读 run projection
2. 前端内容详情改读 `ai_run_contents`
3. 删除 message runtime 的依赖

### Phase 5: 删除旧方案

1. 删除 `GetMessageRuntime`
2. 删除 `historyRuntime.ts` 及相关测试
3. 删除 `AIChatMessage.RuntimeJSON` 写入逻辑
4. 删除 `AIChatMessage.RuntimeJSON` 字段和对应 migration/测试
5. 删除所有 message 级 runtime 恢复逻辑

## 旧数据处理策略

本次重构不以兼容旧脏数据为前提。

建议策略：

1. 新数据严格走新模型
2. 旧数据不做复杂自动迁移
3. 如果必须迁移旧数据，只允许 best-effort 迁移，不允许为了旧数据回填继续污染新协议

原因：

- 旧 `runtime_json` 本身已经不可靠
- 继续兼容旧脏结构会把新系统再次拖回双轨状态

## 数据膨胀与生命周期管理

### 事件增长风险

`ai_run_events` 会记录高频 `delta`、plan、tool_call、tool_result 等事件，天然会快速增长。

这是事件日志模型的成本，必须提前纳入设计，而不能等单表膨胀后再补救。

### 存储治理要求

1. `ai_run_events` 和 `ai_run_contents` 必须预留按时间归档或分区的能力
2. 需要定义明确的数据保留周期
3. 冷数据应允许下沉到归档存储，而不是永久保留在热表

### 建议策略

建议采用两层策略：

1. 热数据保留完整 `event log + projection + contents`
2. 超过设定周期的冷数据，允许只保留精简版 projection 和外层 message，清理原始 events 与大内容

文档当前不强绑定具体数据库分区实现，但要求后续实现计划中明确：

- 是否采用按月分表或分区
- 归档保留周期
- 冷热数据切换规则
- 清理任务的执行方式

## 搜索能力边界

重构后，会话文本分散在：

- `ai_chat_messages.content`
- `ai_run_projections.projection_json`
- `ai_run_contents`

因此必须明确搜索边界，避免后续继续依赖跨表 `LIKE` 模糊查询。

### 当前要求

1. 不把跨表全文检索作为本次重构的默认内建能力
2. 默认只保证外层 message 摘要和 projection 元信息可被基础查询使用
3. 如果业务需要完整聊天记录全文检索，应通过专门的搜索索引实现，而不是直接依赖关系库跨表扫描

### 后续扩展方向

如需完整搜索，建议单独建设搜索索引层，将以下内容统一投递：

- message 摘要
- projection 标题与步骤
- executor 文本内容
- tool_result 可搜索摘要

## 风险与权衡

### 风险

1. 写路径复杂度上升
2. run 完成时需要额外生成 projection
3. 前后端接口要一起切换

### 权衡

这些复杂度是有价值的，因为它们换来：

1. 可回放
2. 可审计
3. 历史展示稳定
4. 查询能力更强
5. 后续扩展 `replan`、多 agent、审批流时边界更清晰

## 结论

本次 AI 会话存储重构要明确执行以下决策：

1. 废弃 `message.content + runtime_json` 作为历史展示真相来源
2. 以 `ai_run_events` 作为唯一事实来源
3. 以 `ai_run_projections.projection_json` 作为默认历史读取模型
4. 以 `ai_run_contents` 作为长内容与懒加载承载层
5. `executor` 文本按 `tool_call` 切块，并将工具调用与结果挂到对应 `executor` 块下
6. 不为旧脏数据延续兼容补丁路径

这是一次完整替换，不是旧模型的延长线。
