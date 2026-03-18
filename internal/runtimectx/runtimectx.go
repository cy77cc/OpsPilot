package runtimectx

import (
	"context"
	"sync"
)

type ctxKey struct{}

type AIMetadata struct {
	SessionID string
	RunID     string
	UserID    uint64
	Scene     string
}

type Context struct {
	TraceID   string
	UID       string
	Role      string
	ClientIP  string
	RequestID string
	StartTime int64
	EndTime   int64
	Latency   int64
	Token     string
	Services  any
	AIMeta    AIMetadata
	data      map[any]any
	mu        sync.RWMutex
}

func NewContext(opts ...func(ctx *Context)) *Context {
	ctx := &Context{
		data: make(map[any]any),
	}
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

func WithContext(parent context.Context, c *Context) context.Context {
	return context.WithValue(parent, ctxKey{}, c)
}

func FromContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(ctxKey{}).(*Context)
	return c
}

func Ensure(ctx context.Context) (context.Context, *Context) {
	if current := FromContext(ctx); current != nil {
		return ctx, current
	}
	runtime := NewContext()
	return WithContext(ctx, runtime), runtime
}

func WithServices(ctx context.Context, services any) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.Services = services
	return ctx
}

func Services(ctx context.Context) any {
	runtime := FromContext(ctx)
	if runtime == nil {
		return nil
	}
	return runtime.Services
}

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

func WithTraceID(ctx context.Context, traceID string) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.TraceID = traceID
	return ctx
}

func TraceID(ctx context.Context) string {
	runtime := FromContext(ctx)
	if runtime == nil {
		return ""
	}
	return runtime.TraceID
}

func WithAIMetadata(ctx context.Context, meta AIMetadata) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.AIMeta = meta
	return ctx
}

func AIMetadataFrom(ctx context.Context) AIMetadata {
	runtime := FromContext(ctx)
	if runtime == nil {
		return AIMetadata{}
	}
	return runtime.AIMeta
}

func WithValue(ctx context.Context, key, value any) context.Context {
	ctx, runtime := Ensure(ctx)
	runtime.Set(key, value)
	return ctx
}

func Value(ctx context.Context, key any) any {
	runtime := FromContext(ctx)
	if runtime == nil {
		return nil
	}
	return runtime.Get(key)
}

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

func (c *Context) Get(key any) any {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[key]
}

func (c *Context) Set(key, value any) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}
