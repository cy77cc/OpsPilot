// Package logic provides business logic for AI service
package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"gorm.io/gorm"
)

const (
	confirmationStatusPending   = "pending"
	confirmationStatusConfirmed = "confirmed"
	confirmationStatusCancelled = "cancelled"
	confirmationStatusExpired   = "expired"
)

// ErrConfirmationNotFound is returned when confirmation is not found
var ErrConfirmationNotFound = errors.New("confirmation not found")

// ConfirmationService manages confirmation requests
type ConfirmationService struct {
	db                  *gorm.DB
	defaultTimeout      time.Duration
	pollInterval        time.Duration
	expireCheckInterval time.Duration
}

// ConfirmationRequestInput is the input for requesting confirmation
type ConfirmationRequestInput struct {
	RequestUserID uint64
	TraceID       string
	ToolName      string
	ToolMode      string
	RiskLevel     string
	ParamsJSON    string
	PreviewJSON   string
	Reason        string
	Timeout       time.Duration
}

// NewConfirmationService creates a new ConfirmationService
func NewConfirmationService(db *gorm.DB) *ConfirmationService {
	return &ConfirmationService{
		db:                  db,
		defaultTimeout:      5 * time.Minute,
		pollInterval:        300 * time.Millisecond,
		expireCheckInterval: 1 * time.Second,
	}
}

// RequestConfirmation creates a new confirmation request
func (s *ConfirmationService) RequestConfirmation(ctx context.Context, in ConfirmationRequestInput) (*model.ConfirmationRequest, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("confirmation service not initialized")
	}
	if strings.TrimSpace(in.ToolName) == "" {
		return nil, errors.New("tool name is required")
	}
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = s.defaultTimeout
	}
	now := time.Now()
	item := &model.ConfirmationRequest{
		ID:            fmt.Sprintf("confirm-%d", now.UnixNano()),
		RequestUserID: in.RequestUserID,
		TraceID:       strings.TrimSpace(in.TraceID),
		ToolName:      strings.TrimSpace(in.ToolName),
		ToolMode:      normalizeOrDefault(in.ToolMode, "mutating"),
		RiskLevel:     normalizeOrDefault(in.RiskLevel, "medium"),
		ParamsJSON:    strings.TrimSpace(in.ParamsJSON),
		PreviewJSON:   strings.TrimSpace(in.PreviewJSON),
		Status:        confirmationStatusPending,
		Reason:        strings.TrimSpace(in.Reason),
		ExpiresAt:     now.Add(timeout),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// WaitForConfirmation waits for confirmation response
func (s *ConfirmationService) WaitForConfirmation(ctx context.Context, id string) (*model.ConfirmationRequest, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("confirmation service not initialized")
	}
	confirmID := strings.TrimSpace(id)
	if confirmID == "" {
		return nil, errors.New("confirmation id is required")
	}
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	for {
		item, err := s.Get(ctx, confirmID)
		if err != nil {
			return nil, err
		}
		if item.Status == confirmationStatusPending && time.Now().After(item.ExpiresAt) {
			if _, err := s.markExpired(ctx, confirmID); err != nil {
				return nil, err
			}
			item, err = s.Get(ctx, confirmID)
			if err != nil {
				return nil, err
			}
		}
		if item.Status != confirmationStatusPending {
			return item, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// Confirm confirms a pending request
func (s *ConfirmationService) Confirm(ctx context.Context, id string) (*model.ConfirmationRequest, error) {
	return s.setDecision(ctx, id, confirmationStatusConfirmed)
}

// Cancel cancels a pending request
func (s *ConfirmationService) Cancel(ctx context.Context, id string) (*model.ConfirmationRequest, error) {
	return s.setDecision(ctx, id, confirmationStatusCancelled)
}

// Get retrieves a confirmation request
func (s *ConfirmationService) Get(ctx context.Context, id string) (*model.ConfirmationRequest, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("confirmation service not initialized")
	}
	var item model.ConfirmationRequest
	err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrConfirmationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ExpirePending expires all pending confirmations that have timed out
func (s *ConfirmationService) ExpirePending(ctx context.Context, now time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("confirmation service not initialized")
	}
	res := s.db.WithContext(ctx).Model(&model.ConfirmationRequest{}).
		Where("status = ? AND expires_at <= ?", confirmationStatusPending, now).
		Updates(map[string]any{
			"status":       confirmationStatusExpired,
			"cancelled_at": now,
			"updated_at":   now,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// StartTimeoutWatcher starts a background goroutine to expire pending confirmations
func (s *ConfirmationService) StartTimeoutWatcher(ctx context.Context) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(s.expireCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			case <-ticker.C:
				_, _ = s.ExpirePending(ctx, time.Now())
			}
		}
	}()
	return func() { <-done }
}

func (s *ConfirmationService) setDecision(ctx context.Context, id, status string) (*model.ConfirmationRequest, error) {
	confirmID := strings.TrimSpace(id)
	if confirmID == "" {
		return nil, errors.New("confirmation id is required")
	}
	now := time.Now()
	item, err := s.Get(ctx, confirmID)
	if err != nil {
		return nil, err
	}
	if item.Status != confirmationStatusPending {
		return nil, fmt.Errorf("confirmation already %s", item.Status)
	}
	if now.After(item.ExpiresAt) {
		return s.markExpired(ctx, confirmID)
	}
	updates := map[string]any{
		"status":     status,
		"updated_at": now,
	}
	switch status {
	case confirmationStatusConfirmed:
		updates["confirmed_at"] = now
	case confirmationStatusCancelled:
		updates["cancelled_at"] = now
	}
	res := s.db.WithContext(ctx).Model(&model.ConfirmationRequest{}).
		Where("id = ? AND status = ?", confirmID, confirmationStatusPending).
		Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	return s.Get(ctx, confirmID)
}

func (s *ConfirmationService) markExpired(ctx context.Context, id string) (*model.ConfirmationRequest, error) {
	confirmID := strings.TrimSpace(id)
	now := time.Now()
	res := s.db.WithContext(ctx).Model(&model.ConfirmationRequest{}).
		Where("id = ? AND status = ?", confirmID, confirmationStatusPending).
		Updates(map[string]any{
			"status":       confirmationStatusExpired,
			"cancelled_at": now,
			"updated_at":   now,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	return s.Get(ctx, confirmID)
}

func normalizeOrDefault(in, def string) string {
	v := strings.TrimSpace(in)
	if v == "" {
		return def
	}
	return v
}
