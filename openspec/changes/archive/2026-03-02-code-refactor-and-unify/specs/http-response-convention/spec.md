## ADDED Requirements

### Requirement: 所有 HTTP 接口响应 200 并携带统一 body 结构

系统中所有业务 API（包括成功、业务错误、权限拒绝、参数校验失败）的响应 HTTP Status Code SHALL 固定为 200。响应 body 结构 SHALL 为：

```json
{ "code": <xcode>, "msg": "<描述>", "data": <payload> }
```

其中：
- `code` 使用 `internal/xcode` 中定义的业务码（成功为 1000，错误为 2xxx/3xxx/4xxx）
- `msg` 为对应业务码的中文描述
- `data` 在错误响应中可省略

#### Scenario: 成功响应
- **WHEN** handler 调用 `httpx.OK(c, data)`
- **THEN** 响应为 HTTP 200，body 为 `{"code":1000,"msg":"请求成功","data":<data>}`

#### Scenario: 业务错误响应
- **WHEN** handler 调用 `httpx.Fail(c, xcode.ParamError, "字段不合法")`
- **THEN** 响应为 HTTP 200，body 为 `{"code":2000,"msg":"字段不合法"}`

#### Scenario: 权限拒绝响应
- **WHEN** 用户无权限，handler 调用 `httpx.Fail(c, xcode.Forbidden, "")`
- **THEN** 响应为 HTTP 200，body 为 `{"code":2004,"msg":"无权限"}`

---

### Requirement: 列表接口使用嵌套 data 结构

列表类接口的 `data` 字段 SHALL 为包含 `list` 和 `total` 的对象，不得将 list 和 total 平铺在顶层。

#### Scenario: 列表响应结构
- **WHEN** 接口返回分页列表
- **THEN** body 结构为 `{"code":1000,"msg":"请求成功","data":{"list":[...],"total":N}}`

---

### Requirement: 参数绑定失败使用专用函数

参数绑定失败（`ShouldBindJSON` / `ShouldBindQuery` 返回错误）的响应 SHALL 通过 `httpx.BindErr(c, err)` 统一输出，code 为 `xcode.ParamError`（2000）。

#### Scenario: JSON 绑定失败
- **WHEN** 请求体格式错误，`ShouldBindJSON` 返回 error
- **THEN** 响应 HTTP 200，body 为 `{"code":2000,"msg":"<binding错误信息>"}`

---

### Requirement: 禁止在 handler 中内联 gin.H 响应

所有 handler MUST NOT 直接调用 `c.JSON(...)` 配合内联 `gin.H{...}` 构造响应。全部响应 SHALL 通过 `httpx` 包函数输出。

#### Scenario: 现有内联响应被替换
- **WHEN** 代码中出现 `c.JSON(http.StatusInternalServerError, gin.H{"success": false, ...})`
- **THEN** 该行被替换为 `httpx.Fail(c, xcode.ServerError, err.Error())`
