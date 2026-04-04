package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db         *gorm.DB
	now        func() time.Time
	defaultTTL time.Duration
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:         db,
		now:        func() time.Time { return time.Now().UTC() },
		defaultTTL: 30 * time.Minute,
	}
}

func NewServiceWithOptions(db *gorm.DB, now func() time.Time, ttl time.Duration) *Service {
	svc := NewService(db)
	if now != nil {
		svc.now = now
	}
	if ttl > 0 {
		svc.defaultTTL = ttl
	}
	return svc
}

func (s *Service) Issue(ctx context.Context, intent governance.OperationIntent, reason string) (*governance.ApprovalInfo, error) {
	if s == nil || s.db == nil {
		return nil, governance.NewGovError(governance.CodeInternalError, "approval service not configured")
	}
	scope := governance.MergeScopeFromContext(ctx, intent.Scope)
	if strings.TrimSpace(scope.Domain) == "" {
		return nil, governance.NewGovError(governance.CodeApprovalTokenInvalid, "domain is required")
	}
	if scope.ClusterID == 0 && scope.ProjectID == 0 && scope.TeamID == 0 {
		return nil, governance.NewGovError(governance.CodeApprovalTokenInvalid, "scope is required")
	}

	now := s.now()
	expiresAt := now.Add(s.defaultTTL)
	rec := model.OperationApproval{
		Ticket:         fmt.Sprintf("gov-appr-%d", now.UnixNano()),
		Domain:         scope.Domain,
		ScopeClusterID: uintPtrOrNil(scope.ClusterID),
		ScopeProjectID: uintPtrOrNil(scope.ProjectID),
		ScopeTeamID:    uintPtrOrNil(scope.TeamID),
		Namespace:      strings.TrimSpace(scope.Namespace),
		Environment:    strings.TrimSpace(scope.Environment),
		Resource:       strings.TrimSpace(scope.Resource),
		ResourceID:     strings.TrimSpace(scope.ResourceID),
		Action:         strings.TrimSpace(scope.Action),
		ContextJSON:    mustJSONString(scope.Context),
		Reason:         strings.TrimSpace(reason),
		Status:         "pending",
		RequestBy:      intent.OperatorID,
		ExpiresAt:      &expiresAt,
	}
	if err := s.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, err
	}
	return &governance.ApprovalInfo{
		Ticket:    rec.Ticket,
		ExpiresAt: rec.ExpiresAt,
		Reason:    rec.Reason,
	}, nil
}

func (s *Service) Confirm(ctx context.Context, ticket string, approverID uint, approved bool, note string) error {
	if s == nil || s.db == nil {
		return governance.NewGovError(governance.CodeInternalError, "approval service not configured")
	}
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return governance.NewGovError(governance.CodeApprovalTokenInvalid, "ticket is required")
	}
	now := s.now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rec model.OperationApproval
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("ticket = ?", ticket).First(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return governance.NewGovError(governance.CodeApprovalTokenInvalid, "approval ticket not found")
			}
			return err
		}
		switch strings.ToLower(strings.TrimSpace(rec.Status)) {
		case "approved", "rejected":
			return nil
		}
		rec.ReviewBy = approverID
		if approved {
			rec.Status = "approved"
		} else {
			rec.Status = "rejected"
		}
		rec.ReplayMessage = strings.TrimSpace(note)
		rec.UpdatedAt = now
		return tx.Save(&rec).Error
	})
}

func (s *Service) Consume(ctx context.Context, intent governance.OperationIntent) error {
	if s == nil || s.db == nil {
		return governance.NewGovError(governance.CodeInternalError, "approval service not configured")
	}
	ticket := strings.TrimSpace(intent.ApprovalToken)
	if ticket == "" {
		return governance.NewGovError(governance.CodeApprovalTokenInvalid, "approval token is required")
	}
	scope := governance.MergeScopeFromContext(ctx, intent.Scope)
	now := s.now()
	var replayErr error

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rec model.OperationApproval
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("ticket = ?", ticket).First(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return governance.NewGovError(governance.CodeApprovalTokenInvalid, "approval ticket not found")
			}
			return err
		}
		if err := s.ensureScopeMatches(rec, scope); err != nil {
			return err
		}
		if rec.ExpiresAt != nil && now.After(*rec.ExpiresAt) {
			return governance.NewGovError(governance.CodeApprovalTokenExpired, "approval token expired")
		}
		switch strings.ToLower(strings.TrimSpace(rec.Status)) {
		case "pending":
			return governance.NewGovError(governance.CodeApprovalNotApproved, "approval ticket not approved")
		case "rejected":
			return governance.NewGovError(governance.CodeApprovalRejected, "approval ticket rejected")
		case "approved":
			if rec.ConsumedAt != nil {
				rec.ReplayCount++
				rec.ReplayBy = intent.OperatorID
				replayAt := now
				rec.ReplayAt = &replayAt
				rec.ReplayCode = governance.CodeApprovalTokenReplay
				rec.ReplayMessage = governance.CodeApprovalTokenReplay
				if err := tx.Save(&rec).Error; err != nil {
					return err
				}
				replayErr = governance.NewGovError(governance.CodeApprovalTokenReplay, "approval token replayed")
				return nil
			}
			consumedAt := now
			rec.ConsumedAt = &consumedAt
			rec.ConsumedBy = intent.OperatorID
			rec.ReplayCount = 0
			rec.ReplayAt = nil
			rec.ReplayBy = 0
			rec.ReplayCode = ""
			rec.ReplayMessage = ""
			if err := tx.Save(&rec).Error; err != nil {
				return err
			}
			return nil
		default:
			return governance.NewGovError(governance.CodeApprovalNotApproved, "approval ticket not approved")
		}
	})
	if err != nil {
		return err
	}
	return replayErr
}

func (s *Service) ensureScopeMatches(rec model.OperationApproval, scope governance.Scope) error {
	if rec.Domain != strings.TrimSpace(scope.Domain) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !uintPtrMatches(rec.ScopeClusterID, scope.ClusterID) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !uintPtrMatches(rec.ScopeProjectID, scope.ProjectID) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !uintPtrMatches(rec.ScopeTeamID, scope.TeamID) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !strings.EqualFold(strings.TrimSpace(rec.Namespace), strings.TrimSpace(scope.Namespace)) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !strings.EqualFold(strings.TrimSpace(rec.Environment), strings.TrimSpace(scope.Environment)) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !strings.EqualFold(strings.TrimSpace(rec.Resource), strings.TrimSpace(scope.Resource)) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !strings.EqualFold(strings.TrimSpace(rec.ResourceID), strings.TrimSpace(scope.ResourceID)) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !strings.EqualFold(strings.TrimSpace(rec.Action), strings.TrimSpace(scope.Action)) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	if !jsonMatches(rec.ContextJSON, scope.Context) {
		return governance.NewGovError(governance.CodeApprovalScopeMismatch, "approval token scope mismatch")
	}
	return nil
}

func uintPtrOrNil(v uint) *uint {
	if v == 0 {
		return nil
	}
	return &v
}

func uintPtrMatches(expected *uint, actual uint) bool {
	switch {
	case expected == nil && actual == 0:
		return true
	case expected == nil || actual == 0:
		return false
	default:
		return *expected == actual
	}
}

func mustJSONString(v any) string {
	if v == nil {
		return ""
	}
	buf, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(buf)
}

func jsonMatches(expected string, actual map[string]any) bool {
	if strings.TrimSpace(expected) == "" {
		return len(actual) == 0
	}
	if len(actual) == 0 {
		return false
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return false
	}
	return expected == string(actualJSON)
}
