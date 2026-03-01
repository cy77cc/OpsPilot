package ai

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handler) listSessions(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	scene := c.Query("scene")
	httpx.OK(c, h.store.listSessions(uid, scene))
}

func (h *handler) currentSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	scene := c.Query("scene")
	session, found := h.store.currentSession(uid, scene)
	if !found {
		httpx.OK(c, nil)
		return
	}
	httpx.OK(c, session)
}

func (h *handler) getSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	session, found := h.store.getSession(uid, c.Param("id"))
	if !found {
		httpx.Fail(c, xcode.NotFound, "session not found")
		return
	}
	httpx.OK(c, session)
}

func (h *handler) deleteSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	h.store.deleteSession(uid, c.Param("id"))
	httpx.OK(c, nil)
}

func (h *handler) updateSessionTitle(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	title := normalizeSessionTitle(req.Title)
	if title == "" {
		httpx.Fail(c, xcode.ParamError, "title is required")
		return
	}
	session, err := h.store.updateSessionTitle(uid, c.Param("id"), title)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.Fail(c, xcode.NotFound, "session not found")
			return
		}
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, session)
}

func (h *handler) analyze(c *gin.Context) {
	var req map[string]any
	_ = c.ShouldBindJSON(&req)
	summary := "MVP阶段分析能力已启用"
	if h.svcCtx.AI != nil && h.svcCtx.AI.Model != nil {
		msg, err := h.svcCtx.AI.Model.Generate(c.Request.Context(), []*schema.Message{
			schema.UserMessage("请根据输入生成简短运维分析摘要，最多120字：" + toString(req)),
		})
		if err == nil && msg != nil && strings.TrimSpace(msg.Content) != "" {
			summary = msg.Content
		}
	}
	httpx.OK(c, gin.H{
		"id":        "ana-" + strconvFormatInt(time.Now().UnixNano()),
		"type":      "generic",
		"title":     "AI 分析结果",
		"summary":   summary,
		"details":   req,
		"createdAt": time.Now(),
	})
}

func (h *handler) recommendations(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Type    string         `json:"type"`
		Context map[string]any `json:"context"`
		Limit   int            `json:"limit"`
	}
	_ = c.ShouldBindJSON(&req)
	scene := normalizeScene(toString(req.Context["scene"]))
	if scene == "global" {
		scene = normalizeScene(toString(req.Context["page"]))
	}
	if req.Limit <= 0 {
		req.Limit = 5
	}
	existing := h.store.getRecommendations(uid, scene, req.Limit)
	if len(existing) == 0 {
		existing = []recommendationRecord{{
			ID:        "rec-" + strconvFormatInt(time.Now().UnixNano()),
			UserID:    uid,
			Scene:     scene,
			Type:      "suggestion",
			Title:     "通用建议",
			Content:   "先执行只读诊断，再进行变更操作，并确认回滚方案。",
			Relevance: 0.7,
			CreatedAt: time.Now(),
		}}
	}
	out := make([]gin.H, 0, len(existing))
	for _, r := range existing {
		out = append(out, gin.H{
			"id":              r.ID,
			"type":            r.Type,
			"title":           r.Title,
			"content":         r.Content,
			"followup_prompt": r.FollowupPrompt,
			"reasoning":       r.Reasoning,
			"relevance":       r.Relevance,
		})
	}
	httpx.OK(c, out)
}

func (h *handler) k8sAnalyze(c *gin.Context) {
	httpx.OK(c, gin.H{
		"insights": []string{"建议优先检查异常 Pod 的重启次数和事件。"},
		"risks":    []string{"高峰时段直接变更副本可能引发抖动。"},
	})
}

func (h *handler) actionPreview(c *gin.Context) {
	var req struct {
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	_ = c.ShouldBindJSON(&req)
	httpx.OK(c, gin.H{
		"approval_token": "approve-" + strconvFormatInt(time.Now().UnixNano()),
		"intent":         req.Action,
		"risk":           "medium",
		"params":         req.Params,
		"previewDiff":    "MVP preview",
	})
}

func (h *handler) actionExecute(c *gin.Context) {
	var req struct {
		ApprovalToken string `json:"approval_token"`
	}
	_ = c.ShouldBindJSON(&req)
	httpx.OK(c, gin.H{
		"approval_token": req.ApprovalToken,
		"status":         "executed",
	})
}

func (h *handler) refreshSuggestions(uid uint64, scene, answer string) []recommendationRecord {
	scene = normalizeScene(scene)
	prompt := "你是 suggestion 智能体。基于下面回答提炼 3 条可执行建议，每条一行，格式为：标题|内容|相关度(0-1)|思考摘要（不超过60字）。回答内容如下：\n" + answer
	out := []recommendationRecord{}
	if h.svcCtx.AI != nil {
		msg, err := h.svcCtx.AI.Generate(context.Background(), []*schema.Message{schema.UserMessage(prompt)})
		if err == nil && msg != nil {
			lines := strings.Split(msg.Content, "\n")
			for _, line := range lines {
				trim := strings.TrimSpace(line)
				if trim == "" {
					continue
				}
				parts := strings.SplitN(trim, "|", 4)
				if len(parts) < 2 {
					continue
				}
				rel := 0.7
				if len(parts) >= 3 {
					if v, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil {
						rel = v
					}
				}
				reasoning := ""
				if len(parts) == 4 {
					reasoning = strings.TrimSpace(parts[3])
				}
				out = append(out, recommendationRecord{
					ID:             "rec-" + strconvFormatInt(time.Now().UnixNano()),
					UserID:         uid,
					Scene:          scene,
					Type:           "suggestion",
					Title:          strings.TrimSpace(parts[0]),
					Content:        strings.TrimSpace(parts[1]),
					FollowupPrompt: strings.TrimSpace(parts[1]),
					Reasoning:      reasoning,
					Relevance:      rel,
					CreatedAt:      time.Now(),
				})
			}
		}
	}
	if len(out) == 0 {
		out = append(out, recommendationRecord{
			ID:             "rec-" + strconvFormatInt(time.Now().UnixNano()),
			UserID:         uid,
			Scene:          scene,
			Type:           "suggestion",
			Title:          "先做健康检查",
			Content:        "优先检查资源/日志，再进行部署或配置变更。",
			FollowupPrompt: "先帮我做一次资源健康检查，然后再给变更建议。",
			Reasoning:      "先确认现状可降低误操作风险，再执行变更更稳妥。",
			Relevance:      0.7,
			CreatedAt:      time.Now(),
		})
	}
	h.store.setRecommendations(uid, scene, out)
	return out
}
