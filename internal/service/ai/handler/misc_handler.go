package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *AIHandler) listSessions(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	scene := strings.TrimSpace(c.Query("scene"))
	httpx.OK(c, h.sessions.List(uid, scene))
}

func (h *AIHandler) currentSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	scene := strings.TrimSpace(c.Query("scene"))
	items := h.sessions.List(uid, scene)
	if len(items) == 0 {
		httpx.OK(c, nil)
		return
	}
	httpx.OK(c, items[0])
}

func (h *AIHandler) getSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	sess, ok := h.sessions.Get(uid, c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.NotFound, "session not found")
		return
	}
	httpx.OK(c, sess)
}

func (h *AIHandler) branchSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Title           string `json:"title"`
		AnchorMessageID string `json:"anchor_message_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	sourceID := c.Param("id")
	src, ok := h.sessions.Get(uid, sourceID)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "source session not found")
		return
	}
	now := time.Now()
	branched := &logic.AISession{
		ID:        "sess-" + uuid.NewString(),
		UserID:    uid,
		Scene:     src.Scene,
		Title:     strings.TrimSpace(req.Title),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if branched.Title == "" {
		branched.Title = "Branched: " + src.Title
	}
	h.sessions.Put(branched)
	httpx.OK(c, branched)
}

func (h *AIHandler) deleteSession(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	h.sessions.Delete(uid, c.Param("id"))
	httpx.OK(c, gin.H{"status": "ok"})
}

func (h *AIHandler) updateSessionTitle(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	session, ok := h.sessions.Get(uid, c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.NotFound, "session not found")
		return
	}
	session.Title = strings.TrimSpace(req.Title)
	session.UpdatedAt = time.Now()
	h.sessions.Put(session)
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
