# AI 会话与 Projection 契约收敛设计

**日期**: 2026-03-20
**状态**: Draft

## 背景

`ai-session-storage-refactor` 合入后，历史会话恢复已经切到 `run projection + run content` 模型，但 assistant 最终回答仍然同时存在于两条链路：

- `session.messages[].content`
- `run projection.summary.content`

当前前端历史回放会同时消费这两份数据：

- `message.content` 被渲染到 assistant 正文区域
- `projection.summary.content` 被包装成 `runtime.summary` 再渲染一次

结果是同一份最终结论在 UI 上出现两遍。与此同时，工具执行过程也被拆成两条独立展示记录：

- `tool_call`
- `tool_result`

这会导致一步工具执行在时间线里占据两个位置，破坏步骤正文中的阅读连续性。

本次设计不做兼容性补丁，而是直接收敛 assistant 历史存储与展示契约。

## 目标

1. assistant 历史正文只保留一个事实来源
2. 消除 session 接口与 projection 接口之间的 assistant 内容冲突
3. 历史 assistant 正文统一渲染在现有 `message.content` 正文位置
4. 去掉“结论卡片 + 正文”双槽位展示
5. 一次工具执行在步骤正文里只占一个内联引用位置
6. 保持当前流式态与历史回放态最终收敛到同一种展示语义

## 非目标

1. 不兼容没有 `run_id` 的旧 assistant 历史消息
2. 不保留 assistant `content` 作为历史兜底字段
3. 不在本次设计里重做 projection 的整体事件模型
4. 不新增第二套“历史专用”渲染组件

## 问题定义

### 1. assistant 正文双写

assistant 最终回答同时写入 session message 和 projection summary，造成：

- 历史恢复时需要决定谁是真相
- 前端容易同时渲染两份文案
- 后续任何字段演进都会面对双边同步问题

### 2. 最终结论双槽位展示

当前 `AssistantReply` 既有 `runtime.summary` 卡片，也有 markdown 正文区域。只要两边承载的是同一段结论文本，就会出现重复显示。

### 3. 工具执行双记录展示

同一个工具调用被投影成 `tool_call` 和 `tool_result` 两条 activity，表现为：

- 一次工具执行占两个位置
- 步骤正文里已经有工具引用，但下方又追加结果记录
- 历史回放和流式过程都显得过于拥挤

## 设计原则

1. assistant 历史正文必须有且只有一个事实来源
2. session 负责消息骨架，run 负责 assistant 执行产物
3. projection 为历史回放提供结构化读模型
4. 最终正文和过程信息必须分层，不得重复承载同一份文本
5. 工具执行在 UI 中应表现为“单个可更新引用”，而不是两个列表项

## 目标契约

### 1. Session Message 契约

`session message` 只承担会话骨架，不再承担 assistant 最终正文存储。

规则如下：

- `user` message 继续保存并返回 `content`
- `assistant` message 不再保存正文 `content`
- `assistant` message 必须保存并返回 `run_id`
- assistant 历史恢复必须通过 `run_id -> projection`

### 2. Run Projection 契约

`run projection` 成为 assistant 历史展示的唯一正文来源。

规则如下：

- `projection.summary.content` 是 assistant 最终正文
- `projection.blocks` 承载 plan、handoff、error、executor 等过程结构
- `projection.blocks[*].items[*]` 承载步骤内联内容引用与工具引用

### 3. Run Content 契约

`run content` 保留为惰性加载通道，但不再承担 assistant 最终正文主来源。

规则如下：

- executor 长文本仍可通过 `content_id` 懒加载
- 工具参数和工具结果全文仍可通过 `content_id` 懒加载
- assistant 最终正文优先直接放在 `projection.summary.content`

## UI 展示模型

assistant 回复统一拆成三层：

1. 过程层：plan / handoff / error
2. 步骤层：步骤正文中的 text + tool_ref 顺序流
3. 正文层：最终 markdown 正文

### 最终正文展示规则

- 历史 assistant 消息的 `message.content` 由 `projection.summary.content` 填充
- `AssistantReply` 只保留一个最终正文 markdown 槽位
- 不再把完整结论文本放进 `runtime.summary`

### 内存态与历史态切换规则

流式阶段与历史阶段的底层数据源不同，但 UI 不应出现闪烁或跳变。

规则如下：

- 流式阶段允许前端继续使用内存中的 `message.content` 追加 delta
- 收到 run 完成信号后，前端不立刻清空当前正文，也不先进入错误态
- 前端应静默拉取或重取 `run projection`
- projection 可用后，用 `projection.summary.content` 无闪烁接管正文数据源
- 接管只替换底层数据，不改变当前消息气泡位置与滚动锚点

如果 projection 在短时间窗口内尚未可读，前端应保持流式结束时的内存态正文，而不是立刻显示“Projection 缺失”错误。

### Summary 区域规则

`runtime.summary` 不再承载完整结论正文。

允许保留的内容类型：

- 工具数
- 风险级别
- 诊断对象数量
- 其他短字段摘要

不允许再放：

- 完整结论段落
- 与最终 markdown 正文重复的长文本

## 工具展示模型

### 目标语义

一次工具执行在 UI 上表现为一个内联工具引用。

这个引用会随着执行过程更新状态：

- 执行中
- 已完成
- 失败

但它始终是同一个对象、同一个位置、同一个 `call_id`。

### 投影规则

projection 到前端 runtime 的映射不再生成两条独立 activity：

- 不再为同一次工具执行分别追加 `tool_call` 和 `tool_result` 两条展示记录
- 步骤正文继续通过 `segments: text | tool_ref` 表达顺序
- `tool_ref` 指向单个工具实体，工具实体内部同时包含：
  - `tool_name`
  - `arguments`
  - `status`
  - `preview`
  - `rawContent`

### 渲染规则

- 步骤正文按 `segments` 顺序渲染
- 遇到 `tool_ref` 时渲染一个工具引用组件
- 工具完成后在原位置更新状态并允许展开查看参数与结果
- 不再在步骤下方额外追加 `tool_result` 行

### 工具异常兜底状态

工具引用不能永久停留在“执行中”状态。

如果只看到 `tool_call`，但没有对应的完成结果，应根据上下文解析为明确的终态：

- 当前流式会话中超过超时阈值且 run 已结束：显示“未完成”
- 历史 projection 中存在孤立工具引用且 run 状态不是 streaming：显示“异常中断”
- run 整体失败时：工具引用显示失败态，而不是继续 loading

这类状态允许展开查看已知参数，但结果区域应显示“无结果”或错误提示。

## 组件与模块调整

### 后端

#### Session 接口

需要收紧 assistant message 返回结构：

- assistant 消息移除 `content`
- assistant 消息保留 `run_id`

#### Projection 接口

保持 `summary.content` 作为最终正文来源。

#### Content 接口

继续提供惰性内容查询，不承担最终正文主来源。

### 前端

#### `historyProjection.ts`

需要调整为：

- assistant 历史 message 的 `content` 直接取 `projection.summary.content`
- `runtime.summary` 不再塞入 `{ label: "结论", value: summary.content }`
- tool 映射不再生成两条独立 activity 占位

#### `AssistantReply.tsx`

需要调整为：

- 删除“完整结论卡片”展示路径
- 保留单一最终 markdown 正文区域
- 工具只通过步骤正文中的 `ToolReference` 内联展示
- 不再把同一步的 `tool_call` / `tool_result` 额外挂在正文下方

#### `replyRuntime.ts`

需要调整为：

- `applyToolCall` 创建或激活同一个工具引用对象
- `applyToolResult` 更新同一个工具引用对象
- 同一个 `call_id` 不再衍生第二个独立展示条目

## 数据流

### 历史 assistant 恢复

1. session 接口返回 assistant message 外壳与 `run_id`
2. 前端根据 `run_id` 拉取 projection
3. `projection.summary.content` 填充到 assistant message 的正文位
4. `projection.blocks` 转换成 plan / segments / tool references
5. `AssistantReply` 渲染过程层与最终正文

### 当前流式态到历史态收敛

1. 流式阶段前端仍可在内存中使用 `message.content` 追加 delta
2. run 完成后，持久化结果以 projection 为准
3. 前端在收到完成信号后静默拉取 projection，并以其接管最终正文来源
4. projection 未可用的短窗口内，继续保留内存态正文，不立即切换错误态
5. 历史重载时不再依赖 session 保存的 assistant 正文

## 错误处理

### Projection 缺失

如果 assistant message 有 `run_id` 但 projection 缺失，需要区分“短暂未就绪”和“真实缺失”。

处理原则：

- 当前会话刚结束后的短窗口内：保持内存态正文，重试拉取 projection
- 超过重试窗口仍不可用：显示明确错误态
- 历史重载首次拉取即缺失：允许后端基于 event log 重建 projection；若仍失败，再显示错误态
- 不回退到 session assistant `content`

### Summary 缺失

如果 projection 存在但 `summary.content` 缺失，应视为投影不完整。

处理原则：

- UI 显示“回答内容不可恢复”之类的错误提示
- 不回退到 session assistant `content`

### Tool 内容缺失

如果工具引用存在但结果全文懒加载失败：

- 保留工具引用本身
- 展示 preview 或错误态
- 不影响最终正文显示

## 发布与切换策略

这是一次前后端契约切换，不能假设双端完全同时发布。

发布原则如下：

1. 后端先发布，确保 session / projection / content 接口同时满足新契约
2. 前端后发布，再切换到“assistant 正文仅来自 projection”的读取逻辑
3. 发布窗口内允许保留极短期上线保护逻辑，但该逻辑只用于发版切换，不作为长期兼容方案

如果选择保留上线保护逻辑，限制如下：

- 优先读取 projection
- 仅在发布窗口内且 projection 暂不可用时，才允许短暂保留内存态正文
- 不新增面向旧历史数据的长期 `session.content` 回退契约

## 后端一致性要求

当前实现虽然在读取 projection 时支持基于 event log 重建，但设计上仍应尽量缩短“run 完成”与“projection 可读”之间的窗口。

要求如下：

- run 完成后的 projection 持久化应作为结束链路的一部分，而不是异步旁路任务
- `GetRunProjection` 必须保留基于 event log 的重建能力，避免短暂落库延迟直接暴露给前端
- 前端应把 projection 缺失视为“待重试或需重建”的信号，而不是第一时间渲染最终失败态

## 测试策略

### 后端

1. session 接口对 assistant message 不再返回 `content`
2. session 接口对 assistant message 返回 `run_id`
3. projection 接口返回 `summary.content`
4. content 接口继续返回 executor / tool 的惰性正文

### 前端

1. 历史 hydrate 仅使用 `projection.summary.content` 作为 assistant 正文
2. 有 projection summary 时，最终正文只渲染一次
3. `runtime.summary` 不再重复包含完整结论文本
4. 单个 tool 引用在步骤中只占一个位置
5. `applyToolCall` 与 `applyToolResult` 会收敛到同一个工具引用对象
6. 流式结束后切换到 projection 不得引起正文闪烁或滚动锚点跳动
7. 孤立 tool 引用会落到“未完成”或“异常中断”态，而不是永久 loading

## 实施步骤

1. 收紧后端 session message 输出契约，移除 assistant `content`
2. 确认 run 完成链路中 projection 持久化与 `GetRunProjection` 重建兜底都可用
3. 调整历史 hydrate 逻辑，正文只取 `projection.summary.content`
4. 删除 `runtime.summary` 中对完整结论文本的组装
5. 调整 `AssistantReply`，移除结论卡片型重复展示
6. 收敛 tool 引用模型，取消 `tool_call` / `tool_result` 双占位，并补齐异常终态
7. 增加流式结束到 projection 接管的平滑切换逻辑
8. 按“后端先、前端后”的顺序发布
9. 更新后端与前端测试覆盖新契约与切换窗口

## 风险与权衡

### 风险

- 如果 projection 生成链路不稳定，assistant 历史将直接不可读
- 前后端要同时切换契约，短期内容易出现字段对不上
- tool 引用从双记录改为单引用后，部分测试需要整体重写

### 权衡

本设计刻意放弃旧 assistant `content` 兼容路径，换取更干净的单一事实来源。考虑到旧历史已删除，这个取舍是合理的。

## 验收标准

满足以下条件即可视为完成：

1. 历史 assistant 回复不再出现重复结论
2. assistant 历史正文仅来源于 `projection.summary.content`
3. session assistant message 不再返回正文 `content`
4. 工具执行在步骤正文中只显示一个内联引用位置
5. 工具完成后在原位置更新，而不是新增第二条结果记录
6. 流式结束切换到 projection 时，UI 不出现明显闪烁或滚动跳动
7. projection 短暂未就绪时，前端不会立刻进入误报错误态
8. 孤立工具引用会显示明确终态，不会永久转圈
