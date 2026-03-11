// Package ai 提供最终答案渲染功能。
//
// 本文件负责将执行计划和结果格式化为用户可读的最终答案。
// 答案包含标题、结论、关键依据、建议和可选的原始证据。
package ai

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/summarizer"
)

// finalAnswerRenderer 是最终答案渲染器。
type finalAnswerRenderer struct{}

// newFinalAnswerRenderer 创建新的最终答案渲染器。
func newFinalAnswerRenderer() *finalAnswerRenderer {
	return &finalAnswerRenderer{}
}

// Render 渲染最终答案，返回段落列表。
//
// 答案结构:
//   1. 标题 (Headline)
//   2. 结论 (Conclusion)
//   3. 关键依据 (KeyFindings + ResourceSummaries)
//   4. 建议 (Recommendations 或 NextActions)
//   5. 原始证据 (可选，根据 RawOutputPolicy)
//
// 参数:
//   - message: 用户原始消息 (当前未使用)
//   - plan: 执行计划 (当前未使用)
//   - result: 执行结果
//   - summaryOut: 总结输出
func (r *finalAnswerRenderer) Render(message string, plan *planner.ExecutionPlan, result *executor.Result, summaryOut summarizer.SummaryOutput) []string {
	_ = message
	_ = plan

	paragraphs := []string{}
	paragraphs = appendUniqueParagraph(paragraphs, sanitizeAnswerText(summaryOut.Headline))
	paragraphs = appendUniqueParagraph(paragraphs, sanitizeAnswerText(summaryOut.Conclusion))
	paragraphs = appendUniqueParagraph(paragraphs, sanitizeAnswerText(summaryOut.Narrative))
	paragraphs = appendUniqueParagraph(paragraphs, sanitizeAnswerText(summaryOut.Summary))

	findings := sanitizeLines(append(append([]string(nil), summaryOut.KeyFindings...), summaryOut.ResourceSummaries...))
	if len(findings) > 0 {
		paragraphs = appendUniqueParagraph(paragraphs, "- "+strings.Join(findings, "\n- "))
	}

	recommendations := sanitizeLines(summaryOut.Recommendations)
	if len(recommendations) == 0 {
		recommendations = sanitizeLines(summaryOut.NextActions)
	}
	if len(recommendations) > 0 {
		paragraphs = appendUniqueParagraph(paragraphs, "- "+strings.Join(recommendations, "\n- "))
	}

	if shouldIncludeEvidence(summaryOut) {
		evidence := collectEvidenceLines(result, 6)
		if len(evidence) > 0 {
			paragraphs = appendUniqueParagraph(paragraphs, "原始执行证据：\n- "+strings.Join(evidence, "\n- "))
		}
	}
	return compactParagraphs(paragraphs)
}

func appendUniqueParagraph(paragraphs []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return paragraphs
	}
	for _, existing := range paragraphs {
		if strings.TrimSpace(existing) == value {
			return paragraphs
		}
	}
	return append(paragraphs, value)
}

// shouldIncludeEvidence 判断是否应该包含原始证据。
// 根据 RawOutputPolicy 判断，"include_evidence" 或 "raw_evidence" 时包含。
func shouldIncludeEvidence(summaryOut summarizer.SummaryOutput) bool {
	policy := strings.ToLower(strings.TrimSpace(summaryOut.RawOutputPolicy))
	return policy == "include_evidence" || policy == "raw_evidence"
}

// collectEvidenceLines 从执行结果中收集证据文本。
// 限制返回数量以避免答案过长。
//
// 收集来源:
//   - 步骤摘要 (step.Summary)
//   - 观察到的事实 (evidence.Data["observed_facts"])
func collectEvidenceLines(result *executor.Result, limit int) []string {
	if result == nil || limit <= 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for _, step := range result.Steps {
		if summary := sanitizeAnswerText(step.Summary); summary != "" {
			out = append(out, summary)
			if len(out) >= limit {
				return dedupeStrings(out)
			}
		}
		for _, evidence := range step.Evidence {
			facts, ok := evidence.Data["observed_facts"].([]string)
			if ok {
				for _, fact := range facts {
					fact = sanitizeAnswerText(fact)
					if fact == "" {
						continue
					}
					out = append(out, fact)
					if len(out) >= limit {
						return dedupeStrings(out)
					}
				}
				continue
			}
			if rawFacts, ok := evidence.Data["observed_facts"].([]any); ok {
				for _, fact := range rawFacts {
					text := sanitizeAnswerText(asString(fact))
					if text == "" {
						continue
					}
					out = append(out, text)
					if len(out) >= limit {
						return dedupeStrings(out)
					}
				}
			}
		}
	}
	return dedupeStrings(out)
}

// sanitizeLines 清理字符串列表，移除空项和重复项。
func sanitizeLines(items []string) []string {
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		item = sanitizeAnswerText(item)
		if item == "" {
			continue
		}
		cleaned = append(cleaned, item)
	}
	return dedupeStrings(cleaned)
}

// sanitizeAnswerText 清理答案文本。
// 移除代码块、Markdown 格式和无意义的输出。
func sanitizeAnswerText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	if strings.Contains(text, "```") {
		return ""
	}
	if strings.Contains(lower, "完整输出如下") || strings.Contains(lower, "raw output") {
		return ""
	}
	if strings.Contains(lower, "filesystem") && strings.Contains(lower, "mounted on") {
		return ""
	}
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "***", "")
	text = strings.ReplaceAll(text, "**", "")
	return strings.TrimSpace(text)
}

// compactParagraphs 压缩段落列表，移除空项。
func compactParagraphs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		text := strings.TrimSpace(item)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

// dedupeStrings 去重字符串列表。
func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// asString 安全地将任意值转换为字符串。
func asString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return ""
	}
}
