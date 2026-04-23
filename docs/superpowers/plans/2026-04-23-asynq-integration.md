# Asynq Task Queue Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate asynq Redis task queue for Bootstrap async execution, replacing goroutine-based execution with proper task queue, retry, and concurrency control.

**Architecture:** asynq Server for task workers, asynq Client for task enqueue, Bootstrap executor registers as task handler, ServiceContext manages queue lifecycle.

**Tech Stack:** Go, asynq, Redis, GORM

---

## Dependencies

This plan depends on:
- Backend bootstrap.go split plan (Task 7 executor.go must be complete)

---

## File Structure

### Files to Create

| File | Purpose | Est. Lines |
|------|---------|------------|
| `internal/modules/cluster/logic/bootstrap/queue.go` | Asynq queue integration | ~150 |
| `internal/core/config/asynq.go` | Asynq configuration | ~30 |
| `internal/svc/queue.go` | Queue lifecycle management | ~80 |
| `internal/cmd/worker.go` | Worker process entry point | ~50 |

### Files to Modify

| File | Change |
|------|--------|
| `internal/core/config/config.go` | Add AsynqConfig section |
| `internal/svc/app_context.go` | Add BootstrapQueue to ServiceContext |
| `internal/cmd/server/main.go` | Initialize queue worker |
| `configs/config.yaml` | Add asynq configuration section |

---

## Task 1: Add asynq dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add asynq dependency**

```bash
go get github.com/hibiken/asynq@latest
```

- [ ] **Step 2: Verify dependency**

```bash
go mod tidy
cat go.mod | grep asynq
```

Expected: `github.com/hibiken/asynq v0.x.x`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add asynq for task queue"
```

---

## Task 2: Create asynq config

**Files:**
- Create: `internal/core/config/asynq.go`

- [ ] **Step 1: Create asynq config struct**

```go
// internal/core/config/asynq.go

package config

// AsynqConfig asynq 任务队列配置
type AsynqConfig struct {
	// Redis 地址
	RedisAddr string `yaml:"redis_addr" json:"redis_addr" env:"ASYNQ_REDIS_ADDR"`
	
	// Redis 密码（可选）
	RedisPassword string `yaml:"redis_password" json:"redis_password" env:"ASYNQ_REDIS_PASSWORD"`
	
	// Redis 数据库编号
	RedisDB int `yaml:"redis_db" json:"redis_db" env:"ASYNQ_REDIS_DB"`
	
	// 最大并发 Worker 数
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency" env:"ASYNQ_MAX_CONCURRENCY"`
	
	// 任务最大重试次数
	MaxRetry int `yaml:"max_retry" json:"max_retry" env:"ASYNQ_MAX_RETRY"`
	
	// 任务超时时间（秒）
	TimeoutSeconds int `yaml:"timeout_seconds" json:"timeout_seconds" env:"ASYNQ_TIMEOUT_SECONDS"`
	
	// 队列优先级配置
	Queues map[string]int `yaml:"queues" json:"queues"`
}

// DefaultAsynqConfig 默认配置
func DefaultAsynqConfig() AsynqConfig {
	return AsynqConfig{
		RedisAddr:      "localhost:6379",
		RedisPassword:  "",
		RedisDB:        0,
		MaxConcurrency: 10,
		MaxRetry:       3,
		TimeoutSeconds: 1800, // 30 minutes
		Queues: map[string]int{
			"critical": 10,
			"default":  5,
			"low":      1,
		},
	}
}

// Validate 验证配置
func (c *AsynqConfig) Validate() error {
	if c.RedisAddr == "" {
		return NewValidateError("asynq.redis_addr", "Redis 地址不能为空")
	}
	if c.MaxConcurrency < 1 {
		return NewValidateError("asynq.max_concurrency", "最大并发数必须大于 0")
	}
	if c.TimeoutSeconds < 60 {
		return NewValidateError("asynq.timeout_seconds", "超时时间不能小于 60 秒")
	}
	return nil
}
```

- [ ] **Step 2: Add to config.go**

Edit `internal/core/config/config.go` to add Asynq field:

```go
// internal/core/config/config.go

type Config struct {
	// ... existing fields
	
	// Asynq 任务队列配置
	Asynq AsynqConfig `yaml:"asynq" json:"asynq"`
}
```

Also update `ValidateConfig`:

```go
func ValidateConfig(cfg *Config) error {
	// ... existing validation
	
	if err := cfg.Asynq.Validate(); err != nil {
		return err
	}
	
	return nil
}
```

- [ ] **Step 3: Run Go build check**

```bash
go build ./internal/core/config/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/core/config/
git commit -m "feat(config): add asynq configuration"
```

---

## Task 3: Create queue.go in bootstrap logic

**Files:**
- Create: `internal/modules/cluster/logic/bootstrap/queue.go`

- [ ] **Step 1: Create queue integration**

```go
// internal/modules/cluster/logic/bootstrap/queue.go

package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/google/uuid"
)

const (
	// TaskTypeBootstrap Bootstrap 任务类型
	TaskTypeBootstrap = "cluster:bootstrap:execute"
	
	// DefaultMaxConcurrency 默认最大并发
	DefaultMaxConcurrency = 10
	
	// DefaultMaxRetry 默认最大重试
	DefaultMaxRetry = 3
	
	// DefaultTimeout 默认超时（30 分钟）
	DefaultTimeout = 30 * time.Minute
)

// Queue Bootstrap 任务队列
type Queue struct {
	client   *asynq.Client
	server   *asynq.Server
	executor *Executor
	config   QueueConfig
}

// QueueConfig 队列配置
type QueueConfig struct {
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	MaxConcurrency int
	MaxRetry       int
	Timeout        time.Duration
	Queues         map[string]int
}

// NewQueue 创建任务队列
func NewQueue(cfg QueueConfig, executor *Executor) *Queue {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	client := asynq.NewClient(redisOpt)

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: cfg.MaxConcurrency,
		Queues:      cfg.Queues,
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			// 重试延迟：第 1 次 1 分钟，第 2 次 5 分钟，第 3 次 10 分钟
		 delays := []time.Duration{1 * time.Minute, 5 * time.Minute, 10 * time.Minute}
		 if n < len(delays) {
			 return delays[n]
		 }
		 return delays[len(delays)-1]
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
			// 记录错误日志
		 fmt.Printf("[asynq] task %s failed: %v\n", t.Type(), err)
		}),
	})

	return &Queue{
		client:   client,
		server:   server,
		executor: executor,
		config:   cfg,
	}
}

// Enqueue 入队 Bootstrap 任务
func (q *Queue) Enqueue(ctx context.Context, config *PreviewReq) (string, error) {
	taskID := generateTaskID()

	payload := TaskPayload{
		TaskID: taskID,
		Config: *config,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化任务 payload 失败: %w", err)
	}

	task := asynq.NewTask(TaskTypeBootstrap, payloadBytes)

	info, err := q.client.EnqueueContext(ctx, task,
		asynq.MaxRetry(q.config.MaxRetry),
		asynq.Timeout(q.config.Timeout),
		asynq.TaskID(taskID),
		asynq.Queue("critical"), // Bootstrap 任务优先级最高
	)
	if err != nil {
		return "", fmt.Errorf("入队失败: %w", err)
	}

	return info.ID, nil
}

// RegisterWorker 注册任务处理器
func (q *Queue) RegisterWorker() {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TaskTypeBootstrap, q.handleBootstrapTask)

	q.server.Start(mux)
}

// handleBootstrapTask 处理 Bootstrap 任务
func (q *Queue) handleBootstrapTask(ctx context.Context, t *asynq.Task) error {
	var payload TaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务 payload 失败: %w", err)
	}

	// 执行 Bootstrap
	err := q.executor.Execute(ctx, payload.TaskID, &payload.Config)
	if err != nil {
		// 如果是上下文取消，不重试
		if ctx.Err() != nil {
			return fmt.Errorf("任务被取消: %w", err)
		}
		// 其他错误可以重试
		return err
	}

	return nil
}

// Shutdown 关闭队列
func (q *Queue) Shutdown() {
	q.server.Shutdown()
	q.client.Close()
}

// GetTaskInfo 获取任务信息（从 asynq Inspector）
func (q *Queue) GetTaskInfo(taskID string) (*asynq.TaskInfo, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     q.config.RedisAddr,
		Password: q.config.RedisPassword,
		DB:       q.config.RedisDB,
	})
	defer inspector.Close()

	info, err := inspector.GetTaskInfo("critical", taskID)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// generateTaskID 生成任务 ID
func generateTaskID() string {
	return fmt.Sprintf("bootstrap-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])
}

// TaskPayload 任务 payload
type TaskPayload struct {
	TaskID string    `json:"task_id"`
	Config PreviewReq `json:"config"`
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/queue.go
git commit -m "feat(backend): add asynq queue integration for bootstrap"
```

---

## Task 4: Update Logic facade to use queue

**Files:**
- Modify: `internal/modules/cluster/logic/bootstrap/logic.go`

- [ ] **Step 1: Add Queue to Logic**

```go
// internal/modules/cluster/logic/bootstrap/logic.go

// Add field
type Logic struct {
	db        *gorm.DB
	ssh       *SSHExecutor
	executor  *Executor
	validator *Validator
	queue     *Queue  // 新增
}

// Update NewLogic
func NewLogic(db *gorm.DB, sshClient *sshclient.Client, queueConfig QueueConfig) *Logic {
	sshExec := NewSSHExecutor(sshClient)
	executor := NewExecutor(db, sshExec)
	validator := NewValidator(db)
	queue := NewQueue(queueConfig, executor)

	return &Logic{
		db:        db,
		ssh:       sshExec,
		executor:  executor,
		validator: validator,
		queue:     queue,
	}
}

// Update Apply method
func (l *Logic) Apply(ctx context.Context, req *PreviewReq) (string, error) {
	// 入队任务
	taskID, err := l.queue.Enqueue(ctx, req)
	if err != nil {
		return "", err
	}
	return taskID, nil
}

// Add StartWorker method
func (l *Logic) StartWorker() {
	l.queue.RegisterWorker()
}

// Add Shutdown method
func (l *Logic) Shutdown() {
	l.queue.Shutdown()
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/logic/bootstrap/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/logic/bootstrap/logic.go
git commit -m "refactor(backend): integrate queue into bootstrap logic"
```

---

## Task 5: Create queue lifecycle manager

**Files:**
- Create: `internal/svc/queue.go`

- [ ] **Step 1: Create queue lifecycle manager**

```go
// internal/svc/queue.go

package svc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/modules/cluster/logic/bootstrap"
)

// QueueManager 任务队列生命周期管理器
type QueueManager struct {
	bootstrapLogic *bootstrap.Logic
	config         config.AsynqConfig
	mu             sync.Mutex
	running        bool
}

// NewQueueManager 创建队列管理器
func NewQueueManager(cfg config.AsynqConfig, bootstrapLogic *bootstrap.Logic) *QueueManager {
	return &QueueManager{
		bootstrapLogic: bootstrapLogic,
		config:         cfg,
	}
}

// Start 启动 Worker
func (m *QueueManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("queue manager already running")
	}

	// 注册 Worker
	m.bootstrapLogic.StartWorker()
	m.running = true

	return nil
}

// Shutdown 关闭 Worker
func (m *QueueManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	// 设置超时等待
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 关闭队列
	m.bootstrapLogic.Shutdown()
	m.running = false

	// 等待关闭完成
	<-shutdownCtx.Done()

	return nil
}

// IsRunning 检查是否运行中
func (m *QueueManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/svc/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/svc/queue.go
git commit -m "feat(backend): add queue lifecycle manager"
```

---

## Task 6: Update ServiceContext

**Files:**
- Modify: `internal/svc/app_context.go`

- [ ] **Step 1: Add QueueManager to ServiceContext**

```go
// internal/svc/app_context.go

// Add to ServiceContext struct
type ServiceContext struct {
	DB             *gorm.DB
	Rdb            redis.UniversalClient
	CacheFacade    *cache.Facade
	CasbinEnforcer *casbin.Enforcer
	Prometheus     prominfra.Client
	
	// 新增
	BootstrapLogic *bootstrap.Logic
	QueueManager   *QueueManager
}

// Update NewServiceContext
func NewServiceContext(cfg *config.Config) (*ServiceContext, error) {
	// ... existing initialization

	// 创建 Bootstrap Logic
	bootstrapLogic := bootstrap.NewLogic(db, sshClient, bootstrap.QueueConfig{
		RedisAddr:      cfg.Asynq.RedisAddr,
		RedisPassword:  cfg.Asynq.RedisPassword,
		RedisDB:        cfg.Asynq.RedisDB,
		MaxConcurrency: cfg.Asynq.MaxConcurrency,
		MaxRetry:       cfg.Asynq.MaxRetry,
		Timeout:        time.Duration(cfg.Asynq.TimeoutSeconds) * time.Second,
		Queues:         cfg.Asynq.Queues,
	})

	// 创建 Queue Manager
	queueManager := NewQueueManager(cfg.Asynq, bootstrapLogic)

	return &ServiceContext{
		DB:             db,
		Rdb:            rdb,
		CacheFacade:    cacheFacade,
		CasbinEnforcer: casbinEnforcer,
		Prometheus:     promClient,
		BootstrapLogic: bootstrapLogic,
		QueueManager:   queueManager,
	}, nil
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/svc/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/svc/app_context.go
git commit -m "feat(backend): integrate queue into service context"
```

---

## Task 7: Create standalone worker entry point

**Files:**
- Create: `internal/cmd/worker/main.go`

- [ ] **Step 1: Create worker process**

```go
// internal/cmd/worker/main.go

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cy77cc/OpsPilot/internal/bootstrap"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

var configFile = flag.String("c", "./configs/config.yaml", "config file path")

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 ServiceContext
	svcCtx, err := svc.NewServiceContext(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建服务上下文失败: %v\n", err)
		os.Exit(1)
	}

	// 启动队列 Worker
	ctx := context.Background()
	if err := svcCtx.QueueManager.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "启动 Worker 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[worker] asynq worker started, waiting for tasks...")

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	fmt.Printf("[worker] received signal: %v, shutting down...\n", sig)

	// 关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := svcCtx.QueueManager.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "关闭失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[worker] shutdown complete")
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/cmd/worker/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/worker/main.go
git commit -m "feat(backend): add standalone worker entry point"
```

---

## Task 8: Update server to start worker

**Files:**
- Modify: `internal/cmd/server/main.go`

- [ ] **Step 1: Start worker in server startup**

```go
// internal/cmd/server/main.go

// Add after creating ServiceContext
func main() {
	// ... existing setup
	
	svcCtx, err := svc.NewServiceContext(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 启动队列 Worker（可选，也可以独立运行）
	if cfg.Asynq.MaxConcurrency > 0 {
		if err := svcCtx.QueueManager.Start(context.Background()); err != nil {
			log.Printf("启动队列 Worker 失败: %v", err)
			// 不阻断服务启动，可以单独运行 worker
		} else {
			log.Println("队列 Worker 启动成功")
		}
	}
	
	// ... rest of server startup
}
```

- [ ] **Step 2: Update shutdown handler**

```go
// Add shutdown handler
func shutdown(svcCtx *svc.ServiceContext) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭队列 Worker
	if svcCtx.QueueManager.IsRunning() {
		svcCtx.QueueManager.Shutdown(ctx)
	}

	// ... existing shutdown logic
}
```

- [ ] **Step 3: Run Go build check**

```bash
go build ./internal/cmd/server/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/server/main.go
git commit -m "feat(backend): integrate worker into server startup"
```

---

## Task 9: Add config.yaml example

**Files:**
- Modify: `configs/config.yaml`

- [ ] **Step 1: Add asynq section**

```yaml
# configs/config.yaml

# ... existing config

asynq:
  redis_addr: "localhost:6379"
  redis_password: ""
  redis_db: 0
  max_concurrency: 10
  max_retry: 3
  timeout_seconds: 1800
  queues:
    critical: 10
    default: 5
    low: 1
```

- [ ] **Step 2: Commit**

```bash
git add configs/config.yaml
git commit -m "config: add asynq configuration example"
```

---

## Task 10: Update handler to use queue

**Files:**
- Modify: `internal/modules/cluster/handler/bootstrap_handler.go`

- [ ] **Step 1: Update ApplyBootstrap handler**

```go
// internal/modules/cluster/handler/bootstrap_handler.go

func (h *BootstrapHandler) ApplyBootstrap(c *gin.Context) {
	var req bootstrap.PreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	// 验证请求
	_, issues := h.logic.Preview(c.Request.Context(), &req)
	if len(issues) > 0 {
		httpx.OK(c, gin.H{
			"code":              2000,
			"msg":               "validation failed",
			"validation_issues": issues,
		})
		return
	}

	// 入队任务（异步执行）
	taskID, err := h.logic.Apply(c.Request.Context(), &req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"message": "Bootstrap 任务已入队",
	})
}
```

- [ ] **Step 2: Run Go build check**

```bash
go build ./internal/modules/cluster/handler/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/cluster/handler/bootstrap_handler.go
git commit -m "refactor(backend): update handler to use task queue"
```

---

## Task 11: Add Docker support for worker

**Files:**
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`

- [ ] **Step 1: Add worker service to docker-compose**

```yaml
# docker-compose.yml

services:
  # ... existing services
  
  worker:
    build:
      context: .
      dockerfile: Dockerfile
    command: ["./bin/worker"]
    environment:
      - ASYNQ_REDIS_ADDR=redis:6379
    depends_on:
      - redis
      - db
    networks:
      - app-network
```

- [ ] **Step 2: Update Dockerfile to build worker**

```dockerfile
# Dockerfile

# Build worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/worker ./internal/cmd/worker
```

- [ ] **Step 3: Commit**

```bash
git add Dockerfile docker-compose.yml
git commit -m "ops: add worker service to docker"
```

---

## Task 12: Integration test

- [ ] **Step 1: Start Redis**

```bash
docker-compose up redis -d
```

- [ ] **Step 2: Start Worker**

```bash
./bin/worker -c configs/config.yaml
```

- [ ] **Step 3: Start Server**

```bash
./bin/server -c configs/config.yaml
```

- [ ] **Step 4: Submit Bootstrap task via API**

```bash
curl -X POST http://localhost:8080/api/v1/clusters/bootstrap/apply \
  -H "Content-Type: application/json" \
  -d '{"name":"test-cluster","control_plane_host_id":1,...}'
```

Expected: Returns `{"task_id":"bootstrap-xxx","status":"pending"}`

- [ ] **Step 5: Verify task execution**

```bash
curl http://localhost:8080/api/v1/clusters/bootstrap/bootstrap-xxx
```

Expected: Task status progresses through steps

- [ ] **Step 6: Commit**

```bash
git add .
git commit -m "test: asynq integration test passed"
```

---

## Success Criteria

1. ✅ asynq dependency added to go.mod
2. ✅ AsynqConfig in config.go with validation
3. ✅ Queue.go in bootstrap logic with Enqueue/RegisterWorker
4. ✅ Worker entry point in internal/cmd/worker/main.go
5. ✅ ServiceContext includes QueueManager
6. ✅ Handler uses queue.Enqueue for async execution
7. ✅ Docker-compose includes worker service
8. ✅ Integration test passes: task enqueued → worker executes → status updated