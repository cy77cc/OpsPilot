## NEW Spec: Code Comment Convention

本规范定义 Go 代码注释的标准格式和内容要求。

### Requirement: 四级注释结构

所有 Go 源文件 MUST 包含四级注释：

1. 文件级注释
2. 结构体/类型级注释
3. 方法/函数级注释
4. 行内注释

**Acceptance Criteria:**
- [ ] 每个 Go 文件 MUST 有文件级注释
- [ ] 所有公开类型 (首字母大写) MUST 有注释
- [ ] 所有公开方法 MUST 有注释
- [ ] 复杂逻辑块 SHOULD 有行内注释解释意图

### Requirement: 中文注释

所有注释 MUST 使用中文编写。

**Acceptance Criteria:**
- [ ] 注释内容 MUST 使用中文
- [ ] 技术术语 MAY 保留英文
- [ ] 代码示例 MAY 使用英文

### Requirement: 文件级注释格式

文件级注释 MUST 放在 `package` 声明之前。

**Acceptance Criteria:**
- [ ] 第一行 MUST 描述包的职责
- [ ] MAY 包含架构图或流程图
- [ ] MAY 列出主要入口点

**Example:**
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

### Requirement: 结构体注释格式

结构体注释 MUST 描述用途和字段含义。

**Acceptance Criteria:**
- [ ] 第一行 MUST 描述结构体用途
- [ ] 复杂结构体 SHOULD 列出字段说明
- [ ] 私有字段 MAY 在字段后添加行内注释

**Example:**
```go
// Orchestrator 是 AI 编排核心，管理执行流水线的状态和依赖。
type Orchestrator struct {
    sessions   *state.SessionState  // 会话状态存储
    rewriter   *rewrite.Rewriter    // 输入改写阶段
    planner    *planner.Planner     // 任务规划阶段
}
```

### Requirement: 方法注释格式

方法注释 MUST 包含参数说明、返回值说明和副作用。

**Acceptance Criteria:**
- [ ] 第一行 MUST 描述方法功能
- [ ] 有参数时 MUST 有参数说明
- [ ] 有返回值时 MUST 有返回值说明
- [ ] 有副作用时 SHOULD 说明

**Example:**
```go
// Run 启动编排流水线，处理用户消息并返回结果。
//
// 参数:
//   - ctx: 上下文，用于取消和超时控制
//   - req: 请求参数，包含用户消息和会话信息
//   - emit: 事件回调，用于流式输出
//
// 返回: 成功返回 nil，失败返回错误
func (o *Orchestrator) Run(ctx context.Context, req RunRequest, emit StreamEmitter) error
```

### Requirement: 行内注释使用场景

行内注释 SHOULD 解释"为什么"而非"是什么"。

**Acceptance Criteria:**
- [ ] 非显而易见的逻辑 SHOULD 有行内注释
- [ ] 业务规则 SHOULD 有行内注释
- [ ] 性能优化代码 SHOULD 有解释
- [ ] 变通方案或 hack MUST 有解释

**Example:**
```go
if o.maxIters > 0 {
    // 限制最大迭代次数，防止 planner 无限循环
    iter = min(iter, o.maxIters)
}

// 延迟双删: 等待可能的并发读完成后再删
time.Sleep(50 * time.Millisecond)
```

---

## 文件类型专项规范

### Requirement: 路由文件注释规范

路由文件 (routes.go) 的注释 SHOULD 聚焦于端点分组。

**Acceptance Criteria:**
- [ ] 文件级 MUST 说明路由分组结构
- [ ] 路由分组 SHOULD 有注释说明
- [ ] 每个路由端点 SHOULD 有简短注释

### Requirement: Handler 文件注释规范

Handler 文件的注释 SHOULD 包含 HTTP 方法和路径信息。

**Acceptance Criteria:**
- [ ] 方法注释 MUST 包含 HTTP 方法和路径
- [ ] 方法注释 SHOULD 包含请求/响应格式
- [ ] 特殊处理逻辑 SHOULD 有行内注释

### Requirement: Model 文件注释规范

Model 文件的注释 SHOULD 聚焦于字段业务含义。

**Acceptance Criteria:**
- [ ] 结构体注释 MUST 说明表名
- [ ] 结构体注释 SHOULD 说明关联关系
- [ ] 每个字段 SHOULD 有业务含义注释

### Requirement: DAO 文件注释规范

DAO 文件的注释 MUST 说明缓存策略。

**Acceptance Criteria:**
- [ ] 文件级 MUST 说明缓存策略
- [ ] 方法级 SHOULD 说明缓存处理逻辑
- [ ] 复杂查询 SHOULD 有注释

### Requirement: 中间件文件注释规范

中间件文件的注释 SHOULD 说明执行流程和拦截点。

**Acceptance Criteria:**
- [ ] 方法注释 MUST 说明认证/授权流程
- [ ] 关键判断点 SHOULD 有注释

### Requirement: AI 核心模块注释规范

AI 核心模块的注释 MUST 包含架构概览。

**Acceptance Criteria:**
- [ ] 文件级 MUST 包含架构图或流程图
- [ ] 结构体注释 MUST 说明各组件职责
- [ ] 方法注释 MUST 说明阶段转换逻辑
