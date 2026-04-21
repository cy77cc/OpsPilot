# AI 表单助手设计文档 (AI Form Assistant Design)

## 1. 目标 (Objectives)
在运维平台表单填写中引入 AI 辅助能力，降低用户在编写复杂配置（如 PromQL、Cron 表达式、正则规则等）时的心智负担和门槛，提升运维效率。

## 2. 核心场景 (Key Scenarios)
- **复杂查询编写：** 帮助编写 PromQL、SQL 或 Elasticsearch 查询语句。
- **格式化配置：** 生成 Cron 表达式、正则表达式、JSON/YAML 模板。
- **文本润色：** 辅助填写告警描述、处理建议或任务说明。
- **卡顿引导：** 在用户犹豫不决时（停留超过 3 秒）主动提供帮助选项。

## 3. 详细设计 (Detailed Design)

### 3.1 交互设计 (UX/UI)
- **触发入口：** 在 Input 组件的 `suffix` 区域嵌入星星图标 (✨)。图标默认灰色，悬停时变为紫色并伴有缩放效果。
- **气泡提示 (Pause Hint)：** 当用户连续 3 秒无输入且输入框不为空时，在 ✨ 图标旁淡入一个微小的提示气泡：“✨ 需要 AI 帮助吗？”，点击可直接打开对话框。
- **流式预览对话框：**
    - 点击触发后，在输入框正下方弹出一个极简风格的对话框。
    - 用户输入自然语言需求（如：“帮我写个内存超过 80% 的告警”）。
    - 预览区以流式打印（Typing effect）显示 AI 生成的结果。
    - 用户点击“采纳建议”后，结果自动填入原输入框并关闭对话框。
- **全局控制：** 在用户个人中心提供“开启 AI 表单辅助”的总开关。

### 3.2 后端接口与架构 (Backend Architecture)
- **核心组件：**
    - **FormAssistAgent:** 基于 `adk.ChatModelAgent` 实现的轻量级 Agent。
    - **Prompt Engineering:** 手写 System Prompt，注入 `FieldMeta` 和 `FormContext`。
    - **OutputNormalizationTool:** 作为 Agent 的工具，负责将 LLM 的原始输出清洗为符合 `rules` 的纯净格式（如去除 Markdown 代码块标识、校验语法）。
- **协议：** 基于 Server-Sent Events (SSE) 的流式通信。
- **端点：** `POST /api/ai/v1/assist/form/stream`
- **流程：**
    1. Handler 接收表单上下文。
    2. 构造包含字段用途和规则的 Prompt。
    3. 调用 `ChatModelAgent`。
    4. 结果经 `OutputNormalizationTool` 处理后流式返回。

### 3.3 前端组件设计
- **核心组件：** `AIFormAssistant` (新组件) 和 扩展后的 `GuidedFormItem`。
- **Hooks：** `useFormAIService` 用于处理 SSE 连接、流式状态管理及卡顿计时逻辑。

## 4. 成功指标 (Success Criteria)
- 接口响应延迟（首字）控制在 500ms 以内。
- AI 生成结果的“用户采纳率”达到 60% 以上。
- 用户配置复杂规则的时间缩短 40%。

## 5. 风险与规避
- **生成错误：** 在预览区提供免责声明，并允许用户在采纳后手动二次修改。
- **上下文隐私：** `form_context` 仅发送必要字段，不包含敏感信息。
