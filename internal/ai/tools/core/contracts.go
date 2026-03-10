package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

type ToolMode string

const (
	ToolModeReadonly ToolMode = "readonly"
	ToolModeMutating ToolMode = "mutating"
)

type ToolRisk string

const (
	ToolRiskLow    ToolRisk = "low"
	ToolRiskMedium ToolRisk = "medium"
	ToolRiskHigh   ToolRisk = "high"
)

type ToolDomain string

const (
	DomainGeneral        ToolDomain = "general"
	DomainInfrastructure ToolDomain = "infrastructure"
	DomainService        ToolDomain = "service"
	DomainCICD           ToolDomain = "cicd"
	DomainMonitor        ToolDomain = "monitor"
	DomainConfig         ToolDomain = "config"
	DomainUser           ToolDomain = "user"
)

type ToolResult struct {
	OK        bool   `json:"ok"`
	ErrorCode string `json:"error_code,omitempty"`
	Data      any    `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	Source    string `json:"source"`
	LatencyMS int64  `json:"latency_ms"`
}

type ToolExecutionError struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Recoverable bool     `json:"recoverable"`
	Suggestions []string `json:"suggestions,omitempty"`
	HintAction  string   `json:"hint_action,omitempty"`
}

type ApprovalRequiredError struct {
	Token     string
	Tool      string
	ExpiresAt time.Time
	Message   string
}

func (e *ApprovalRequiredError) Error() string {
	if e == nil {
		return "approval required"
	}
	if e.Message != "" {
		return e.Message
	}
	return "approval required"
}

func IsApprovalRequired(err error) (*ApprovalRequiredError, bool) {
	var apErr *ApprovalRequiredError
	if errors.As(err, &apErr) {
		return apErr, true
	}
	return nil, false
}

type ConfirmationRequiredError struct {
	Token     string
	Tool      string
	Preview   map[string]any
	ExpiresAt time.Time
	Message   string
}

func (e *ConfirmationRequiredError) Error() string {
	if e == nil {
		return "confirmation required"
	}
	if e.Message != "" {
		return e.Message
	}
	return "confirmation required"
}

func IsConfirmationRequired(err error) (*ConfirmationRequiredError, bool) {
	var cfErr *ConfirmationRequiredError
	if errors.As(err, &cfErr) {
		return cfErr, true
	}
	return nil, false
}

type ToolEventEmitter func(event string, payload any)
type ToolMemoryAccessor interface {
	GetLastToolParams(toolName string) map[string]any
	SetLastToolParams(toolName string, params map[string]any)
}

type ctxKey string

const (
	eventEmitterCtxKey   ctxKey = "ai_tool_event_emitter"
	userIDCtxKey         ctxKey = "ai_user_id"
	approvalCtxKey       ctxKey = "ai_approval_token"
	runtimeCtxKey        ctxKey = "ai_runtime_context"
	memoryAccessorCtxKey ctxKey = "ai_tool_memory_accessor"
)

func WithToolEventEmitter(ctx context.Context, emitter ToolEventEmitter) context.Context {
	return context.WithValue(ctx, eventEmitterCtxKey, emitter)
}

func WithToolUser(ctx context.Context, userID uint64, approvalToken string) context.Context {
	ctx = context.WithValue(ctx, userIDCtxKey, userID)
	return context.WithValue(ctx, approvalCtxKey, approvalToken)
}

func WithToolRuntimeContext(ctx context.Context, runtime map[string]any) context.Context {
	return context.WithValue(ctx, runtimeCtxKey, runtime)
}

func ToolRuntimeContextFromContext(ctx context.Context) map[string]any {
	v, _ := ctx.Value(runtimeCtxKey).(map[string]any)
	if v == nil {
		return map[string]any{}
	}
	return v
}

func WithToolMemoryAccessor(ctx context.Context, accessor ToolMemoryAccessor) context.Context {
	return context.WithValue(ctx, memoryAccessorCtxKey, accessor)
}

func ToolMemoryAccessorFromContext(ctx context.Context) ToolMemoryAccessor {
	v := ctx.Value(memoryAccessorCtxKey)
	if v == nil {
		return nil
	}
	acc, _ := v.(ToolMemoryAccessor)
	return acc
}

func ToolUserFromContext(ctx context.Context) (uint64, string) {
	uid, _ := ctx.Value(userIDCtxKey).(uint64)
	token, _ := ctx.Value(approvalCtxKey).(string)
	return uid, token
}

func EmitToolEvent(ctx context.Context, event string, payload any) {
	v := ctx.Value(eventEmitterCtxKey)
	if v == nil {
		return
	}
	emitter, ok := v.(ToolEventEmitter)
	if !ok || emitter == nil {
		return
	}
	emitter(event, payload)
}

func MarshalToolResult(result ToolResult) (string, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

type ToolInputError struct {
	Code    string
	Field   string
	Message string
}

func (e *ToolInputError) Error() string {
	if e == nil {
		return "invalid tool input"
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Field)
	}
	return e.Code
}

func NewMissingParam(field, message string) error {
	return &ToolInputError{Code: "missing_param", Field: field, Message: message}
}

func NewInvalidParam(field, message string) error {
	return &ToolInputError{Code: "invalid_param", Field: field, Message: message}
}

func NewParamConflict(field, message string) error {
	return &ToolInputError{Code: "param_conflict", Field: field, Message: message}
}

func AsToolInputError(err error) (*ToolInputError, bool) {
	var ie *ToolInputError
	if errors.As(err, &ie) {
		return ie, true
	}
	return nil, false
}

var toolCallSeq uint64

func nextToolCallID() string {
	n := atomic.AddUint64(&toolCallSeq, 1)
	return fmt.Sprintf("tc-%d-%d", time.Now().UnixNano(), n)
}

func NextToolCallID() string {
	return nextToolCallID()
}

// NormalizeToolName keeps tool naming compatible with OpenAI-style function names.
// Legacy dotted names are converted to underscore style.
func NormalizeToolName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	return strings.ReplaceAll(trimmed, ".", "_")
}

type PlatformDeps struct {
	DB        *gorm.DB
	Clientset *kubernetes.Clientset
}
