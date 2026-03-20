# AI 会话体验优化设计

**日期**: 2026-03-20
**状态**: Draft

## 背景

上一轮 AI 会话存储重构已经把历史读取拆分为 `session`、`projection`、`content` 三层，但当前前端会话体验仍然存在明显问题：

- AI 抽屉尚未打开时，就会开始请求 session 内容
- 打开抽屉后会默认拉取首个 session 的详情，其他 session 切换后长期停留在 loading 状态
- 历史 assistant 回复会在首屏直接展开完整 steps，信息密度过高
- 当前滚动跟踪逻辑不稳定，用户拉到底部后仍可能被跳到更早的位置

这些问题不是数据结构问题，而是请求触发时机、前端状态归属和滚动所有权划分不清导致的体验退化。

## 目标

### 主目标

1. 抽屉未打开时不发起 AI session 相关请求
2. 抽屉打开后只请求 session 列表和最近更新的 session 详情，不预取其他 session
3. 修复多 session 场景下除首个 session 外长期 loading 的问题
4. 历史 assistant 回复默认只展示结果摘要与 step 标题，step 内容按需展开
5. 当前生成中的回复继续保持实时展开，不受历史懒加载策略影响
6. 自动滚动严格遵守“贴底时跟随，上滑后停止，再到底部时恢复”的交互

### 非目标

1. 不在本次设计中引入新的抽屉布局或视觉风格改版
2. 不在本次设计中改造当前流式回复的数据协议
3. 不在本次设计中新增真正的 step 级后端读取接口
4. 不在本次设计中持久化用户上次浏览的 session

## 范围

### In Scope

- `CopilotSurface` 的 session 列表加载、session 详情加载和滚动跟踪逻辑
- 历史 assistant 回复的 runtime 懒加载和 step 二级折叠
- 相关前端缓存粒度和错误展示策略
- 必要的测试调整与补充

### Out of Scope

- 新的后端 step-detail API
- session 排序策略变更以外的后端查询优化
- 历史消息内容格式重构
- AI 抽屉整体 UI 重绘

## 产品决策

本次选择“前端显式分层，后端接口保持不变”的方向。

原因如下：

- 当前问题核心在前端状态边界不清，不需要先扩展后端协议
- 现有接口已经足以支持“打开抽屉加载最近 session”与“历史 runtime 按需展开”
- 先修正加载和滚动行为，可以最小代价恢复可用体验
- 如果未来确实需要 step 级网络懒加载，本次拆分出的状态边界可以直接承接

## 现状问题

### 1. 会话预加载过早

当前实现会在抽屉尚未真正使用时就触发 scene 级 session 请求，导致：

- 首屏多余请求增加
- 页面进入时就产生 AI 相关网络噪音
- 用户尚未打开 AI 助手时就消耗 session 加载成本

### 2. session 详情加载边界错误

当前 `activeConversationKey` 与 `defaultMessages` 的隐式耦合让消息详情请求由内部机制驱动，导致：

- 首个 session 会被默认拉取
- 其他 session 切换时 loading 与缓存无法按 session key 独立收敛
- session 详情状态难以单独追踪、重试和回退

### 3. 历史步骤首屏噪音过大

历史 assistant 回复在首屏直接渲染完整步骤内容，带来两个问题：

- 会话列表切换后，历史消息内容块过长，阅读密度过高
- 用户通常先想确认“这条回复做了哪些步骤”，而不是一开始就看全部细节

### 4. 自动滚动会抢夺用户控制权

当前滚动行为将“初次定位到底部”“切换 session 后定位”“流式回复跟随”“用户手动滚动”混在若干 `effect` 中，导致：

- 用户已经上滑时仍可能被新的渲染更新带跑
- 流式输出和历史内容补全共享同一套滚动触发条件
- 在某些时刻会跳到过早的消息位置，而不是稳定贴底

## 方案概览

本次设计把 AI 抽屉拆成三条独立链路：

1. session 列表链路
2. session 详情链路
3. 历史 runtime 链路

三条链路分别有自己的加载、缓存和错误状态，避免再依赖 `useXChat` 的隐式 `defaultMessages` 详情拉取。

## 设计

### Section 1: 数据加载与状态边界

#### 1.1 Session 列表状态

引入 `sessionListState`，只负责当前 scene 的会话摘要列表。

职责：

- 仅在抽屉 `open = true` 时触发加载
- 调用 `getSessions(scene)`
- 返回按更新时间排序的 session 摘要列表
- 在列表为空时提供 `New chat`

不负责：

- 任意 session 的消息详情
- 历史 runtime 或 projection
- 当前流式消息状态

#### 1.2 Session 详情状态

引入 `sessionDetailStateById`，按 session id 管理消息详情状态。

每个 session 详情项包含：

- `status: idle | loading | ready | error`
- `messages`
- `error`
- `loadedAt`
- `requestId`
- `abortController`

职责：

- 打开抽屉后只加载最近更新的 session 详情
- 用户切换到某个 session 时，仅在该 session 尚未 ready 时请求详情
- 已 ready 的 session 直接复用缓存
- 某个 session 加载失败时，只影响该 session，不污染全局

并发与竞态控制要求：

- 当用户快速切换 session 时，新请求发起前应主动 abort 掉上一个仍处于 loading 的非当前详情请求
- 每次 session 详情请求都应绑定唯一 `requestId`
- 即使旧请求未能及时 abort，返回结果也必须先校验 `requestId`，再决定是否写入状态
- 任何过期响应都不得覆盖当前激活 session 或更新较新的缓存结果

这样可以确保：

- 只有当前激活 session 会进入 loading
- session A 加载失败不会让 session B 卡住
- 不会再预取未点击的其他 session

#### 1.3 当前激活 session 规则

本次不做“记住上次打开位置”的持久化逻辑。

抽屉打开后的规则固定为：

1. 请求当前 scene 的 session 列表
2. 若列表非空，默认激活最近更新的 session
3. 立即请求该 session 的详情
4. 若列表为空，则默认停留在 `New chat`

#### 1.4 当前流式消息边界

当前流式对话继续由 `useXChat` 与 `PlatformChatProvider` 驱动，但它不再负责“历史 session 详情拉取”。

边界调整为：

- `useXChat` 只负责当前激活会话内的新消息发送与流式响应
- 历史 session 的消息列表由显式的 session detail 状态提供
- 当前生成完成后，再把结果合并回当前 session detail 缓存

这意味着 [CopilotSurface.tsx](/root/project/k8s-manage/web/src/components/AI/CopilotSurface.tsx) 中基于 `defaultMessages` 的隐式详情加载链路应被移除，改成由外层显式提供 active session 的消息数据。

### Section 2: 历史消息 Steps 二级折叠

#### 2.1 只作用于历史 assistant 消息

本次折叠策略只作用于历史 assistant 消息。

当前正在生成中的 assistant 回复仍然保持：

- 当前 active step 实时展开
- 工具调用与文本实时更新
- 不引入额外折叠层阻断阅读

#### 2.2 两层展示结构

历史 assistant 消息默认采用两层结构：

1. 回复摘要层
2. 执行过程层

回复摘要层默认可见，展示：

- 最终 summary / markdown body
- 基础完成状态

执行过程层默认折叠，展示入口文案，例如“查看执行过程”。

#### 2.3 首次展开时加载 runtime

若该历史 assistant 消息存在 `run_id`，执行过程层首次展开时：

1. 进入该消息自己的 `loading` 状态
2. 调用现有 `getRunProjection(runId)`
3. 按 projection 解析出 plan steps、activities、tool references
4. 缓存到 `historyRuntimeStateByMessageId`

若加载失败：

- 只在该消息范围内展示“重试加载详情”
- 不影响当前 session 其他消息

#### 2.4 Step 二级折叠规则

runtime 加载成功后，执行过程层内部再展示 step 列表。

step 列表默认行为：

- 每个 step 初始只显示标题与状态
- step 内容默认不展开
- 用户点击某个 step 时，再展开该 step 的正文和工具引用

这里的“点击 step 再请求具体内容”在本次设计中的实现方式是：

- 网络请求粒度仍然是“首次展开执行过程时拉整条 projection”
- 单个 step 的后续展开仅为前端本地展开

采用这一策略的原因：

- 当前后端没有 step 级详情接口
- 用户诉求是“默认先看标题，再按需看内容”
- 先满足交互体验，比新增复杂接口更合适

#### 2.5 历史 runtime 缓存粒度

为每条历史 assistant 消息维护独立状态：

- `historyRuntimeLoadState[messageId]`
- `historyRuntimeData[messageId]`
- `expandedHistorySteps[messageId][stepId]`

这样可以做到：

- 同一条消息展开后再次查看不重复请求
- 不同消息之间互不干扰
- step 的展开状态局部化，不污染整条会话

缓存回收要求：

- 本次设计不要求永久保留所有历史 runtime 缓存
- 当抽屉关闭时，可以清理非当前激活 session 的 `historyRuntimeData`
- 当前激活 session 内已展开的历史 runtime 可以继续保留，以避免短时间内重复请求
- 若实现复杂度可控，允许引入轻量 LRU 淘汰，优先清理最久未访问且不属于当前激活 session 的缓存条目

本次实现的最低要求是：

- `open = false` 时清理非活跃 session 的历史 runtime 缓存

#### 2.6 历史展开时的布局稳定性

历史消息首次展开执行过程，或展开某个 step 时，会导致消息块高度明显增加。

这类变化属于用户主动触发的局部展开，不属于流式跟随行为，因此必须满足以下规则：

- 不触发 `stream-follow`
- 不把滚动状态从 `detached` 错误切回 `following`
- 不因为 step 列表一次性渲染而把当前阅读视口顶飞

布局稳定建议：

- 优先利用浏览器滚动锚定能力，例如 `overflow-anchor`
- 若首次注入 step 列表后仍出现明显视口漂移，允许在展开完成后执行一次局部微调滚动
- 微调目标应是保持用户展开前后的视觉焦点稳定，而不是强制回到底部

### Section 3: 滚动跟踪状态机

#### 3.1 两个状态

滚动跟踪只保留两个明确状态：

- `following`
- `detached`

`following` 表示系统拥有滚动控制权。

`detached` 表示用户拥有滚动控制权。

#### 3.2 状态切换规则

进入 `following` 的条件：

- 抽屉初次打开后完成当前 session 首次渲染
- 用户切换 session，且目标 session 完成渲染
- 用户点击“回到底部”按钮
- 用户手动滚回到底部阈值以内

进入 `detached` 的条件：

- 用户手动上滑，离开底部阈值

#### 3.3 跟随触发条件

只有在以下条件同时满足时，才允许程序自动滚动：

- 当前抽屉处于打开状态
- 当前激活 session 未变化
- 当前滚动状态为 `following`
- 新增变化来自当前激活 session 的最后一条 assistant 消息

以下变化不得触发自动跟随：

- 非当前 session 的数据更新
- 历史 assistant 消息 runtime 首次加载
- 历史 step 的展开/收起
- 其他与最后一条 assistant 消息无关的重渲染

#### 3.4 流式中断与重试

如果当前会话正在流式输出，随后因为网络波动、后端异常或中断而展示错误 / Retry UI：

- 不自动改变当前滚动状态
- 若中断前处于 `following`，则保持 `following`
- 若中断前处于 `detached`，则保持 `detached`
- 中断本身不应触发额外滚动修正

如果用户随后点击重试：

- 当前状态仍为 `following` 时，新的流式输出继续跟随当前会话底部区域
- 若用户在中断期间手动上滑并进入 `detached`，重试后的输出不得抢回滚动

#### 3.5 初始定位与流式跟随分离

需要显式区分两种滚动意图：

1. `initial-scroll`
2. `stream-follow`

`initial-scroll` 只在以下时机执行一次：

- 打开抽屉并加载最近 session 后
- 切换 session 并完成消息渲染后

目标始终是容器底部。

`stream-follow` 只在当前 active session 的最后一条 assistant 消息持续增长时执行。

目标是保持最后一条 assistant 消息底部处于 follow zone，而不是盲目 `scrollToBottom`。

#### 3.6 用户控制优先

当用户已经不在底部附近时：

- 任何新内容都不能再抢夺滚动
- 只允许展示“回到底部”按钮
- 直到用户自己回到底部或点击按钮，才恢复跟随

这条规则必须严格覆盖当前流式输出场景。

### Section 4: 组件职责

#### `CopilotSurface`

负责：

- 抽屉打开时机驱动的 session 列表加载
- 最近 session 选择与 session 详情加载
- 当前 session 消息数据拼装
- 滚动状态机
- “回到底部”按钮显隐与恢复跟随

不负责：

- 历史 runtime 的解析细节
- step 内容展开状态渲染细节

#### `AssistantReply`

负责：

- 区分当前消息与历史消息的渲染模式
- 历史消息执行过程层的折叠入口
- step 标题列表和 step 内容展开
- 局部 loading / retry UI

不负责：

- session 列表或详情请求
- 全局滚动逻辑

#### `historyProjection`

负责：

- 从 `run_id` 加载 projection 与 content
- 把 projection 转成历史 runtime 结构
- message 级缓存

不负责：

- 控制 session 切换
- 控制 scroll follow

## 错误处理

### Session 列表失败

- 抽屉显示空态和重试入口
- 不自动进入 session 详情加载

### 单个 Session 详情失败

- 仅该 session 展示错误提示和重试入口
- 用户可继续切换到其他 session

### 历史 Runtime 加载失败

- 仅该消息的执行过程层展示错误状态
- 保留回复摘要层可见
- 提供“重试加载详情”

### Projection 数据缺失

若 projection 缺少 step 内容或不可恢复：

- 执行过程层允许展示“详情不可用”
- 不影响摘要层与会话阅读

## 测试策略

### 单元测试

需要补充或调整：

- 抽屉关闭时不触发 session 列表请求
- 抽屉打开时只请求列表和最近 session 详情
- 快速连续切换多个 session 时，旧请求会被 abort 或被过期保护丢弃
- 切换到其他 session 时只加载该 session，自身 loading 独立
- 历史 assistant 消息首次展开执行过程时才加载 projection
- 已加载过的历史 runtime 再次展开不重复请求
- 抽屉关闭时会清理非活跃 session 的历史 runtime 缓存
- 历史 step 默认只显示标题，点击后才出现内容
- 历史 step 展开不会触发自动跟随，且不应明显打断当前阅读位置
- 流式中断后保留当前 `following` / `detached` 状态；重试时按当前状态决定是否继续跟随
- 用户上滑后进入 `detached`，流式输出不再自动滚动
- 用户回到底部或点击按钮后恢复 `following`

### 集成测试

重点覆盖：

- 多 session 切换场景
- 历史消息展开与重试
- 历史大块内容展开时的视口稳定性
- 当前流式输出与手动滚动冲突场景

## 实施顺序

建议按以下顺序实施：

1. 重构 `CopilotSurface` 的 session 列表与详情状态边界，移除 `defaultMessages` 隐式详情加载
2. 修复多 session loading 问题，并补 session 切换测试
3. 为历史 assistant 消息接入“执行过程”一级折叠
4. 为历史 steps 增加标题级二级折叠与局部缓存
5. 重构滚动状态机，拆分 `initial-scroll` 与 `stream-follow`
6. 补齐滚动与懒加载相关测试

## 验收标准

满足以下条件即可认为本次设计实现正确：

1. 页面初始加载时，不会因为 AI 抽屉组件存在而请求 session 列表或 session 详情
2. 用户打开 AI 抽屉后，只会请求当前 scene 的 session 列表和最近 session 的详情
3. 用户切换任意历史 session 时，该 session 能独立完成加载，不会长期停留在 loading
4. 历史 assistant 回复默认只显示摘要与 step 标题，不直接展开全部 step 内容
5. 用户展开历史执行过程后，只在首次展开时请求 projection；再次展开不重复请求
6. 用户手动上滑后，新的流式输出不会抢夺滚动位置
7. 用户回到底部后，流式输出会恢复自动跟随
