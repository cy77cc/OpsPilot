// Package knowledge provides AI knowledge base functions
package knowledge

import "strings"

// HelpTopic represents a help topic
type HelpTopic struct {
	ID       string
	Title    string
	Keywords []string
	Content  string
}

// HelpTopics contains predefined help topics
var HelpTopics = []HelpTopic{
	{
		ID:       "K1",
		Title:    "平台与登录",
		Keywords: []string{"登录", "掉线", "401", "403", "权限不足", "项目切换"},
		Content:  "登录异常优先区分认证问题（401）与授权问题（403）；登录后先切换项目上下文再执行操作。",
	},
	{
		ID:       "K2",
		Title:    "主机纳管",
		Keywords: []string{"主机", "纳管", "探测", "ssh", "连接", "在线"},
		Content:  "路径：主机管理 -> 主机纳管。步骤：填写连接信息 -> 连通性探测 -> 提交纳管 -> 在线确认。常见失败：网络不通、凭据错误、账号权限不足。",
	},
	{
		ID:       "K3",
		Title:    "告警处置",
		Keywords: []string{"告警", "critical", "监控", "报警", "健康度", "异常"},
		Content:  "先按严重级别排序（critical 优先），再看触发指标、对象与时间窗，随后执行只读诊断（CPU/内存/磁盘/日志）。",
	},
	{
		ID:       "K4",
		Title:    "服务发布",
		Keywords: []string{"发布", "上线", "回滚", "服务", "变更", "部署"},
		Content:  "标准流程：预览 -> 风险确认 -> 执行 -> 观测 -> 回滚（必要时）。生产高风险变更需二次确认。",
	},
	{
		ID:       "K5",
		Title:    "配置中心",
		Keywords: []string{"配置", "diff", "回滚", "配置中心", "发布配置"},
		Content:  "流程：编辑草稿 -> Diff 确认 -> 发布。异常时回滚到历史版本。",
	},
	{
		ID:       "K6",
		Title:    "任务管理",
		Keywords: []string{"任务", "cron", "定时", "执行历史", "自动化"},
		Content:  "流程：创建任务 -> 配置 Cron -> 测试验证 -> 观察执行历史并处理失败。",
	},
	{
		ID:       "K7",
		Title:    "权限治理",
		Keywords: []string{"权限", "rbac", "角色", "用户", "授权", "审计"},
		Content:  "遵循最小权限：先角色后用户，默认只读按需授权，生产写操作建议审批 + 审计。",
	},
	{
		ID:       "K8",
		Title:    "AI 助手使用",
		Keywords: []string{"ai", "助手", "提问", "怎么问", "智能", "copilot"},
		Content:  "提问建议：诊断类先要只读步骤；变更类先要预览与回滚方案；值班类要求 10 分钟可执行计划。",
	},
	{
		ID:       "K9",
		Title:    "值班应急",
		Keywords: []string{"事故", "应急", "值班", "恢复", "故障", "升级"},
		Content:  "15分钟处置：0-5分钟确认影响并冻结高风险变更，5-10分钟采集指标日志事件，10-15分钟执行回滚/降级/限流。",
	},
}

// BuildHelpKnowledgeDirective builds a help directive for the given message
func BuildHelpKnowledgeDirective(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return ""
	}
	if !IsHelpIntent(msg) {
		return ""
	}

	faqBest, _ := MatchFAQKnowledge(msg)

	var best *HelpTopic
	bestScore := 0
	for i := range HelpTopics {
		topic := &HelpTopics[i]
		score := 0
		for _, kw := range topic.Keywords {
			if strings.Contains(msg, strings.ToLower(kw)) {
				score++
			}
		}
		if score > bestScore {
			best = topic
			bestScore = score
		}
	}

	if best == nil && faqBest == nil {
		return "帮助回答要求：先给入口路径，再给分步操作；涉及变更时必须补充风险与回滚建议；涉及实时状态时提醒需查询实时数据确认。"
	}

	sections := []string{"帮助知识（用于回答当前问题）:"}
	if best != nil {
		sections = append(sections,
			"- 主题: "+best.Title+" ("+best.ID+")",
			"- 内容: "+best.Content,
		)
	}
	if faqBest != nil {
		sections = append(sections,
			"- FAQ参考: "+faqBest.ID,
			"- FAQ问题: "+faqBest.Question,
			"- FAQ答案: "+faqBest.Answer,
		)
	}
	sections = append(sections, "- 回答要求: 先给入口路径，再给分步操作；涉及变更必须补充风险与回滚建议；涉及实时状态需提醒调用实时数据确认。")
	return strings.Join(sections, "\n")
}

// IsHelpIntent checks if the message indicates a help intent
func IsHelpIntent(msg string) bool {
	markers := []string{
		"如何", "怎么", "帮助", "文档", "指南", "手册", "步骤", "在哪", "入口", "怎么做",
		"排查", "处理", "登录失败", "告警", "发布", "回滚", "rbac", "faq", "值班", "应急",
	}
	for _, marker := range markers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
