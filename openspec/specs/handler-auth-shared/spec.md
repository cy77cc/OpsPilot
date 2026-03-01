# Handler Auth Shared

## Purpose

定义 `internal/httpx` 包提供的共享鉴权与参数解析工具函数，消除各 handler 中重复实现的 UID 提取、管理员判断、权限检查等逻辑，统一为单一可信来源。

## Requirements

### Requirement: httpx 包提供统一的 UID 提取函数

`internal/httpx` SHALL 提供 `UIDFromCtx(c *gin.Context) uint64` 函数，从 gin context 的 `"uid"` key 提取用户 ID，支持 `uint`、`uint64`、`int`、`int64`、`float64` 等类型断言，提取失败返回 0。各 handler MUST 使用此函数替代自行实现的 `uidFromContext` / `getUID` 等同名函数。

#### Scenario: 从上下文提取 UID
- **WHEN** JWT 中间件已将 uid 写入 gin context，handler 调用 `httpx.UIDFromCtx(c)`
- **THEN** 返回正确的用户 ID uint64 值

#### Scenario: 上下文无 UID
- **WHEN** context 中无 `"uid"` key（未登录或匿名请求）
- **THEN** 返回 0

---

### Requirement: httpx 包提供统一的管理员判断函数

`internal/httpx` SHALL 提供 `IsAdmin(db *gorm.DB, userID uint64) bool` 函数。判断逻辑 SHALL 同时覆盖两种管理员来源：
1. 用户名（username）大小写不敏感等于 `"admin"`
2. 用户拥有 code 大小写不敏感等于 `"admin"` 的角色

#### Scenario: 用户名 admin 判断
- **WHEN** 用户 username 为 `"admin"` 或 `"Admin"`
- **THEN** `IsAdmin` 返回 `true`

#### Scenario: 角色 admin 判断
- **WHEN** 用户拥有 code 为 `"admin"` 的角色
- **THEN** `IsAdmin` 返回 `true`

#### Scenario: 普通用户
- **WHEN** 用户既无 admin 用户名也无 admin 角色
- **THEN** `IsAdmin` 返回 `false`

---

### Requirement: httpx 包提供统一的权限检查函数

`internal/httpx` SHALL 提供 `HasAnyPermission(db *gorm.DB, userID uint64, codes ...string) bool`。检查逻辑：
1. admin 用户（`IsAdmin` 为 true）直接返回 true
2. 查询用户通过角色关联获得的权限 code 集合
3. 集合中含 `"*:*"` 则返回 true
4. 集合中含任一 `codes` 参数或对应的 `<domain>:*` 通配符则返回 true

#### Scenario: admin 用户跳过权限检查
- **WHEN** `IsAdmin` 为 true
- **THEN** `HasAnyPermission` 返回 true，不查询权限表

#### Scenario: 通配符权限
- **WHEN** 用户拥有 `"*:*"` 权限
- **THEN** 对任意 code 请求均返回 true

#### Scenario: 域级通配符
- **WHEN** 用户拥有 `"k8s:*"` 权限，检查 `"k8s:read"`
- **THEN** 返回 true

#### Scenario: 无匹配权限
- **WHEN** 用户权限集合与所有 codes 均不匹配
- **THEN** 返回 false

---

### Requirement: httpx 包提供 Authorize 快捷函数

`internal/httpx` SHALL 提供 `Authorize(c *gin.Context, db *gorm.DB, codes ...string) bool`。当权限不满足时，函数自动调用 `httpx.Fail(c, xcode.Forbidden, "")` 并返回 false；满足时返回 true。Handler 层 MUST 以 `if !httpx.Authorize(c, h.svcCtx.DB, "k8s:read") { return }` 模式替换现有的 `authorize()` 方法。

#### Scenario: 权限通过
- **WHEN** 用户拥有所需权限
- **THEN** 函数返回 true，不写入任何响应

#### Scenario: 权限拒绝
- **WHEN** 用户缺少权限
- **THEN** 函数向 context 写入 `{"code":2004,"msg":"无权限"}` 并返回 false

---

### Requirement: httpx 包提供参数解析工具函数

`internal/httpx` SHALL 提供以下函数，替代各 handler 中重复定义的版本：
- `UintFromParam(c *gin.Context, key string) uint`
- `UintFromQuery(c *gin.Context, key string) uint`
- `ToUint64(v any) uint64`（支持 uint/uint64/int/int64/float64 类型断言）

#### Scenario: 路径参数解析
- **WHEN** 路由路径为 `/api/v1/clusters/:id`，请求路径为 `/api/v1/clusters/42`
- **THEN** `httpx.UintFromParam(c, "id")` 返回 `uint(42)`

#### Scenario: 非数字参数
- **WHEN** 路径参数无法解析为数字
- **THEN** 返回 `uint(0)`，不 panic
