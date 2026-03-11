# Design: 代码注释规范

## 注释风格

统一使用**中文注释**，遵循 Go 官方 godoc 格式。

## 四级注释模板

### 1. 文件级注释

放在 `package` 声明之前：

```go
// Package orchestrator 实现 AI 编排核心逻辑。
//
// 架构概览:
//   Rewrite → Plan → Execute → Summarize
//
// 主要入口:
//   - NewOrchestrator: 创建编排器实例
//   - Run: 执行完整流水线
package ai
```

### 2. 结构体/类型注释

```go
// Orchestrator 是 AI 编排核心，管理执行流水线的状态和依赖。
//
// 字段说明:
//   - sessions: 会话状态存储
//   - rewriter: 输入改写阶段
//   - planner: 任务规划阶段
type Orchestrator struct {
    sessions   *state.SessionState  // 会话状态存储
    rewriter   *rewrite.Rewriter    // 输入改写阶段
    planner    *planner.Planner     // 任务规划阶段
}
```

### 3. 方法/函数注释

```go
// Run 启动编排流水线，处理用户消息并返回结果。
//
// 参数:
//   - ctx: 上下文，用于取消和超时控制
//   - req: 请求参数，包含用户消息和会话信息
//   - emit: 事件回调，用于流式输出
//
// 返回: 成功返回 nil，失败返回错误
func (o *Orchestrator) Run(ctx context.Context, req RunRequest, emit StreamEmitter) error {
```

### 4. 行内注释

解释"为什么"而非"是什么"：

```go
if o.maxIters > 0 {
    // 限制最大迭代次数，防止 planner 无限循环
    iter = min(iter, o.maxIters)
}
```

---

## 各类型文件注释规范

### 类型 1: 路由文件 (routes.go)

**特点**: 端点注册，相对简单

**注释重点**:
- 文件级：说明路由分组
- 函数级：简要说明注册的路由
- 行内：路由分组注释

**示例**:

```go
// Package host 提供主机管理相关的 HTTP 路由注册。
//
// 路由分组:
//   - /hosts/*     主机 CRUD、SSH 终端、文件操作
//   - /credentials SSH 密钥管理
package host

// RegisterHostHandlers 注册主机模块的所有 HTTP 路由。
func RegisterHostHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
    h := handler.NewHandler(svcCtx)
    h.StartHealthCollector()

    // 主机核心接口
    g := v1.Group("/hosts", middleware.JWTAuth())
    {
        // 查询类
        g.GET("", h.List)          // 列表
        g.GET("/:id", h.Get)       // 详情

        // 操作类
        g.POST("", h.Create)       // 创建
        g.POST("/probe", h.Probe)  // 探测
    }

    // 凭证管理接口
    cred := v1.Group("/credentials", middleware.JWTAuth())
    {
        cred.GET("/ssh_keys", h.ListSSHKeys)
        cred.POST("/ssh_keys", h.CreateSSHKey)
    }
}
```

### 类型 2: Handler 文件

**特点**: HTTP 请求处理，调用 service 层

**注释重点**:
- 文件级：说明处理器的职责
- 方法级：HTTP 方法、路径参数、响应格式
- 行内：特殊处理逻辑

**示例**:

```go
// Package handler 实现主机模块的 HTTP 请求处理。
package handler

// List 返回主机列表。
//
// GET /api/v1/hosts
// 响应: { "list": [...], "total": n }
func (h *Handler) List(c *gin.Context) {
    list, err := h.hostService.List(c.Request.Context())
    if err != nil {
        httpx.Fail(c, xcode.ServerError, err.Error())
        return
    }
    httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// Get 返回单个主机详情。
//
// GET /api/v1/hosts/:id
// 路径参数: id - 主机 ID
// 响应: 主机对象或 404
func (h *Handler) Get(c *gin.Context) {
    id, ok := parseID(c)
    if !ok {
        return
    }
    node, err := h.hostService.Get(c.Request.Context(), id)
    if err != nil {
        httpx.Fail(c, xcode.NotFound, "host not found")
        return
    }
    httpx.OK(c, node)
}

// Metrics 返回主机健康指标历史数据。
//
// GET /api/v1/hosts/:id/metrics
// 注意: 需要启用配置 host.health.diagnostics
func (h *Handler) Metrics(c *gin.Context) {
    // 检查功能开关
    if !config.HostHealthDiagnosticsEnabled() {
        httpx.Fail(c, xcode.Forbidden, "host health diagnostics is disabled")
        return
    }
    // ...
}
```

### 类型 3: Model 文件

**特点**: 数据结构定义

**注释重点**:
- 文件级：说明模型所属领域
- 结构体级：表名、关联关系
- 字段级：业务含义

**示例**:

```go
// Package model 定义数据库模型。
package model

// AIChatSession 存储 AI 聊天会话元数据。
//
// 表名: ai_chat_sessions
// 关联: AIChatMessage (一对多)
type AIChatSession struct {
    ID        string    // 会话唯一标识
    UserID    uint64    // 所属用户 ID
    Scene     string    // 场景标识 (如: host, cluster, service)
    Title     string    // 会话标题 (自动生成或用户修改)
    CreatedAt time.Time // 创建时间
    UpdatedAt time.Time // 更新时间
}

// AIChatMessage 存储会话中的单条消息。
//
// 表名: ai_chat_messages
// 字段说明:
//   - Thinking: AI 思考过程 (仅 assistant 角色有效)
//   - MetadataJSON: 扩展元数据 (JSON 格式)
type AIChatMessage struct {
    ID           string    // 消息唯一标识
    SessionID    string    // 所属会话 ID
    Role         string    // 角色: user/assistant/system
    Content      string    // 消息内容
    Thinking     string    // AI 思考过程
    Status       string    // 状态: pending/done/error
    MetadataJSON string    // 扩展元数据
    CreatedAt    time.Time // 创建时间
    UpdatedAt    time.Time // 更新时间
}
```

### 类型 4: DAO 文件

**特点**: 数据访问 + 缓存策略

**注释重点**:
- 文件级：缓存策略说明
- 结构体级：依赖组件
- 方法级：缓存处理逻辑
- 行内：复杂查询解释

**示例**:

```go
// Package user 实现用户数据访问层。
//
// 缓存策略:
//   - 读: 先查缓存，未命中再查数据库
//   - 写: 延迟双删策略，保证缓存一致性
package user

// UserDAO 封装用户数据的访问和缓存逻辑。
type UserDAO struct {
    db    *gorm.DB                   // 数据库连接
    cache *expirable.LRU[string, any] // 本地 LRU 缓存
    rdb   redis.UniversalClient      // Redis 分布式缓存
}

// Create 创建用户并更新缓存。
func (d *UserDAO) Create(ctx context.Context, user *model.User) error {
    // 写入数据库
    if err := d.db.WithContext(ctx).Create(user).Error; err != nil {
        return err
    }
    // 同步写入 Redis 缓存
    key := fmt.Sprintf("%s%d", constants.UserIdKey, user.ID)
    if d.rdb != nil {
        if bs, err := json.Marshal(&user); err == nil {
            d.rdb.SetEx(ctx, key, bs, constants.RdbTTL)
        }
    }
    return nil
}

// Update 更新用户信息，使用延迟双删保证缓存一致性。
//
// 延迟双删流程:
//   1. 删除缓存
//   2. 更新数据库
//   3. 等待 50ms
//   4. 再次删除缓存 (防止旧数据回填)
func (d *UserDAO) Update(ctx context.Context, user *model.User) error {
    key := fmt.Sprintf("%s%d", constants.UserIdKey, user.ID)

    // 第一次删除: 防止后续读取到旧缓存
    if d.rdb != nil {
        d.rdb.Del(ctx, key)
    }

    // 更新数据库
    if err := d.db.WithContext(ctx).Save(user).Error; err != nil {
        return err
    }

    // 延迟双删: 等待可能的并发读完成后再删
    time.Sleep(50 * time.Millisecond)
    if d.rdb != nil {
        d.rdb.Del(ctx, key)
    }
    return nil
}
```

### 类型 5: 中间件文件

**特点**: 请求拦截，关注执行顺序

**注释重点**:
- 方法级：认证/授权流程
- 行内：关键判断点

**示例**:

```go
// Package middleware 提供 HTTP 请求中间件。
package middleware

// JWTAuth 返回 JWT 认证中间件。
//
// 认证流程:
//   1. 从 Header 或 Query 获取 token
//   2. 验证 token 格式和签名
//   3. 解析用户 ID 并存入 gin.Context
//
// 失败响应: 401 Unauthorized
// 成功后续: c.Set("uid", uid)
func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 优先从 Header 获取 token
        accessTokenH := c.Request.Header.Get("Authorization")

        // Header 无 token 时，尝试从 Query 参数获取 (用于 WebSocket)
        if accessTokenH == "" {
            if qToken := strings.TrimSpace(c.Query("token")); qToken != "" {
                accessTokenH = "Bearer " + qToken
            } else {
                c.AbortWithStatusJSON(http.StatusUnauthorized, ...)
                return
            }
        }

        // 解析 Bearer token
        parts := strings.SplitN(accessTokenH, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, ...)
            return
        }

        // 验证并解析 token
        accessToken, err := utils.ParseToken(parts[1])
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, ...)
            return
        }

        // 将用户 ID 注入上下文，供后续 handler 使用
        c.Set("uid", accessToken.Uid)
        c.Next()
    }
}
```

### 类型 6: 工具函数文件

**特点**: 可复用的通用功能

**注释重点**:
- 函数级：输入输出、错误情况
- 行内：算法解释

**示例**:

```go
// Package utils 提供通用工具函数。
package utils

// MyClaims 定义 JWT token 的载荷结构。
type MyClaims struct {
    Uid uint // 用户 ID
    jwt.RegisteredClaims
}

// GenToken 使用 HS256 算法生成 JWT token。
//
// 参数:
//   - id: 用户 ID
//   - isRefreshToken: true 生成 refresh token (有效期更长)
//
// 返回: 签名后的 token 字符串
func GenToken(id uint, isRefreshToken bool) (string, error) {
    // 根据类型选择有效期
    tokenExpireDuration := config.CFG.JWT.Expire
    if isRefreshToken {
        tokenExpireDuration = config.CFG.JWT.RefreshExpire
    }

    // 使用 HS256 签名
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
    return token.SignedString(MySecret)
}
```

### 类型 7: AI 核心模块

**特点**: 复杂编排逻辑

**注释重点**:
- 文件级：架构图、执行流程
- 结构体级：各组件职责
- 方法级：阶段转换逻辑
- 行内：设计决策解释

**示例**:

```go
// Package ai 实现 AI 编排核心逻辑。
//
// 架构概览:
//
//   ┌─────────────────────────────────────────────────────────┐
//   │                    Orchestrator                         │
//   │                                                       │
//   │   ┌─────────┐   ┌─────────┐   ┌──────────┐   ┌───────┐ │
//   │   │ Rewrite │──▶│ Planner │──▶│ Executor │──▶│Summarz│ │
//   │   └─────────┘   └─────────┘   └──────────┘   └───────┘ │
//   │        │             │             │              │     │
//   │        ▼             ▼             ▼              ▼     │
//   │   normalize      plan+tools    expert agents   answer   │
//   └─────────────────────────────────────────────────────────┘
//
// 主要入口:
//   - NewOrchestrator: 创建编排器实例
//   - Run: 执行完整流水线
package ai

// StreamEmitter 定义流式事件回调函数类型。
// 返回 true 继续执行，false 表示客户端断开。
type StreamEmitter func(StreamEvent) bool

// Orchestrator 是 AI 编排核心，管理执行流水线的状态和依赖。
type Orchestrator struct {
    sessions          *state.SessionState    // 会话状态存储
    executions        *runtime.ExecutionStore // 执行状态存储
    rewriter          *rewrite.Rewriter      // 输入改写阶段
    planner           *planner.Planner       // 任务规划阶段
    executor          *executor.Executor     // 任务执行阶段
    summarizer        *summarizer.Summarizer // 结果总结阶段
    renderer          *finalAnswerRenderer   // 最终答案渲染器
    metrics           *AIMetrics             // 指标收集器
    maxIters          int                    // 最大迭代次数
    heartbeatInterval time.Duration          // 心跳间隔
}

// Run 启动编排流水线，处理用户消息并返回结果。
//
// 执行流程:
//   1. Rewrite: 将口语化输入改写为结构化目标
//   2. Plan: 解析资源并生成执行计划
//   3. Execute: 调用专家 Agent 执行各步骤
//   4. Summarize: 汇总结果生成最终答案
func (o *Orchestrator) Run(ctx context.Context, req RunRequest, emit StreamEmitter) error {
    // ...
}
```

---

## 执行计划

### Phase 1: AI 模块 (优先级最高)

约 51 文件，11,371 行代码

```
internal/ai/
├── orchestrator.go        # 核心
├── model.go               # 模型初始化
├── config.go              # 配置
├── gateway_contract.go    # 网关契约
├── metrics.go             # 指标
├── errors.go              # 错误定义
├── executor/              # 执行器
├── planner/               # 规划器
├── rewrite/               # 改写器
├── summarizer/            # 总结器
├── experts/               # 专家 Agent
├── tools/                 # 工具
├── state/                 # 状态管理
├── runtime/               # 运行时
├── events/                # 事件
└── rag/                   # RAG
```

### Phase 2: Service 层

约 120 文件，24,655 行代码

```
internal/service/
├── ai/
├── automation/
├── cicd/
├── cluster/
├── cmdb/
├── dashboard/
├── deployment/
├── host/
├── jobs/
├── monitoring/
├── notification/
├── project/
├── rbac/
├── service/
├── topology/
└── user/
```

### Phase 3: 基础设施

约 40 文件

```
internal/
├── middleware/    # 中间件
├── utils/         # 工具函数
├── dao/           # 数据访问
├── config/        # 配置
├── cache/         # 缓存
├── client/        # 外部客户端
├── component/     # 组件
├── infra/         # 基础设施
├── logger/        # 日志
├── httpx/         # HTTP 工具
├── websocket/     # WebSocket
└── xcode/         # 错误码
```

### Phase 4: Model 层

约 26 文件，1,687 行代码

```
internal/model/
├── ai_*.go        # AI 相关模型
├── user*.go       # 用户模型
├── host*.go       # 主机模型
└── ...
```

## 质量检查

每个文件完成后检查：

1. [ ] 文件级注释存在
2. [ ] 所有公开类型有注释
3. [ ] 所有公开方法有注释
4. [ ] 复杂逻辑有行内注释
5. [ ] 注释使用中文
6. [ ] 注释格式符合 godoc
