# OpsPilot 全面代码审查问题清单（前后端）

日期：2026-04-14  
审查范围：`web/`（前端）+ `internal/ api/ cmd/`（后端）  
审查维度：代码组织结构、功能正确性、安全性  
产出说明：本清单由前端子 agent、后端子 agent 结果与主线程复核汇总，已做去重与颗粒度对齐，作为后续修复 backlog。

---

## 严重级别定义

- `P0`：高危安全或核心能力失效，需优先修复
- `P1`：高风险缺陷或明显错误设计，需尽快修复
- `P2`：中风险问题，建议排期修复
- `P3`：低风险/维护性问题，择机治理

---

## 问题清单

### R-001 (`P0`, 后端安全) JWT 密钥初始化时机错误，认证根可能失效

- 证据：
  - `internal/core/utils/jwt.go:29`（`MySecret` 在包初始化时读取 `config.CFG.JWT.Secret`）
  - `cmd/opspilot/root.go:28`（配置在运行期才 `MustNewConfig()`）
- 风险：JWT 签名/验签可能长期使用空密钥或错误密钥，导致 token 可伪造或认证行为异常。
- 建议：
  - 移除全局静态 `MySecret`
  - 在 `GenToken/ParseToken` 内按当前配置动态获取密钥并校验非空
  - 启动时对 `jwt.secret` 做 fail-fast
- 测试缺口：缺少“空密钥拒绝启动、配置变更后密钥生效、伪造 token 拒绝”测试。

### R-002 (`P0`, 后端安全) 通知 WebSocket 无鉴权且信任 `user_id` 查询参数

- 证据：
  - `internal/bootstrap/modules.go:63`（`/ws/notifications` 直接注册，无 JWT）
  - `internal/websocket/handler.go:30`（读取 `user_id` 查询参数）
  - `internal/websocket/handler.go:20`（`CheckOrigin` 直接 `true`）
- 风险：攻击者可伪装任意用户订阅通知，造成跨账号数据泄露。
- 建议：
  - 将通知 WS 迁入受 JWT 保护路由
  - 用户身份仅来源于 token claims，移除 `user_id` 查询入参
  - `CheckOrigin` 使用明确白名单
- 测试缺口：缺少“伪造 user_id 被拒绝、跨域来源被拒绝”测试。

### R-003 (`P0`, 后端安全) 项目管理与部署路由缺少 JWT/授权

- 证据：
  - `internal/modules/project/api/routes.go:19-23`（`/projects` 路由组未挂 `JWTAuth`）
- 风险：未认证即可访问项目创建、列表与部署接口。
- 建议：
  - 改为 `v1.Group("/projects", middleware.JWTAuth())`
  - 对读写操作增加权限校验（如 `project:read` / `project:write`）
- 测试缺口：缺少“未登录 401、无权限 403”回归测试。

### R-004 (`P0`, 后端安全) SSH 凭据明文落库（探测/创建/更新链路）

- 证据：
  - `internal/modules/host/logic/probe.go:64`（`PasswordCipher` 直接写入请求密码）
  - `internal/modules/host/logic/onboarding.go:51,124,160`（主机记录持久化密码明文）
- 风险：数据库泄漏时可直接获得主机 SSH 凭据。
- 建议：
  - 写库前统一加密（使用 `security.encryption_key`）
  - 读取时统一解密
  - 对历史数据做迁移修复
- 测试缺口：缺少“落库密文、解密可用、空加密配置拒绝写入”测试。

### R-005 (`P0`, 后端安全) 主机详情接口暴露 `ssh_password`

- 证据：
  - `internal/modules/host/model/node.go:33`（`SSHPassword` 可序列化）
  - `internal/modules/host/handler/host_query.go:33,60`（`List/Get` 直接返回实体）
- 风险：已登录用户可直接读取主机密码字段。
- 建议：
  - `SSHPassword` 改为 `json:"-"`
  - `List/Get` 使用脱敏 DTO，显式排除敏感字段
- 测试缺口：缺少响应体敏感字段断言测试。

### R-006 (`P0`, 后端安全/权限模型) 高危主机操作仅有 JWT、缺少细粒度授权

- 证据：
  - `internal/modules/host/api/routes.go:66,69,72,93`（终端、文件、凭据相关路由）
  - `internal/modules/host/handler/files_handler.go:93`（文件管理未 `Authorize`）
  - `internal/modules/host/handler/terminal_session.go:128`（终端会话未 `Authorize`）
  - `internal/modules/host/handler/credentials_handler.go:25`（凭据管理未 `Authorize`）
- 风险：任意登录用户可执行高危运维操作（读写远端文件、终端交互、管理密钥）。
- 建议：
  - 在相关 handler 增加 `httpx.Authorize`
  - 拆分权限域：`host:file:*` / `host:terminal:*` / `host:credential:*`
- 测试缺口：缺少“普通用户访问高危接口被拒绝”的权限矩阵测试。

### R-007 (`P1`, 前后端安全) WebSocket 将 token 放在 URL 查询串并打印日志

- 证据：
  - `web/src/hooks/useNotificationWebSocket.ts:92,96`（`?token=...`）
  - `web/src/hooks/useNotificationWebSocket.ts:102`（日志打印完整 URL）
  - `web/src/pages/Hosts/HostTerminalPage.tsx:195-197`（终端 WS 同样拼接 `token` 查询参数）
  - `internal/core/middleware/jwt.go:26`（后端接受 query token）
- 风险：token 易经代理日志、浏览器日志、监控链路泄露。
- 建议：
  - 改为短时一次性 WS ticket 或基于 HttpOnly 会话方案
  - 移除任何包含凭据的日志
  - 服务端禁用 query token（保留过渡期可灰度）
- 测试缺口：缺少“URL 不含 token、query token 被拒绝、ticket 正常鉴权”测试。

### R-008 (`P1`, 前端安全) 通知 `action_url` 直接跳转，缺少协议/域名校验

- 证据：
  - `web/src/components/Notification/NotificationPanel.tsx:37`
  - `web/src/contexts/NotificationContext.tsx:179`
- 风险：若通知内容被污染，可触发恶意跳转或脚本协议跳转。
- 建议：
  - 统一封装 `safeNavigate(actionUrl)`
  - 仅允许站内相对路径或 allowlist 域名的 `http/https`
  - 显式拒绝 `javascript:`、`data:` 等协议
- 测试缺口：缺少 URL 安全策略单测（仅有点击流程测试）。

### R-009 (`P1`, 后端安全) SSH 客户端禁用主机密钥校验

- 证据：
  - `internal/client/ssh/ssh.go:33`（`ssh.InsecureIgnoreHostKey()`）
- 风险：中间人攻击可窃取凭据与会话内容。
- 建议：
  - 引入 `known_hosts`/指纹白名单校验
  - 首次接入走显式信任流程并持久化指纹
- 测试缺口：缺少“主机指纹不匹配拒绝连接”测试。

### R-010 (`P1`, 后端功能正确性/并发) 探测 token 消费存在竞态窗口

- 证据：
  - `internal/modules/host/logic/host_service.go:664-670`（更新后未校验 `RowsAffected`）
- 风险：并发请求可能双消费同一 token，破坏一次性令牌约束。
- 建议：
  - 使用事务 + 行级锁，或更新后强制校验 `RowsAffected == 1`
  - 失败时返回“已消费”错误
- 测试缺口：缺少并发双消费竞态测试。

### R-011 (`P1`, 后端安全) 告警 webhook 接口无签名校验

- 证据：
  - `internal/modules/monitoring/api/routes.go:43`（`/alerts/receiver` 无 JWT）
  - `internal/modules/monitoring/handler/handler.go:75`（仅 JSON 解析后入库）
- 风险：外部可伪造告警写库并触发通知风暴。
- 建议：
  - 增加 HMAC 签名校验（或 mTLS / 来源白名单）
  - 签名失败直接拒绝并记录审计
- 测试缺口：缺少“伪造来源/签名错误拒绝”测试。

### R-012 (`P1`, 后端功能正确性) 通知 HTTP 路由未接 JWT，功能实际不可用

- 证据：
  - `internal/modules/notification/api/routes.go:28-36`（`/notifications` 路由组未挂 `JWTAuth`）
  - `internal/modules/notification/handler/notification.go:57-60`（依赖上下文 `uid`，缺失则返回未授权）
- 风险：通知查询与状态更新接口在常规请求下持续返回未授权，前端能力失效。
- 建议：
  - `notifications := r.Group("/notifications", middleware.JWTAuth())`
  - 增加路由级鉴权回归测试
- 测试缺口：缺少通知路由鉴权与 happy-path 测试。

### R-013 (`P2`, 后端安全) JWT 中间件允许 URL `token` 参数

- 证据：
  - `internal/core/middleware/jwt.go:26`
- 风险：凭证通过 URL 传播，扩大泄露面。
- 建议：
  - 默认仅允许 `Authorization: Bearer`
  - WS 等特殊场景使用一次性票据
- 测试缺口：缺少“query token 拒绝”测试。

### R-014 (`P2`, 后端功能正确性) 通知更新消息 ID 序列化错误

- 证据：
  - `internal/websocket/hub.go:192`（`string(rune(notifID))`）
- 风险：客户端无法正确匹配通知 ID，状态更新可能丢失。
- 建议：
  - 改为 `strconv.FormatUint(uint64(notifID), 10)`
- 测试缺口：缺少 WS update 消息字段格式测试。

### R-015 (`P2`, 前端安全/产品一致性) 登录注册错误直接透传后端原文

- 证据：
  - `web/src/api/api.ts:121`
  - `web/src/pages/Auth/LoginPage.tsx:23`
  - `web/src/pages/Auth/RegisterPage.tsx:21`
- 风险：可能暴露内部错误细节并放大账号枚举信号。
- 建议：
  - 认证场景使用通用错误文案 + 受控错误码映射
  - 详细错误仅进入日志通道
- 测试缺口：缺少认证错误去敏展示测试。

### R-016 (`P2`, 前端功能一致性) 菜单可见但路由不可达

- 证据：
  - `web/src/app/layout/navigation.config.tsx:77-82`（关闭治理菜单时仍展示 `/settings/users|roles|permissions`）
  - `web/src/app/routes/platform.routes.tsx:45,49,53`（同场景重定向回 `/settings`）
- 风险：用户看到入口但无法访问，造成功能断链与误导。
- 建议：
  - 对齐菜单与路由策略（隐藏入口或提供可访问页面）
- 测试缺口：缺少 feature flag 下菜单-路由一致性测试。

### R-017 (`P3`, 前端代码组织) `checkPermission` 为硬编码权限源且与真实 RBAC 脱节

- 证据：
  - `web/src/components/RBAC/Authorized.tsx:36-50`
- 风险：当前虽未调用，但后续误用会导致权限判断漂移。
- 建议：
  - 删除该函数或统一改为 `PermissionContext` / `rbacApi` 来源
- 测试缺口：缺少防误用测试或 lint 约束。

---

## 结构性结论（跨问题）

1. 鉴权策略在模块间不一致（有的仅 JWT、有的无 JWT、有的 JWT+权限），需要统一路由治理规范。  
2. 凭据处理链路存在“传输/存储/输出”多点暴露（URL token、localStorage、明文 SSH 密码、API 响应字段）。  
3. 多处高危能力（主机终端、文件、凭据）缺少最小权限模型，当前控制面过宽。  
4. 测试覆盖明显偏功能 happy-path，安全与权限回归用例不足。

---

## 建议的修复顺序

1. 先处理 `P0`：R-001/002/003/004/005/006  
2. 再处理 `P1`：R-007/008/009/010/011/012  
3. 最后处理 `P2/P3`：R-013~R-017

