# API Routing

后端 API 路由规范，定义服务模块的组织和注册方式。

## Requirements

### REQ-001: 模块化服务架构

后端采用模块化服务架构，每个功能模块独立注册。

```
internal/service/
├── service.go           # 服务注册入口
├── host/                # 主机管理模块
│   ├── routes.go
│   ├── handler.go
│   └── logic.go
├── deployment/          # 部署管理模块
├── jobs/                # 任务调度模块
└── ...
```

### REQ-002: 服务注册模式

每个服务模块通过 `Register*Handlers` 函数注册路由。

```go
// internal/service/jobs/routes.go
func RegisterJobsHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
    h := NewHandler(svcCtx)
    g := v1.Group("/jobs", middleware.JWTAuth())
    {
        g.GET("", h.ListJobs)
        g.POST("", h.CreateJob)
        g.GET("/:id", h.GetJob)
        // ...
    }
}
```

### REQ-003: 统一服务入口

所有服务在 `service.go` 中统一注册。

```go
// internal/service/service.go
func RegisterServices(engine *gin.Engine, serverCtx *svc.ServiceContext) {
    v1 := engine.Group("/api/v1")

    host.RegisterHostHandlers(v1, serverCtx)
    deployment.RegisterDeploymentHandlers(v1, serverCtx)
    jobs.RegisterJobsHandlers(v1, serverCtx)
    // ...
}
```

## API Endpoints

### Jobs API

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /api/v1/jobs | 获取任务列表 | JWT |
| POST | /api/v1/jobs | 创建任务 | JWT |
| GET | /api/v1/jobs/:id | 获取任务详情 | JWT |
| PUT | /api/v1/jobs/:id | 更新任务 | JWT |
| DELETE | /api/v1/jobs/:id | 删除任务 | JWT |
| POST | /api/v1/jobs/:id/start | 启动任务 | JWT |
| POST | /api/v1/jobs/:id/stop | 停止任务 | JWT |
| GET | /api/v1/jobs/:id/executions | 执行记录 | JWT |
| GET | /api/v1/jobs/:id/logs | 任务日志 | JWT |

### Hosts API

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /api/v1/hosts | 主机列表 | JWT |
| POST | /api/v1/hosts | 创建主机 | JWT |
| GET | /api/v1/hosts/:id | 主机详情 | JWT |
| PUT | /api/v1/hosts/:id | 更新主机 | JWT |
| DELETE | /api/v1/hosts/:id | 删除主机 | JWT |
| POST | /api/v1/hosts/probe | 探测主机 | JWT |
| POST | /api/v1/hosts/:id/ssh/exec | SSH 执行 | JWT |

## Response Format

所有 API 返回统一格式：

```json
{
  "code": 1000,
  "msg": "success",
  "data": {
    "list": [],
    "total": 0
  }
}
```

### 错误响应

```json
{
  "code": 4001,
  "msg": "参数错误",
  "data": null
}
```

## Permission Model

API 使用基于资源的权限控制：

```go
// 检查权限
if !httpx.Authorize(c, db, "task:read", "task:*") {
    return // 返回 403
}
```

权限格式: `resource:action` 或 `resource:*`

## Database Models

### Job Model

```go
type Job struct {
    ID          uint       `gorm:"primaryKey" json:"id"`
    Name        string     `json:"name"`
    Type        string     `json:"type"`        // shell, script
    Command     string     `json:"command"`
    HostIDs     string     `json:"host_ids"`
    Cron        string     `json:"cron"`
    Status      string     `json:"status"`      // pending, running, success, failed
    Timeout     int        `json:"timeout"`
    Priority    int        `json:"priority"`
    Description string     `json:"description"`
    LastRun     *time.Time `json:"last_run"`
    NextRun     *time.Time `json:"next_run"`
    CreatedBy   uint       `json:"created_by"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

## Migration Notes

### 2026-03-02: 添加 Jobs API

新增任务调度服务模块：
- 创建 `internal/service/jobs/` 目录
- 实现 CRUD + 启动/停止操作
- 添加 `Job`, `JobExecution`, `JobLog` 模型
