package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	aiobs "github.com/cy77cc/OpsPilot/internal/ai/observability"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

func resolveExecutionUsage(exec *aitools.Execution) (*aiobs.Usage, map[string]any) {
	metadata := map[string]any{}
	if exec != nil && len(exec.Metadata) > 0 {
		for key, value := range exec.Metadata {
			metadata[key] = value
		}
	}

	if usage := normalizeUsage(execUsage(exec)); usage != nil {
		metadata["token_accounting_status"] = "reported"
		metadata["token_accounting_source"] = usage.Source
		metadata["usage"] = usageMap(usage)
		return usage, metadata
	}

	metadata["token_accounting_status"] = "unavailable"
	metadata["token_accounting_source"] = "current_runtime_api_unavailable"
	return nil, metadata
}

func execUsage(exec *aitools.Execution) *aiobs.Usage {
	if exec == nil {
		return nil
	}
	if exec.Usage != nil {
		return exec.Usage
	}
	if usage, ok := usageFromAny(exec.Metadata["usage"]); ok {
		return usage
	}
	if usage, ok := usageFromResult(exec.Result); ok {
		return usage
	}
	return nil
}

func usageFromResult(result any) (*aiobs.Usage, bool) {
	payload, ok := result.(map[string]any)
	if !ok {
		return nil, false
	}
	return usageFromAny(payload["usage"])
}

func usageFromAny(value any) (*aiobs.Usage, bool) {
	payload, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	usage := &aiobs.Usage{
		PromptTokens:     int64FromAny(firstNonNil(payload["prompt_tokens"], payload["promptTokens"])),
		CompletionTokens: int64FromAny(firstNonNil(payload["completion_tokens"], payload["completionTokens"])),
		TotalTokens:      int64FromAny(firstNonNil(payload["total_tokens"], payload["totalTokens"])),
		EstimatedCostUSD: float64FromAny(firstNonNil(payload["estimated_cost_usd"], payload["cost_usd"], payload["estimatedCostUSD"], payload["costUSD"])),
		Currency:         strings.TrimSpace(stringFromAny(payload["currency"])),
		Source:           strings.TrimSpace(stringFromAny(firstNonNil(payload["source"], payload["provider"]))),
	}
	if usage.TotalTokens <= 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if usage.TotalTokens <= 0 && usage.EstimatedCostUSD <= 0 {
		return nil, false
	}
	return usage, true
}

func normalizeUsage(usage *aiobs.Usage) *aiobs.Usage {
	if usage == nil {
		return nil
	}
	cloned := *usage
	if cloned.TotalTokens <= 0 {
		cloned.TotalTokens = cloned.PromptTokens + cloned.CompletionTokens
	}
	if strings.TrimSpace(cloned.Source) == "" {
		cloned.Source = "execution_metadata"
	}
	if cloned.TotalTokens <= 0 && cloned.EstimatedCostUSD <= 0 {
		return nil
	}
	return &cloned
}

func usageMap(usage *aiobs.Usage) map[string]any {
	if usage == nil {
		return nil
	}
	return map[string]any{
		"prompt_tokens":      usage.PromptTokens,
		"completion_tokens":  usage.CompletionTokens,
		"total_tokens":       usage.TotalTokens,
		"estimated_cost_usd": usage.EstimatedCostUSD,
		"currency":           usage.Currency,
		"source":             usage.Source,
	}
}

func updateExecutionRow(ctx context.Context, db *gorm.DB, execID string, updates map[string]any) {
	if db == nil || strings.TrimSpace(execID) == "" {
		return
	}
	_ = db.WithContext(ctx).Model(&model.AIExecution{}).Where("id = ?", execID).Updates(updates).Error
}

func finalizeExecutionRecord(ctx context.Context, db *gorm.DB, execution model.AIExecution, result *aitools.Execution, execErr error) map[string]any {
	finishedAt := time.Now().UTC()
	status := strings.TrimSpace(execution.Status)
	if status == "" || status == "running" {
		status = "success"
	}
	if execErr != nil {
		status = "failed"
	}
	durationMs := executionDurationMs(execution.StartedAt, finishedAt)
	usage, metadata := resolveExecutionUsage(result)
	resultJSON := mustJSON(resultValue(result))
	updates := map[string]any{
		"status":             status,
		"finished_at":        finishedAt,
		"duration_ms":        durationMs,
		"metadata_json":      mustJSON(metadata),
		"result_json":        resultJSON,
		"error_message":      "",
		"prompt_tokens":      int64(0),
		"completion_tokens":  int64(0),
		"total_tokens":       int64(0),
		"estimated_cost_usd": 0.0,
	}
	if execErr != nil {
		resultJSON = mustJSON(map[string]any{"error": execErr.Error()})
		updates["result_json"] = resultJSON
		updates["error_message"] = execErr.Error()
	}
	if usage != nil {
		updates["prompt_tokens"] = usage.PromptTokens
		updates["completion_tokens"] = usage.CompletionTokens
		updates["total_tokens"] = usage.TotalTokens
		updates["estimated_cost_usd"] = usage.EstimatedCostUSD
	}
	updateExecutionRow(ctx, db, execution.ID, updates)
	logExecutionOutcome(execution, status, durationMs, updates["error_message"].(string), resultJSON, metadata)
	aiobs.ObserveToolExecution(aiobs.ExecutionRecord{
		Scene:     execution.Scene,
		ToolName:  execution.ToolName,
		ToolMode:  execution.ToolMode,
		RiskLevel: execution.RiskLevel,
		Status:    status,
		Duration:  time.Duration(durationMs) * time.Millisecond,
		Usage:     usage,
	})
	return updates
}

func logExecutionOutcome(execution model.AIExecution, status string, durationMs int64, errMsg string, resultJSON string, metadata map[string]any) {
	l := logger.L()
	if l == nil {
		return
	}
	fields := []logger.Field{
		{Key: "execution_id", Value: execution.ID},
		{Key: "session_id", Value: execution.SessionID},
		{Key: "plan_id", Value: execution.PlanID},
		{Key: "step_id", Value: execution.StepID},
		{Key: "tool", Value: execution.ToolName},
		{Key: "scene", Value: execution.Scene},
		{Key: "status", Value: status},
		{Key: "duration_ms", Value: durationMs},
		{Key: "params", Value: truncateJSON(execution.ParamsJSON)},
		{Key: "result", Value: truncateJSON(resultJSON)},
		{Key: "metadata", Value: metadata},
	}
	if strings.TrimSpace(errMsg) != "" {
		fields = append(fields, logger.Field{Key: "error_message", Value: errMsg})
		l.Warn("ai execution completed with error", fields...)
		return
	}
	l.Info("ai execution completed", fields...)
}

func resultValue(exec *aitools.Execution) any {
	if exec == nil {
		return nil
	}
	return exec.Result
}

func executionDurationMs(startedAt *time.Time, finishedAt time.Time) int64 {
	if startedAt == nil {
		return 0
	}
	if finishedAt.Before(*startedAt) {
		return 0
	}
	return finishedAt.Sub(*startedAt).Milliseconds()
}

func truncateJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) <= 2048 {
		return raw
	}
	return raw[:2048] + "...(truncated)"
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func int64FromAny(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		out, _ := v.Int64()
		return out
	default:
		return 0
	}
}

func float64FromAny(value any) float64 {
	switch v := value.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		out, _ := v.Float64()
		return out
	default:
		return 0
	}
}
