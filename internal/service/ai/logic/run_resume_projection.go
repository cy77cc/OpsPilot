package logic

import (
	"context"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

type ResumableCredentials struct {
	RunID           string `json:"run_id,omitempty"`
	ClientRequestID string `json:"client_request_id,omitempty"`
	LatestEventID   string `json:"latest_event_id,omitempty"`
	ApprovalID      string `json:"approval_id,omitempty"`
	Resumable       bool   `json:"resumable"`
}

func (l *Logic) BuildResumableCredentials(ctx context.Context, run *model.AIRun) (*ResumableCredentials, error) {
	if run == nil || !isTailOpenStatus(run.Status) {
		return nil, nil
	}

	creds := &ResumableCredentials{
		RunID:           strings.TrimSpace(run.ID),
		ClientRequestID: strings.TrimSpace(run.ClientRequestID),
		Resumable:       true,
	}
	if creds.ClientRequestID == "" {
		creds.ClientRequestID = creds.RunID
	}

	if l != nil && l.RunEventDAO != nil {
		events, err := l.RunEventDAO.ListByRun(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		if len(events) > 0 {
			creds.LatestEventID = strings.TrimSpace(events[len(events)-1].ID)
		}
	}

	approvalID, err := l.lookupPendingApprovalID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	creds.ApprovalID = approvalID

	return creds, nil
}

func (l *Logic) lookupPendingApprovalID(ctx context.Context, runID string) (string, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil || strings.TrimSpace(runID) == "" {
		return "", nil
	}

	var task model.AIApprovalTask
	err := l.svcCtx.DB.WithContext(ctx).
		Where("run_id = ? AND status = ?", strings.TrimSpace(runID), "pending").
		Order("created_at DESC, id DESC").
		First(&task).Error
	if err == nil {
		return strings.TrimSpace(task.ApprovalID), nil
	}
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	return "", err
}
