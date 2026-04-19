# 告警规则与通知渠道可配置化设计

日期：2026-04-19  
状态：已确认（待实现计划）

## 1. 背景

OpsPilot 现有监控模块已经具备以下能力：

- 告警规则、通知渠道、投递记录的基础 API（`/alert-rules`、`/alert-channels`、`/alert-deliveries`）。
- Alertmanager Webhook 接入（`/alerts/receiver`）与告警事件入库。
- 前端监控页已有列表展示能力，但配置交互能力不足（新增/编辑/测试发送/路由策略缺失或不完整）。

当前目标是把“可看”升级为“可配置、可路由、可审计、可排障”。

## 2. 目标与范围

### 2.1 目标

- 在系统内配置告警规则与通知渠道。
- 支持混合路由：规则绑定优先，未绑定时按严重级别路由。
- 支持全局 + 项目级配置，项目采用“继承全局 + 增量覆盖”。
- 支持固定失败重试与完整投递记录。
- 支持保存前/保存后测试发送。
- 敏感字段加密存储、接口和页面脱敏展示。
- 权限上实现“运营/SRE 可写、开发只读”。

### 2.2 首发支持渠道

- Webhook
- 钉钉
- 企业微信
- 邮件（SMTP）

### 2.3 非目标

- 首发不做短信通道。
- 首发不做集群级作用域（仅全局 + 项目）。
- 首发不把路由主逻辑外移到外部 Alertmanager 配置中心。

## 3. 方案比较

### 方案 A：在现有 monitoring 代码中直接堆叠功能

优点：开发快。  
缺点：`handler/logic` 进一步膨胀，长期维护和测试成本高。

### 方案 B：在 monitoring 模块内做分层增强（推荐）

优点：兼顾交付速度与后续扩展性，能承载项目级覆盖、混合路由、测试发送和安全要求。  
缺点：首期改动面比方案 A 大。

### 方案 C：强依赖外部 Alertmanager 路由，平台主要做编排

优点：贴近 Prometheus 生态。  
缺点：同步复杂度高，难满足平台内测试发送与项目级覆盖体验。

结论：采用方案 B。

## 4. 总体架构

在 `internal/modules/monitoring` 内新增或拆分以下逻辑层职责：

- `rule_config`：规则 CRUD，阈值模式与 PromQL 模式管理，作用域管理。
- `channel_config`：渠道 CRUD，敏感字段加密与脱敏回显。
- `routing_policy`：规则绑定优先 + 严重级别路由兜底的决策引擎。
- `delivery_engine`：统一投递执行、重试、记录。
- `channel_tester`：临时配置测试与已保存配置测试。
- `notification_provider`：复用 `internal/modules/notification/handler/provider.go`，补齐 SMTP 邮件发送实现。

## 5. 数据模型设计

### 5.1 规则（`alert_rules`）

在现有字段基础上扩展：

- `rule_mode`：`threshold | promql`
- `project_id`：可空；为空代表全局。
- `inherit_key`：用于项目级覆盖映射到全局同名规则。
- `is_override`：是否为覆盖条目。
- `promql_expr`：PromQL 模式表达式。

兼容规则：

- `threshold` 模式使用 `metric/operator/threshold/duration_sec/window_sec/granularity_sec`。
- `promql` 模式下阈值字段可为空或忽略，优先用 `promql_expr`。

### 5.2 渠道（`alert_notification_channels`）

在现有字段基础上扩展：

- `project_id`：可空；为空代表全局。
- `provider`：`webhook|dingtalk|wecom|email`。
- `config_cipher_json`：敏感配置密文。
- `config_masked_json`：脱敏回显缓存（可选，便于前端展示）。

说明：

- `target` 保留作为兼容字段；新流程优先使用 `provider + config`。

### 5.3 规则-渠道绑定（新增表）

建议新增 `alert_rule_channel_bindings`：

- `id`
- `rule_id`
- `channel_id`
- `project_id`（可空）
- `priority`
- `enabled`
- `created_at/updated_at`

### 5.4 严重级别路由策略（新增表）

建议新增 `alert_severity_routes`：

- `id`
- `scope`：`global|project`
- `project_id`（scope=project 时必填）
- `severity`：`critical|warning|info`
- `channel_ids_json`
- `enabled`
- `created_at/updated_at`

### 5.5 投递记录（`alert_notification_deliveries`）

在现有字段基础上扩展：

- `attempt_no`
- `next_retry_at`
- `latency_ms`
- `request_id`
- `error_code`

## 6. 作用域与覆盖策略

作用域：`global + project`。  
项目级策略：继承全局并做增量覆盖。

有效配置计算顺序：

1. 读取全局配置（规则、绑定、严重级别路由、渠道）。
2. 读取项目差异配置。
3. 按 `inherit_key` 或显式映射合并。
4. 生成项目有效视图；未覆盖项自动回退全局。

删除项目覆盖条目后，必须可无缝回退到全局默认行为。

## 7. 混合路由与投递流程

告警触发后的路由顺序：

1. 定位告警所属项目上下文（如无法定位则视为全局）。
2. 加载“有效规则”。
3. 若规则存在显式渠道绑定，按绑定投递。
4. 若无绑定，按严重级别路由投递（优先项目，回退全局）。
5. 若仍为空，兜底到 `default-log`，避免静默丢失。

重试策略（固定）：

- 最多 3 次。
- 退避间隔：1s、2s、4s。
- 每次尝试都写投递记录。

幂等与风暴控制：

- 按 `alert_fingerprint + channel_id + time_window` 做去重窗口控制。
- 不阻断告警入库，只限制短窗口重复通知。

## 8. 安全设计

### 8.1 敏感字段加密

复用现有 `internal/core/utils/secret.go`：

- 写入时 `EncryptText(plain, config.CFG.Security.EncryptionKey)`。
- 使用时按需 `DecryptText(cipher, key)`。
- key 缺失时拒绝写入并返回明确错误。

适用字段示例：

- webhook token / secret
- dingtalk/wecom webhook 完整 URL（或 token 段）
- email 收件人列表（按业务要求可视作敏感）

### 8.2 脱敏输出

接口返回中不下发明文敏感值，仅返回脱敏内容：

- 长字符串：保留前后少量字符，中间 `***`
- URL：仅展示 host 与部分 path
- 邮箱：`a***@domain.com`

### 8.3 权限边界

- 读：`monitoring:read`
- 写：`monitoring:write`

角色分配策略在 RBAC 中配置：

- 运营/SRE 角色赋予 `monitoring:write`
- 开发角色仅赋予 `monitoring:read`

## 9. SMTP 邮件配置设计

按系统级全局配置，不在渠道中保存 SMTP 账号密码。

建议新增配置段（`configs/config.yaml` 与环境变量）：

```yaml
notification:
  smtp:
    host: ${SMTP_HOST}
    port: ${SMTP_PORT}
    username: ${SMTP_USERNAME}
    password: ${SMTP_PASSWORD}
    from: ${SMTP_FROM}
    use_tls: true
    starttls: true
    timeout: 5s
```

渠道中仅保存邮件通知目标与模板参数（如 `to/cc/subject_prefix`）。

## 10. API 设计

基于现有 API 增量扩展：

- `GET /alert-rules`：支持 `scope/project_id/mode` 过滤。
- `POST /alert-rules`、`PUT /alert-rules/:id`：支持 `rule_mode`、`promql_expr`、`project_id`。
- `GET /alert-rules/effective`：返回全局+项目合并后的有效规则。
- `GET /alert-channels`、`POST /alert-channels`、`PUT /alert-channels/:id`：支持 `project_id` 与 provider 配置。
- `POST /alert-channels/test`：支持临时配置测试与已保存配置测试。
- `GET|PUT /alert-rules/:id/channels`：规则绑定渠道。
- `GET|PUT /alert-routing/severity`：管理严重级别路由（全局/项目）。
- `GET /alert-deliveries`：增加 attempt/error/retry 维度查询。

兼容性约束：

- 保持现有 `/alert-rules`、`/alert-channels`、`/alert-deliveries` 基础行为可用。
- 新字段全部采用向后兼容策略（可空或带默认值），避免破坏旧调用方。

## 11. 前端交互设计

新增或改造监控配置页面：

- 规则页：支持阈值模式与 PromQL 模式切换、启停、项目覆盖标识。
- 渠道页：支持 Webhook/钉钉/企微/邮件配置，敏感字段脱敏回显。
- 路由页：配置规则绑定与严重级别路由，明确“绑定优先”。
- 投递页：查看投递详情、失败原因、重试轨迹。
- 渠道测试：在编辑抽屉支持“保存前测试”和“保存后测试”。

路由建议：

- `/monitor/rules`
- `/monitor/channels`
- `/monitor/routing`
- `/monitor/deliveries`

## 12. 错误处理与可观测性

错误处理：

- 配置错误：保存时静态校验并返回结构化错误。
- 发送错误：执行重试并记录最终状态。
- 加解密错误：返回可操作错误信息（如“请检查加密密钥配置”）。

可观测性指标：

- `notification_delivery_total{provider,status}`
- `notification_retry_total{provider}`
- `notification_delivery_latency_ms{provider}`
- `notification_route_fallback_total{level}`（用于监控兜底路径）

日志追踪字段：

- `alert_id`
- `rule_id`
- `channel_id`
- `request_id`
- `attempt_no`

## 13. 测试策略

单元测试：

- 规则模式解析（threshold/promql）。
- 项目增量覆盖与回退。
- 混合路由优先级。
- 敏感字段加密/脱敏逻辑。
- SMTP provider 配置校验与发送路径。

集成测试：

- 告警触发到投递全链路。
- 重试行为与投递记录一致性。
- `alert-channels/test`（临时配置和已保存配置）。
- `monitoring:read/write` 权限边界。

前端测试：

- 规则编辑交互。
- 渠道配置与测试发送。
- 路由配置展示与保存。
- 投递记录列表与详情展示。

## 14. 验收标准

- 可在系统内完成规则与渠道新增、编辑、启停。
- 可配置并生效 Webhook/钉钉/企微/SMTP 邮件渠道。
- 支持全局 + 项目级继承覆盖，项目覆盖可回退全局。
- 路由符合“规则绑定优先，严重级别兜底”。
- 投递失败固定重试 3 次，记录完整。
- 渠道测试支持保存前和保存后。
- 敏感字段加密存储且脱敏展示。
- RBAC 达成“运营/SRE 可写，开发只读”。
