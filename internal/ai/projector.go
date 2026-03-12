package ai

import (
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
)

type turnProjector struct {
	emit      StreamEmitter
	meta      events.EventMeta
	enabled   bool
	turnID    string
	nextPos   int
	openBlock map[string]bool
}

func newTurnProjector(emit StreamEmitter, meta events.EventMeta, rollout RolloutConfig) *turnProjector {
	turnID := strings.TrimSpace(meta.TurnID)
	if turnID == "" {
		return &turnProjector{emit: emit, meta: meta, enabled: false}
	}
	return &turnProjector{
		emit:      emit,
		meta:      meta,
		enabled:   rollout.TurnBlockStreamingEnabled(),
		turnID:    turnID,
		openBlock: map[string]bool{},
	}
}

func (p *turnProjector) Start(phase string) {
	if !p.enabled {
		return
	}
	meta := p.meta
	meta.TurnID = p.turnID
	emitEvent(p.emit, events.TurnStarted, meta, map[string]any{
		"turn_id": p.turnID,
		"role":    "assistant",
		"phase":   strings.TrimSpace(phase),
		"status":  "streaming",
	})
	p.SetState("streaming", phase)
}

func (p *turnProjector) SetState(status, phase string) {
	if !p.enabled {
		return
	}
	meta := p.meta
	meta.TurnID = p.turnID
	emitEvent(p.emit, events.TurnState, meta, map[string]any{
		"turn_id": p.turnID,
		"status":  strings.TrimSpace(status),
		"phase":   strings.TrimSpace(phase),
	})
}

func (p *turnProjector) Done(status, phase string) {
	if !p.enabled {
		return
	}
	p.SetState(status, phase)
	meta := p.meta
	meta.TurnID = p.turnID
	emitEvent(p.emit, events.TurnDone, meta, map[string]any{
		"turn_id": p.turnID,
		"status":  strings.TrimSpace(status),
		"phase":   strings.TrimSpace(phase),
	})
}

func (p *turnProjector) ensureBlock(blockID, blockType, phase, status, title string, payload map[string]any) {
	if !p.enabled || blockID == "" || p.openBlock[blockID] {
		return
	}
	p.nextPos++
	meta := p.meta
	meta.TurnID = p.turnID
	meta.BlockID = blockID
	emitEvent(p.emit, events.BlockOpen, meta, map[string]any{
		"turn_id":    p.turnID,
		"block_id":   blockID,
		"block_type": strings.TrimSpace(blockType),
		"position":   p.nextPos,
		"status":     strings.TrimSpace(status),
		"phase":      strings.TrimSpace(phase),
		"title":      strings.TrimSpace(title),
		"payload":    cloneMap(payload),
	})
	p.openBlock[blockID] = true
}

func (p *turnProjector) delta(blockID, blockType, phase, status, title string, patch map[string]any) {
	if !p.enabled || blockID == "" {
		return
	}
	p.ensureBlock(blockID, blockType, phase, status, title, nil)
	meta := p.meta
	meta.TurnID = p.turnID
	meta.BlockID = blockID
	emitEvent(p.emit, events.BlockDelta, meta, map[string]any{
		"turn_id":  p.turnID,
		"block_id": blockID,
		"patch":    cloneMap(patch),
	})
}

func (p *turnProjector) replace(blockID, blockType, phase, status, title string, payload map[string]any) {
	if !p.enabled || blockID == "" {
		return
	}
	p.ensureBlock(blockID, blockType, phase, status, title, payload)
	meta := p.meta
	meta.TurnID = p.turnID
	meta.BlockID = blockID
	emitEvent(p.emit, events.BlockReplace, meta, map[string]any{
		"turn_id":  p.turnID,
		"block_id": blockID,
		"payload":  cloneMap(payload),
	})
}

func (p *turnProjector) close(blockID, status string) {
	if !p.enabled || blockID == "" || !p.openBlock[blockID] {
		return
	}
	meta := p.meta
	meta.TurnID = p.turnID
	meta.BlockID = blockID
	emitEvent(p.emit, events.BlockClose, meta, map[string]any{
		"turn_id":  p.turnID,
		"block_id": blockID,
		"status":   strings.TrimSpace(status),
	})
	delete(p.openBlock, blockID)
}

func (p *turnProjector) StageDelta(stage, status, chunk, stepID, expert string) {
	stage = strings.TrimSpace(stage)
	chunk = strings.TrimSpace(chunk)
	if !p.enabled || stage == "" || chunk == "" {
		return
	}
	blockID := fmt.Sprintf("status:%s", stage)
	title := projectorStageTitle(stage)
	patch := map[string]any{
		"stage":         stage,
		"status":        strings.TrimSpace(status),
		"content_chunk": chunk,
	}
	if stepID != "" {
		patch["step_id"] = stepID
	}
	if expert != "" {
		patch["expert"] = expert
	}
	p.delta(blockID, "status", stage, status, title, patch)
	if status == "success" || status == "error" {
		p.close(blockID, status)
	}
}

func (p *turnProjector) TextDelta(blockID, blockType, phase, chunk string) {
	if !p.enabled || chunk == "" {
		return
	}
	p.delta(blockID, blockType, phase, "streaming", "", map[string]any{
		"content_chunk": chunk,
	})
}

func (p *turnProjector) CloseText(blockID, status string) {
	p.close(blockID, status)
}

func (p *turnProjector) Plan(decisionSummary string, payload map[string]any) {
	if !p.enabled {
		return
	}
	data := cloneMap(payload)
	if strings.TrimSpace(decisionSummary) != "" {
		data["summary"] = strings.TrimSpace(decisionSummary)
	}
	p.replace("plan:main", "plan", "plan", "success", "执行计划", data)
}

func (p *turnProjector) ExecutionEvent(name string, meta events.EventMeta, data map[string]any) {
	if !p.enabled {
		return
	}
	switch name {
	case string(events.StepUpdate):
		stepID := stringValue(data["step_id"])
		if stepID == "" {
			stepID = strings.TrimSpace(meta.StepID)
		}
		p.replace(fmt.Sprintf("step:%s", stepID), "status", "execute", stageStatusFromValue(data["status"]), firstNonEmpty(stringValue(data["title"]), "执行步骤"), map[string]any{
			"step_id":              stepID,
			"plan_id":              firstNonEmpty(stringValue(data["plan_id"]), strings.TrimSpace(meta.PlanID)),
			"status":               stringValue(data["status"]),
			"title":                stringValue(data["title"]),
			"expert":               stringValue(data["expert"]),
			"user_visible_summary": stringValue(data["user_visible_summary"]),
			"error_code":           stringValue(data["error_code"]),
			"error_message":        stringValue(data["error_message"]),
		})
		switch stageStatusFromValue(data["status"]) {
		case "success", "error":
			p.close(fmt.Sprintf("step:%s", stepID), stageStatusFromValue(data["status"]))
		}
	case string(events.ToolCall):
		p.SetState("streaming", "execute")
		callID := firstNonEmpty(stringValue(data["call_id"]), fmt.Sprintf("tool:%s", firstNonEmpty(stringValue(data["tool_name"]), stringValue(data["expert"]))))
		p.replace(fmt.Sprintf("tool:%s", callID), "tool", "execute", "running", firstNonEmpty(stringValue(data["tool_name"]), stringValue(data["expert"]), "工具调用"), cloneMap(data))
	case string(events.ToolResult):
		p.SetState("streaming", "execute")
		callID := firstNonEmpty(stringValue(data["call_id"]), fmt.Sprintf("tool:%s", firstNonEmpty(stringValue(data["tool_name"]), stringValue(data["expert"]))))
		status := firstNonEmpty(stringValue(data["status"]), "success")
		p.replace(fmt.Sprintf("tool:%s", callID), "tool", "execute", status, firstNonEmpty(stringValue(data["tool_name"]), stringValue(data["expert"]), "工具结果"), cloneMap(data))
		p.close(fmt.Sprintf("tool:%s", callID), status)
	case string(events.ApprovalRequired):
		p.SetState("waiting_user", "execute")
		stepID := firstNonEmpty(stringValue(data["step_id"]), strings.TrimSpace(meta.StepID))
		p.replace(fmt.Sprintf("approval:%s", stepID), "approval", "execute", "waiting_user", firstNonEmpty(stringValue(data["title"]), "等待审批"), cloneMap(data))
	case string(events.Error):
		p.SetState("error", "summary")
		p.replace("error:active", "error", "summary", "error", "执行错误", cloneMap(data))
	}
}

func projectorStageTitle(stage string) string {
	switch strings.TrimSpace(stage) {
	case "rewrite":
		return "理解你的问题"
	case "plan":
		return "整理排查计划"
	case "execute":
		return "调用专家执行"
	case "summary":
		return "整理最终回答"
	default:
		return "AI 处理进度"
	}
}

func (p *turnProjector) ActiveBlockIDs() []string {
	if p == nil || len(p.openBlock) == 0 {
		return nil
	}
	out := make([]string, 0, len(p.openBlock))
	for blockID := range p.openBlock {
		out = append(out, blockID)
	}
	return out
}
