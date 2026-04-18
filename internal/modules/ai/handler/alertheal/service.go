package alerthealhandler

import (
	"context"
	"errors"
	"strings"
	"time"

	aidaoalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/alertheal"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

var (
	ErrAlertNotFound     = errors.New("alert not found")
	ErrAlertHealNotFound = errors.New("alert-heal job not found")
	ErrRetryNotAllowed   = errors.New("alert-heal job retry not allowed")
)

type Service struct {
	db  *gorm.DB
	dao *aidaoalertheal.DAO
}

func NewService(svcCtx *svc.ServiceContext) *Service {
	if svcCtx == nil {
		return &Service{}
	}
	return &Service{
		db:  svcCtx.DB,
		dao: aidaoalertheal.NewDAO(svcCtx.DB),
	}
}

func (s *Service) DB() *gorm.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *Service) ListJobsByAlert(ctx context.Context, alertID uint) ([]aidaoalertheal.AlertHealJobView, int64, error) {
	if s == nil || s.db == nil || s.dao == nil {
		return nil, 0, errors.New("alert-heal service not initialized")
	}
	if alertID == 0 {
		return nil, 0, ErrAlertNotFound
	}

	alert, err := s.loadAlert(ctx, alertID)
	if err != nil {
		return nil, 0, err
	}

	fingerprint := alertFingerprint(alert.Source)
	if fingerprint == "" {
		return []aidaoalertheal.AlertHealJobView{}, 0, nil
	}

	rows, err := s.dao.ListJobViewsByFingerprint(ctx, fingerprint)
	if err != nil {
		return nil, 0, err
	}
	return rows, int64(len(rows)), nil
}

func (s *Service) GetJob(ctx context.Context, jobID string) (*aidaoalertheal.AlertHealJobView, error) {
	if s == nil || s.dao == nil {
		return nil, errors.New("alert-heal service not initialized")
	}
	row, err := s.dao.GetJobView(ctx, jobID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAlertHealNotFound
	}
	return row, err
}

func (s *Service) RetryJob(ctx context.Context, jobID string) (*aidaoalertheal.AlertHealJobView, error) {
	if s == nil || s.dao == nil {
		return nil, errors.New("alert-heal service not initialized")
	}
	row, err := s.dao.RetryJob(ctx, jobID, time.Now().UTC())
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, ErrAlertHealNotFound
	case errors.Is(err, gorm.ErrInvalidData):
		return nil, ErrRetryNotAllowed
	default:
		return row, err
	}
}

func (s *Service) loadAlert(ctx context.Context, alertID uint) (*monitoringmodel.AlertEvent, error) {
	var alert monitoringmodel.AlertEvent
	if err := s.db.WithContext(ctx).Where("id = ?", alertID).Take(&alert).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAlertNotFound
		}
		return nil, err
	}
	return &alert, nil
}

func alertFingerprint(source string) string {
	source = strings.TrimSpace(source)
	if !strings.HasPrefix(source, "alertmanager/") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(source, "alertmanager/"))
}
