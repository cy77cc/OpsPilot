package alertheal

import (
	"context"
	"errors"
	"strings"
	"time"

	aidaoalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/alertheal"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/google/uuid"
)

var ErrInvalidPayload = errors.New("invalid alert payload")
var ErrServiceNotInitialized = errors.New("alertheal service not initialized")
var ErrExecutorNotInitialized = errors.New("alertheal executor not initialized")
var ErrExecutorNotImplemented = errors.New("alertheal executor is not implemented")

type ingestEventDAO interface {
	UpsertIngestEvents(ctx context.Context, rows []*model.AIAlertIngestEvent) ([]model.AIAlertIngestEvent, error)
}

type healJobDAO interface {
	ClaimRunnableJob(ctx context.Context, now time.Time) (*model.AIAlertHealJob, error)
	CancelIfResolved(ctx context.Context, jobID string) (bool, error)
	MarkSucceeded(ctx context.Context, jobID, runID string) error
	MarkWaitingApproval(ctx context.Context, jobID, lastError string) error
	MarkRetryWait(ctx context.Context, jobID, lastError string, nextRetryAt time.Time) error
}

type ExecutionResult struct {
	RunID string
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
	svcCtx *svc.ServiceContext
}

func NewExecutor(appCtx *svc.ServiceContext) Executor {
	return &defaultExecutor{svcCtx: appCtx}
}

func (e *defaultExecutor) Execute(_ context.Context, _ *model.AIAlertHealJob) (*ExecutionResult, error) {
	if e == nil || e.svcCtx == nil || e.svcCtx.DB == nil {
		return nil, ErrExecutorNotInitialized
	}
	return nil, ErrExecutorNotImplemented
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

func (s *Service) MarkWaitingApproval(ctx context.Context, jobID, lastError string) error {
	if s == nil || s.jobDAO == nil {
		return ErrServiceNotInitialized
	}
	return s.jobDAO.MarkWaitingApproval(ctx, strings.TrimSpace(jobID), strings.TrimSpace(lastError))
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

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
