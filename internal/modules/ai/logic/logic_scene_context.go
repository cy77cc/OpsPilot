package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

// buildSessionTitle 从首条消息生成会话标题。
func buildSessionTitle(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "New AI session"
	}
	return truncateString(trimmed, 48)
}

func normalizeScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return "ai"
	}
	return scene
}

func resolveChatScene(requestScene string, session *ai.AIChatSession) string {
	if strings.TrimSpace(requestScene) != "" {
		return normalizeScene(requestScene)
	}
	if session != nil && strings.TrimSpace(session.Scene) != "" {
		return normalizeScene(session.Scene)
	}
	return "ai"
}

func (l *Logic) buildAugmentedMessage(ctx context.Context, scene string, sceneContext map[string]any, message string) string {
	scene = normalizeScene(scene)
	sections := []string{
		"[Hidden platform context for routing, tool selection, and safety policy]",
		"[Scene]",
		fmt.Sprintf("scene=%s", scene),
	}

	if payload := stringifyJSON(sceneContext); payload != "" && payload != "{}" {
		sections = append(sections,
			"",
			"[Scene Context]",
			fmt.Sprintf("scene_context=%s", payload),
		)
	}

	sceneSections := l.loadSceneAugmentation(ctx, scene)
	if len(sceneSections) > 0 {
		for _, section := range sceneSections {
			if len(section) == 0 {
				continue
			}
			sections = append(sections, "", strings.Join(section, "\n"))
		}
	}

	sections = append(sections,
		"",
		fmt.Sprintf("User request:\n%s", strings.TrimSpace(message)),
	)

	return strings.Join(sections, "\n")
}

func (l *Logic) loadSceneAugmentation(ctx context.Context, scene string) [][]string {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil || strings.TrimSpace(scene) == "" {
		return nil
	}

	var prompts []ai.AIScenePrompt
	_ = l.svcCtx.DB.WithContext(ctx).
		Where("scene = ? AND is_active = ?", scene, true).
		Order("display_order ASC, id ASC").
		Find(&prompts).Error

	var config ai.AISceneConfig
	hasConfig := l.svcCtx.DB.WithContext(ctx).
		Where("scene = ?", scene).
		First(&config).Error == nil

	sceneLines := make([]string, 0, 4)
	if len(prompts) > 0 {
		promptTexts := make([]string, 0, len(prompts))
		for _, item := range prompts {
			if text := strings.TrimSpace(item.PromptText); text != "" {
				promptTexts = append(promptTexts, text)
			}
		}
		if len(promptTexts) > 0 {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_prompts=%s", stringifyJSON(promptTexts)))
		}
	}

	if hasConfig {
		if description := strings.TrimSpace(config.Description); description != "" {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_description=%s", description))
		}
		if constraints := compactJSONString(config.ConstraintsJSON); constraints != "" {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_constraints=%s", constraints))
		}
	}

	sections := make([][]string, 0, 2)
	if len(sceneLines) > 0 {
		sections = append(sections, append([]string{"[Scene Prompts & Constraints]"}, sceneLines...))
	}

	toolLines := make([]string, 0, 3)
	if hasConfig {
		if allowed := compactJSONString(config.AllowedToolsJSON); allowed != "" {
			toolLines = append(toolLines, fmt.Sprintf("allowed_tools=%s", allowed))
		}
		if blocked := compactJSONString(config.BlockedToolsJSON); blocked != "" {
			toolLines = append(toolLines, fmt.Sprintf("blocked_tools=%s", blocked))
		}
	}
	if len(toolLines) > 0 {
		toolLines = append(toolLines, "These tool constraints are mandatory.")
		sections = append(sections, append([]string{"[Tool Constraints]"}, toolLines...))
	}

	return sections
}

func stringifyJSON(value any) string {
	if value == nil {
		return ""
	}
	b, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(b)
}

func compactJSONString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var payload any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return value
	}
	return stringifyJSON(payload)
}

// truncateString 截断字符串到指定长度。
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	return string([]rune(s)[:maxLen])
}

// =============================================================================
// 审批相关方法
// =============================================================================

// SubmitApprovalInput 提交审批结果的输入参数。
type SubmitApprovalInput struct {
	ApprovalID       string
	Approved         bool
	DisapproveReason string
	Comment          string
	UserID           uint64
}

// SubmitApprovalOutput 提交审批结果的输出。
type SubmitApprovalOutput struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

type RetryResumeApprovalInput struct {
	ApprovalID string
	TriggerID  string
	UserID     uint64
}

type RetryResumeApprovalOutput struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}
