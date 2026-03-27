// Package runtimectx 提供运行时上下文管理。
//
// 本文件实现自定义的请求上下文 Context，用于在请求链路中传递
// 追踪信息、用户身份、AI 元数据等运行时数据。
// 支持与标准 context.Context 无缝集成。
package runtimectx

import (
	"context"
	"sync"
)

// ctxKey 是上下文键类型，用于避免键冲突。
type ctxKey struct{}

// AIMetadata 包含 AI 会话相关的元数据。
type AIMetadata struct {
	SessionID    string // 会话 ID
	RunID        string // 运行 ID
	CheckpointID string // 检查点 ID
	UserID       uint64 // 用户 ID
	Scene        string // 场景标识
}

// Context 是运行时上下文，携带请求级别的数据。
//
// 通过 WithContext 注入到标准 context.Context 中，
// 后续可通过 FromContext 提取使用。
type Context struct {
	TraceID   string         // 追踪 ID
	UID       string         // 用户 ID
	Role      string         // 用户角色
	ClientIP  string         // 客户端 IP
	RequestID string         // 请求 ID
	StartTime int64          // 请求开始时间（Unix 时间戳）
	EndTime   int64          // 请求结束时间（Unix 时间戳）
	Latency   int64          // 请求耗时（毫秒）
	Token     string         // 认证 Token
	Services  any            // 服务依赖容器
	AIMeta    AIMetadata     // AI 会话元数据
	data      map[any]any    // 扩展数据存储
	mu        sync.RWMutex   // 读写锁，保护 data
}

// NewContext 创建新的运行时上下文。
//
// 参数:
//   - opts: 可选的配置函数
//
// 返回: 初始化后的 Context 实例。
func NewContext(opts ...func(ctx *Context)) *Context {
	ctx := &Context{
		data: make(map[any]any),
	}
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

// WithContext 将 Context 注入到标准 context.Context 中。
//
// 参数:
//   - parent: 父上下文
//   - c: 运行时上下文
//
// 返回: 包含运行时上下文的新 context.Context。
func WithContext(parent context.Context, c *Context) context.Context {
	return context.WithValue(parent, ctxKey{}, c)
}

// FromContext 从标准 context.Context 中提取运行时上下文。
//
// 参数:
//   - ctx: 标准上下文
//
// 返回: 运行时上下文，如果不存在返回 nil。
func FromContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(ctxKey{}).(*Context)
	return c
}

// Ensure 确保上下文中包含运行时上下文。
//
// 如果已存在则直接返回，否则创建新的并注入。
//
// 参数:
//   - ctx: 标准上下文
//
// 返回: 更新后的上下文和运行时上下文。
func Ensure(ctx context.Context) (context.Context, *Context) {
	if current := FromContext(ctx); current != nil {
		return ctx, current
	}
	runtime := NewContext()
	return WithContext(ctx, runtime), runtime
}

// WithServices 将服务依赖注入到上下文中。
//
// 参数:
//   - ctx: 标准上下文
//   - services: 服务依赖容器
//
// 返回: 更新后的上下文。
func WithServices(ctx context.Context, services any) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.Services = services
	return ctx
}

// Services 从上下文中获取服务依赖。
//
// 参数:
//   - ctx: 标准上下文
//
// 返回: 服务依赖容器，如果不存在返回 nil。
func Services(ctx context.Context) any {
	runtime := FromContext(ctx)
	if runtime == nil {
		return nil
	}
	return runtime.Services
}

// ServicesAs 从上下文中获取类型化的服务依赖。
//
// 参数:
//   - ctx: 标准上下文
//
// 返回: 类型化的服务依赖和是否存在的布尔值。
func ServicesAs[T any](ctx context.Context) (T, bool) {
	var zero T
	value := Services(ctx)
	if value == nil {
		return zero, false
	}
	typed, ok := value.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

// WithTraceID 将追踪 ID 注入到上下文中。
//
// 参数:
//   - ctx: 标准上下文
//   - traceID: 追踪 ID
//
// 返回: 更新后的上下文。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.TraceID = traceID
	return ctx
}

// TraceID 从上下文中获取追踪 ID。
//
// 参数:
//   - ctx: 标准上下文
//
// 返回: 追踪 ID，如果不存在返回空字符串。
func TraceID(ctx context.Context) string {
	runtime := FromContext(ctx)
	if runtime == nil {
		return ""
	}
	return runtime.TraceID
}

// WithAIMetadata 将 AI 元数据注入到上下文中。
//
// 参数:
//   - ctx: 标准上下文
//   - meta: AI 元数据
//
// 返回: 更新后的上下文。
func WithAIMetadata(ctx context.Context, meta AIMetadata) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.AIMeta = meta
	return ctx
}

// AIMetadataFrom 从上下文中获取 AI 元数据。
//
// 参数:
//   - ctx: 标准上下文
//
// 返回: AI 元数据，如果不存在返回空结构。
func AIMetadataFrom(ctx context.Context) AIMetadata {
	runtime := FromContext(ctx)
	if runtime == nil {
		return AIMetadata{}
	}
	return runtime.AIMeta
}

// WithValue 将键值对注入到运行时上下文中。
//
// 参数:
//   - ctx: 标准上下文
//   - key: 键
//   - value: 值
//
// 返回: 更新后的上下文。
func WithValue(ctx context.Context, key, value any) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.Set(key, value)
	return ctx
}

// Value 从运行时上下文中获取指定键的值。
//
// 参数:
//   - ctx: 标准上下文
//   - key: 键
//
// 返回: 值，如果不存在返回 nil。
func Value(ctx context.Context, key any) any {
	runtime := FromContext(ctx)
	if runtime == nil {
		return nil
	}
	return runtime.Get(key)
}

// Detach 分离上下文，创建一个不会被取消的新上下文。
//
// 用于在异步任务中传递上下文数据，避免父上下文取消影响。
// 会复制所有运行时数据到新上下文中。
//
// 参数:
//   - ctx: 原始上下文
//
// 返回: 分离后的新上下文。
func Detach(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	base := context.WithoutCancel(ctx)
	runtime := FromContext(ctx)
	if runtime == nil {
		return base
	}
	cloned := NewContext()
	cloned.TraceID = runtime.TraceID
	cloned.UID = runtime.UID
	cloned.Role = runtime.Role
	cloned.ClientIP = runtime.ClientIP
	cloned.RequestID = runtime.RequestID
	cloned.StartTime = runtime.StartTime
	cloned.EndTime = runtime.EndTime
	cloned.Latency = runtime.Latency
	cloned.Token = runtime.Token
	cloned.Services = runtime.Services
	cloned.AIMeta = runtime.AIMeta
	runtime.mu.RLock()
	for k, v := range runtime.data {
		cloned.data[k] = v
	}
	runtime.mu.RUnlock()
	return WithContext(base, cloned)
}

// Get 从运行时上下文中获取指定键的值。
//
// 线程安全。
//
// 参数:
//   - key: 键
//
// 返回: 值，如果不存在返回 nil。
func (c *Context) Get(key any) any {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[key]
}

// Set 设置运行时上下文中的键值对。
//
// 线程安全。
//
// 参数:
//   - key: 键
//   - value: 值
func (c *Context) Set(key, value any) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}
