package alertheal

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/google/uuid"
)

var ErrInvalidPayload = errors.New("invalid alert payload")

type ingestEventDAO interface {
	UpsertIngestEvent(ctx context.Context, row *model.AIAlertIngestEvent) (*model.AIAlertIngestEvent, error)
}

// Service 提供告警摄取逻辑。
type Service struct {
	dao   ingestEventDAO
	now   func() time.Time
	newID func() string
}

func NewService(dao ingestEventDAO) *Service {
	return &Service{
		dao:   dao,
		now:   time.Now,
		newID: uuid.NewString,
	}
}

// Ingest 执行 payload 归一化并按 dedupe_key 幂等写入。
func (s *Service) Ingest(ctx context.Context, protocol string, raw []byte) ([]model.AIAlertIngestEvent, error) {
	if s == nil || s.dao == nil {
		return nil, errors.New("alertheal service not initialized")
	}

	normalized, err := NormalizePayload(protocol, raw)
	if err != nil {
		return nil, err
	}

	out := make([]model.AIAlertIngestEvent, 0, len(normalized))
	receivedAt := s.now().UTC()
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

		row := &model.AIAlertIngestEvent{
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
		}

		saved, err := s.dao.UpsertIngestEvent(ctx, row)
		if err != nil {
			return nil, err
		}
		if saved == nil {
			saved = row
		}
		out = append(out, *saved)
	}

	return out, nil
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
