package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

func (l *Logic) SubmitApproval(ctx context.Context, input SubmitApprovalInput) (*SubmitApprovalOutput, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil, fmt.Errorf("approval service not initialized")
	}
	return NewApprovalWriteModel(l.svcCtx.DB).SubmitApproval(ctx, input)
}

func (l *Logic) RetryResumeApproval(ctx context.Context, input RetryResumeApprovalInput) (*RetryResumeApprovalOutput, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil, fmt.Errorf("approval service not initialized")
	}
	return NewApprovalWriteModel(l.svcCtx.DB).RetryResumeApproval(ctx, input)
}

// ResumeApprovalInput 恢复审批执行的输入参数。
type ResumeApprovalInput struct {
	SessionID  string
	ApprovalID string
	Approved   bool
	Reason     string
	Comment    string
	UserID     uint64
}

// ResumeApproval 恢复审批执行（SSE 流式）。
//
// 该方法通过 Runner.ResumeWithParams 恢复 AI Agent 执行，
// 并通过 SSE 流式返回后续执行结果。
//
// Deprecated: approval recovery must go through SubmitApproval/RetryResumeApproval + ApprovalWorker.
func (l *Logic) ResumeApproval(ctx context.Context, input ResumeApprovalInput, emit EventEmitter) error {
	if l.ApprovalDAO == nil || l.CheckpointStore == nil || l.AIRouter == nil {
		emit(airuntime.NewErrorEvent("", fmt.Errorf("AI service not initialized")).Event, nil)
		return nil
	}

	// 获取审批任务
	task, err := l.ApprovalDAO.GetByApprovalID(ctx, input.ApprovalID)
	if err != nil {
		return fmt.Errorf("get approval task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("approval task not found")
	}

	// 验证用户权限
	resumeScene := ""
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, task.SessionID, input.UserID, "")
		if err != nil {
			return fmt.Errorf("verify session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session not found or no permission")
		}
		resumeScene = normalizeScene(session.Scene)
	}

	// 更新审批状态
	if task.Status == "pending" {
		status := "approved"
		if !input.Approved {
			status = "rejected"
		}
		if err := l.ApprovalDAO.UpdateStatus(ctx, input.ApprovalID, status, input.UserID, input.Reason, input.Comment); err != nil {
			return fmt.Errorf("update approval status: %w", err)
		}
	}

	// 构建恢复参数
	approvalResult := map[string]any{
		"approved":          input.Approved,
		"disapprove_reason": input.Reason,
		"comment":           input.Comment,
		"approved_by":       input.UserID,
		"approved_at":       time.Now().Format(time.RFC3339),
	}

	// 发送 meta 事件
	meta := airuntime.NewMetaEvent(task.SessionID, task.RunID, 1)
	emit(meta.Event, meta.Data)

	// 创建 Runner 并恢复执行
	ctx = l.runtimeContext(ctx)
	ctx = runtimectx.WithAIMetadata(ctx, runtimectx.AIMetadata{
		SessionID:    task.SessionID,
		RunID:        task.RunID,
		CheckpointID: task.CheckpointID,
		UserID:       input.UserID,
		Scene:        resumeScene,
	})
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           l.AIRouter,
		EnableStreaming: true,
		CheckPointStore: l.CheckpointStore,
	})

	// 使用 ResumeWithParams 恢复执行
	resumeParams := &adk.ResumeParams{
		Targets: map[string]any{
			task.ToolCallID: approvalResult,
		},
	}

	iterator, err := runner.ResumeWithParams(ctx, task.CheckpointID, resumeParams, adk.WithCheckPointID(task.CheckpointID))
	if err != nil {
		return fmt.Errorf("resume execution: %w", err)
	}

	projector := airuntime.NewStreamProjector()
	result, err := processAgentIterator(ctx, iteratorProcessInput{
		Iterator:  iterator,
		Projector: projector,
		Emit:      emit,
	})
	if err != nil {
		return err
	}
	if result.FatalErr != nil {
		projected := projector.Fail(task.RunID, result.FatalErr)
		emit(projected.Event, projected.Data)
		return nil
	}
	if result.Interrupted {
		return nil
	}
	done := projector.Finish(task.RunID)
	if payload, ok := done.Data.(map[string]any); ok {
		ensureDoneSummary(payload, result.SummaryText, result.HasToolErrors)
		done.Data = payload
	}
	emit(done.Event, done.Data)

	return nil
}

// GetApproval 获取审批详情。
func (l *Logic) GetApproval(ctx context.Context, approvalID string, userID uint64) (*ai.AIApprovalTask, error) {
	if l.ApprovalDAO == nil {
		return nil, nil
	}

	task, err := l.ApprovalDAO.GetByApprovalID(ctx, approvalID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, nil
	}

	// 验证用户权限
	if l.ChatDAO != nil && task.SessionID != "" {
		session, err := l.ChatDAO.GetSession(ctx, task.SessionID, userID, "")
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, nil
		}
	}

	return task, nil
}

// ListPendingApprovals 列出用户的待处理审批。
func (l *Logic) ListPendingApprovals(ctx context.Context, userID uint64) ([]ai.AIApprovalTask, error) {
	if l.ApprovalDAO == nil {
		return []ai.AIApprovalTask{}, nil
	}

	return l.ApprovalDAO.ListPendingByUserID(ctx, userID, 50)
}
