package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	now      func() time.Time
	redactor governance.Redactor
}

func NewService(db *gorm.DB, redactor governance.Redactor) *Service {
	if redactor == nil {
		redactor = defaultRedactor{}
	}
	return &Service{
		db:       db,
		now:      func() time.Time { return time.Now().UTC() },
		redactor: redactor,
	}
}

func NewServiceWithClock(db *gorm.DB, redactor governance.Redactor, now func() time.Time) *Service {
	svc := NewService(db, redactor)
	if now != nil {
		svc.now = now
	}
	return svc
}

func (s *Service) Record(ctx context.Context, in governance.FinalizeInput) (uint, error) {
	if s == nil || s.db == nil {
		return 0, governance.NewGovError(governance.CodeInternalError, "audit service not configured")
	}

	scope := governance.MergeScopeFromContext(ctx, in.Intent.Scope)
	status := strings.TrimSpace(string(in.Decision.State))
	if status == "" {
		status = string(governance.StateCompleted)
	}
	code := strings.TrimSpace(in.ExecutionCode)
	if code == "" {
		code = strings.TrimSpace(in.Decision.Code)
	}
	if code == "" {
		switch governance.OperationState(status) {
		case governance.StateApprovalRequired:
			code = governance.CodeApprovalRequired
		case governance.StateRejected:
			code = governance.CodeApprovalRejected
		case governance.StateFailed:
			code = governance.CodeInternalError
		default:
			code = governance.CodeSuccess
		}
	}
	message := firstNonEmpty(in.ExecutionMsg, in.Decision.Message)
	latency := int64(0)
	if !in.StartedAt.IsZero() && !in.FinishedAt.IsZero() && in.FinishedAt.After(in.StartedAt) {
		latency = in.FinishedAt.Sub(in.StartedAt).Milliseconds()
	}

	rec := model.OperationAudit{
		Domain:             scope.Domain,
		ScopeClusterID:     uintPtrOrNil(scope.ClusterID),
		ScopeProjectID:     uintPtrOrNil(scope.ProjectID),
		ScopeTeamID:        uintPtrOrNil(scope.TeamID),
		Namespace:          scope.Namespace,
		Environment:        scope.Environment,
		Resource:           scope.Resource,
		ResourceID:         scope.ResourceID,
		Action:             scope.Action,
		OperatorID:         in.Intent.OperatorID,
		Status:             status,
		Code:               code,
		Message:            message,
		RequestSummaryJSON: marshalRedacted(s.redactor, in.Intent.RequestSummary),
		ResultSummaryJSON:  marshalRedacted(s.redactor, in.Result),
		DiagnosticsJSON:    marshalRedacted(s.redactor, in.Diagnostics),
		ApprovalTicket:     strings.TrimSpace(in.Intent.ApprovalToken),
		LatencyMS:          latency,
	}
	if err := s.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return 0, err
	}
	return rec.ID, nil
}

type defaultRedactor struct{}

func (defaultRedactor) Redact(v any) any {
	return redactValue(v)
}

func marshalRedacted(redactor governance.Redactor, v any) string {
	if v == nil {
		return ""
	}
	if redactor == nil {
		redactor = defaultRedactor{}
	}
	redacted := redactor.Redact(v)
	buf, err := json.Marshal(redacted)
	if err != nil {
		return fmt.Sprint(redacted)
	}
	return string(buf)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func uintPtrOrNil(v uint) *uint {
	if v == 0 {
		return nil
	}
	return &v
}

func redactValue(v any) any {
	switch value := v.(type) {
	case nil:
		return nil
	case string:
		return redactString(value)
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if isSensitiveKey(key) {
				out[key] = "***"
				continue
			}
			out[key] = redactValue(item)
		}
		return out
	case map[string]string:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if isSensitiveKey(key) {
				out[key] = "***"
				continue
			}
			out[key] = redactValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(value))
		for _, item := range value {
			out = append(out, redactValue(item))
		}
		return out
	case []string:
		out := make([]any, 0, len(value))
		for _, item := range value {
			out = append(out, redactValue(item))
		}
		return out
	}
	return v
}

func redactString(s string) any {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return s
	}
	if looksSensitiveString(trimmed) {
		return "***"
	}
	if len(trimmed) > 1 && ((trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}') || (trimmed[0] == '[' && trimmed[len(trimmed)-1] == ']')) {
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return redactValue(decoded)
		}
	}
	return s
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	switch lower {
	case "password", "passwd", "secret", "token", "access_token", "refresh_token", "api_key", "apikey", "client_secret", "authorization", "auth", "kubeconfig", "private_key", "tls_key", "credential", "credentials":
		return true
	}
	for _, suffix := range []string{"_password", "-password", ".password", "_secret", "-secret", ".secret", "_token", "-token", ".token", "_key", "-key", ".key"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func looksSensitiveString(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "bearer ") ||
		strings.Contains(lower, "authorization:") ||
		strings.Contains(lower, "password=") ||
		strings.Contains(lower, "passwd=") ||
		strings.Contains(lower, "secret=") ||
		strings.Contains(lower, "token=") ||
		strings.Contains(lower, "client_secret=") ||
		strings.Contains(lower, "api_key=") ||
		strings.Contains(lower, "private key") ||
		strings.Contains(lower, "begin private key") ||
		strings.Contains(lower, "begin rsa private key")
}
