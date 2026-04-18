package alertheal

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	aidaoalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/alertheal"
	ailogic "github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrInvalidPayload = errors.New("invalid alert payload")
var ErrServiceNotInitialized = errors.New("alertheal service not initialized")
var ErrExecutorNotAvailable = errors.New("alertheal executor is not available")

var newAILogic = ailogic.NewAILogic

type ingestEventDAO interface {
	UpsertIngestEvents(ctx context.Context, rows []*model.AIAlertIngestEvent) ([]model.AIAlertIngestEvent, error)
}

type healJobDAO interface {
	ClaimRunnableJob(ctx context.Context, now time.Time) (*model.AIAlertHealJob, error)
	CancelIfResolved(ctx context.Context, jobID string) (bool, error)
	MarkSucceeded(ctx context.Context, jobID, runID string) error
	MarkWaitingApproval(ctx context.Context, jobID, runID, lastError string, consumeRetry bool) error
	MarkRetryWait(ctx context.Context, jobID, lastError string, nextRetryAt time.Time) error
	RenewAutoFixingLease(ctx context.Context, jobID string, now time.Time) error
}

type ExecutionResult struct {
	RunID           string
	RunStatus       string
	WaitingApproval bool
}

type Executor interface {
	Execute(ctx context.Context, job *model.AIAlertHealJob) (*ExecutionResult, error)
}

// Service 提供告警摄取逻辑。
type Service struct {
	dao    ingestEventDAO
	jobDAO healJobDAO
	now    func() time.Time
	newID  func() string
}

func NewService(dao ingestEventDAO) *Service {
	svc := &Service{
		dao:   dao,
		now:   time.Now,
		newID: uuid.NewString,
	}
	if jobDAO, ok := dao.(healJobDAO); ok {
		svc.jobDAO = jobDAO
	}
	return svc
}

func NewServiceFromAppContext(appCtx *svc.ServiceContext) *Service {
	if appCtx == nil || appCtx.DB == nil {
		return NewService(nil)
	}
	return NewService(aidaoalertheal.NewDAO(appCtx.DB))
}

type defaultExecutor struct {
	svcCtx  *svc.ServiceContext
	aiLogic *ailogic.Logic
	now     func() time.Time
}

func NewExecutor(appCtx *svc.ServiceContext) Executor {
	if appCtx == nil || appCtx.DB == nil {
		return nil
	}
	ai := newAILogic(appCtx)
	if ai == nil || ai.AIRouter == nil || ai.RunDAO == nil {
		return nil
	}
	return &defaultExecutor{
		svcCtx:  appCtx,
		aiLogic: ai,
		now:     time.Now,
	}
}

// Ingest 执行 payload 归一化并按 dedupe_key 幂等写入。
func (s *Service) Ingest(ctx context.Context, protocol string, raw []byte) ([]model.AIAlertIngestEvent, error) {
	if s == nil || s.dao == nil {
		return nil, ErrServiceNotInitialized
	}

	normalized, err := NormalizePayload(protocol, raw)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, ErrInvalidPayload
	}

	receivedAt := s.now().UTC()
	rows := make([]*model.AIAlertIngestEvent, 0, len(normalized))
	for _, item := range normalized {
		fingerprint := strings.TrimSpace(item.Fingerprint)
		if fingerprint == "" {
			return nil, ErrInvalidPayload
		}

		status := strings.TrimSpace(item.Status)
		if status == "" {
			status = "firing"
		}
		source := strings.TrimSpace(item.Source)
		if source == "" {
			source = normalizeProtocol(protocol)
		}
		proto := strings.TrimSpace(item.Protocol)
		if proto == "" {
			proto = normalizeProtocol(protocol)
		}

		rows = append(rows, &model.AIAlertIngestEvent{
			ID:              s.newID(),
			Source:          source,
			Protocol:        proto,
			Fingerprint:     fingerprint,
			Status:          status,
			DedupeKey:       DedupeKey(source, fingerprint, status),
			Severity:        defaultString(item.Severity, "warning"),
			Title:           defaultString(item.Title, fingerprint),
			Target:          strings.TrimSpace(item.Target),
			LabelsJSON:      defaultString(item.LabelsJSON, "{}"),
			AnnotationsJSON: defaultString(item.AnnotationsJSON, "{}"),
			RawPayloadJSON:  defaultString(item.RawPayloadJSON, "{}"),
			StartsAt:        item.StartsAt,
			EndsAt:          item.EndsAt,
			ReceivedAt:      receivedAt,
		})
	}

	out, err := s.dao.UpsertIngestEvents(ctx, rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) ClaimRunnableJob(ctx context.Context, now time.Time) (*model.AIAlertHealJob, error) {
	if s == nil || s.jobDAO == nil {
		return nil, ErrServiceNotInitialized
	}
	return s.jobDAO.ClaimRunnableJob(ctx, now.UTC())
}

func (s *Service) CancelIfResolved(ctx context.Context, job *model.AIAlertHealJob) (bool, error) {
	if s == nil || s.jobDAO == nil {
		return false, ErrServiceNotInitialized
	}
	if job == nil {
		return false, nil
	}
	jobID := strings.TrimSpace(job.ID)
	if jobID == "" {
		return false, nil
	}
	return s.jobDAO.CancelIfResolved(ctx, jobID)
}

func (s *Service) MarkSucceeded(ctx context.Context, jobID, runID string) error {
	if s == nil || s.jobDAO == nil {
		return ErrServiceNotInitialized
	}
	return s.jobDAO.MarkSucceeded(ctx, strings.TrimSpace(jobID), strings.TrimSpace(runID))
}

func (s *Service) MarkWaitingApproval(ctx context.Context, jobID, runID, lastError string, consumeRetry bool) error {
	if s == nil || s.jobDAO == nil {
		return ErrServiceNotInitialized
	}
	return s.jobDAO.MarkWaitingApproval(
		ctx,
		strings.TrimSpace(jobID),
		strings.TrimSpace(runID),
		strings.TrimSpace(lastError),
		consumeRetry,
	)
}

func (s *Service) MarkRetryWait(ctx context.Context, jobID, lastError string, nextRetryAt time.Time) error {
	if s == nil || s.jobDAO == nil {
		return ErrServiceNotInitialized
	}
	if nextRetryAt.IsZero() {
		nextRetryAt = s.now().UTC()
	}
	return s.jobDAO.MarkRetryWait(ctx, strings.TrimSpace(jobID), strings.TrimSpace(lastError), nextRetryAt.UTC())
}

func (s *Service) RenewAutoFixingLease(ctx context.Context, jobID string, now time.Time) error {
	if s == nil || s.jobDAO == nil {
		return ErrServiceNotInitialized
	}
	if now.IsZero() {
		now = s.now().UTC()
	}
	return s.jobDAO.RenewAutoFixingLease(ctx, strings.TrimSpace(jobID), now.UTC())
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func (e *defaultExecutor) Execute(ctx context.Context, job *model.AIAlertHealJob) (*ExecutionResult, error) {
	if e == nil || e.svcCtx == nil || e.svcCtx.DB == nil || e.aiLogic == nil || e.aiLogic.RunDAO == nil || e.aiLogic.AIRouter == nil {
		return nil, ErrExecutorNotAvailable
	}
	if job == nil {
		return nil, errors.New("alertheal job is required")
	}

	event, err := e.loadIngestEvent(ctx, job)
	if err != nil {
		return nil, err
	}

	attemptNo := alertHealAttemptNo(job.RetryCount)
	sessionID := alertHealSessionID(job.ID)
	clientRequestID := alertHealClientRequestID(job.ID, attemptNo)
	input := ailogic.ChatInput{
		SessionID:       sessionID,
		ClientRequestID: clientRequestID,
		TraceID:         alertHealTraceID(job.ID, attemptNo),
		Message:         buildAlertHealPrompt(job, event, attemptNo),
		Scene:           "ai",
		Context:         buildAlertHealContext(job, event, attemptNo),
		UserID:          0,
	}

	if err := e.aiLogic.Chat(ctx, input, func(string, any) {}); err != nil {
		return nil, err
	}
	run, err := e.aiLogic.RunDAO.FindByClientRequestID(ctx, sessionID, clientRequestID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("alertheal run not created for job=%s attempt=%d", strings.TrimSpace(job.ID), attemptNo)
	}

	result := &ExecutionResult{
		RunID:     strings.TrimSpace(run.ID),
		RunStatus: strings.TrimSpace(run.Status),
	}
	switch strings.TrimSpace(run.Status) {
	case model.RunStatusWaitingApproval:
		result.WaitingApproval = true
		return result, nil
	case "completed", "completed_with_tool_errors":
		return result, nil
	default:
		errMessage := strings.TrimSpace(run.ErrorMessage)
		if errMessage != "" {
			return nil, fmt.Errorf("alertheal run=%s status=%s err=%s", result.RunID, result.RunStatus, errMessage)
		}
		return nil, fmt.Errorf("alertheal run=%s status=%s", result.RunID, result.RunStatus)
	}
}

func (e *defaultExecutor) loadIngestEvent(ctx context.Context, job *model.AIAlertHealJob) (*model.AIAlertIngestEvent, error) {
	eventID := strings.TrimSpace(job.EventID)
	if eventID == "" {
		return nil, errors.New("alertheal job event_id is required")
	}
	var event model.AIAlertIngestEvent
	if err := e.svcCtx.DB.WithContext(ctx).Where("id = ?", eventID).Take(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("alertheal ingest event %q not found", eventID)
		}
		return nil, err
	}
	return &event, nil
}

func buildAlertHealContext(job *model.AIAlertHealJob, event *model.AIAlertIngestEvent, attemptNo int) map[string]any {
	contextPayload := map[string]any{
		"job_id":           strings.TrimSpace(job.ID),
		"attempt_no":       attemptNo,
		"event_id":         strings.TrimSpace(event.ID),
		"source":           strings.TrimSpace(event.Source),
		"protocol":         strings.TrimSpace(event.Protocol),
		"fingerprint":      strings.TrimSpace(event.Fingerprint),
		"status":           strings.TrimSpace(event.Status),
		"severity":         strings.TrimSpace(event.Severity),
		"title":            strings.TrimSpace(event.Title),
		"target":           strings.TrimSpace(event.Target),
		"labels_json":      strings.TrimSpace(event.LabelsJSON),
		"annotations_json": strings.TrimSpace(event.AnnotationsJSON),
	}
	if rawPayload := strings.TrimSpace(event.RawPayloadJSON); rawPayload != "" {
		contextPayload["raw_payload_json"] = truncatePromptValue(rawPayload, 4000)
	}
	return contextPayload
}

func buildAlertHealPrompt(job *model.AIAlertHealJob, event *model.AIAlertIngestEvent, attemptNo int) string {
	lines := []string{
		"Execute automated alert healing for this incident.",
		"Prioritize safe, low-risk remediation. Request approval for any high-risk action.",
		"",
		"Alert-heal metadata:",
		fmt.Sprintf("- job_id: %s", strings.TrimSpace(job.ID)),
		fmt.Sprintf("- attempt_no: %d", attemptNo),
		fmt.Sprintf("- event_id: %s", strings.TrimSpace(event.ID)),
		fmt.Sprintf("- source: %s", strings.TrimSpace(event.Source)),
		fmt.Sprintf("- protocol: %s", strings.TrimSpace(event.Protocol)),
		fmt.Sprintf("- fingerprint: %s", strings.TrimSpace(event.Fingerprint)),
		fmt.Sprintf("- status: %s", strings.TrimSpace(event.Status)),
		fmt.Sprintf("- severity: %s", strings.TrimSpace(event.Severity)),
		fmt.Sprintf("- title: %s", strings.TrimSpace(event.Title)),
		fmt.Sprintf("- target: %s", strings.TrimSpace(event.Target)),
		fmt.Sprintf("- labels_json: %s", truncatePromptValue(strings.TrimSpace(event.LabelsJSON), 1200)),
		fmt.Sprintf("- annotations_json: %s", truncatePromptValue(strings.TrimSpace(event.AnnotationsJSON), 1200)),
		fmt.Sprintf("- raw_payload_json: %s", truncatePromptValue(strings.TrimSpace(event.RawPayloadJSON), 2000)),
	}
	return strings.Join(lines, "\n")
}

func truncatePromptValue(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func alertHealAttemptNo(retryCount int) int {
	if retryCount < 0 {
		return 1
	}
	return retryCount + 1
}

func alertHealSessionID(jobID string) string {
	return "ahsess-" + stableAlertHealToken(jobID)
}

func alertHealClientRequestID(jobID string, attemptNo int) string {
	if attemptNo <= 0 {
		attemptNo = 1
	}
	return fmt.Sprintf("ahreq-%s-a%d", stableAlertHealToken(jobID), attemptNo)
}

func alertHealTraceID(jobID string, attemptNo int) string {
	if attemptNo <= 0 {
		attemptNo = 1
	}
	return fmt.Sprintf("alertheal-%s-a%d", stableAlertHealToken(jobID), attemptNo)
}

func stableAlertHealToken(v string) string {
	raw := strings.TrimSpace(v)
	if raw == "" {
		raw = "unknown"
	}
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:8])
}
