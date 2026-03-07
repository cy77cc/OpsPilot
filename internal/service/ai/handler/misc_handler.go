package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *AIHandler) listSessions(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	scene := c.Query("scene")
	httpx.OK(c, h.sessions.ListSessions(uid, scene))
}

func (h *AIHandler) currentSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	scene := c.Query("scene")
	session, found := h.sessions.CurrentSession(uid, scene)
	if !found {
		httpx.OK(c, nil)
		return
	}
	httpx.OK(c, session)
}

func (h *AIHandler) getSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	session, found := h.sessions.GetSession(uid, c.Param("id"))
	if !found {
		httpx.Fail(c, xcode.NotFound, "session not found")
		return
	}
	httpx.OK(c, session)
}

func (h *AIHandler) branchSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		MessageID string `json:"messageId"`
		Title     string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	session, err := h.sessions.BranchSession(uid, c.Param("id"), req.MessageID, req.Title)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			httpx.Fail(c, xcode.NotFound, "session not found")
		case strings.Contains(err.Error(), "anchor message not found"):
			httpx.Fail(c, xcode.ParamError, "anchor message not found")
		default:
			httpx.Fail(c, xcode.ServerError, err.Error())
		}
		return
	}
	httpx.OK(c, session)
}

func (h *AIHandler) deleteSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	h.sessions.DeleteSession(uid, c.Param("id"))
	httpx.OK(c, nil)
}

func (h *AIHandler) updateSessionTitle(c *gin.Context) {
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
	session, err := h.sessions.UpdateSessionTitle(uid, c.Param("id"), title)
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

func (h *AIHandler) refreshSuggestions(uid uint64, scene, answer string) []RecommendationRecord {
	scene = logic.NormalizeScene(scene)
	prompt := "你是 suggestion 智能体。基于下面回答提炼 3 条可执行建议，每条一行，格式为：标题|内容|相关度(0-1)|思考摘要（不超过60字）。回答内容如下：\n" + answer
	out := []RecommendationRecord{}
	if h.ai != nil {
		msg, err := h.ai.Generate(context.Background(), []*schema.Message{schema.UserMessage(prompt)})
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
				out = append(out, RecommendationRecord{
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
		out = append(out, RecommendationRecord{
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
	h.runtime.SetRecommendations(uid, scene, out)
	return out
}
