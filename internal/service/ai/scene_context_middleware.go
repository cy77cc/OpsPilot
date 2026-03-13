package ai

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
)

const ginRuntimeContextKey = "ai.runtime_context"

func (h *HTTPHandler) SceneContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		runtimeCtx := coreai.RuntimeContext{
			Scene:       firstNonEmpty(strings.TrimSpace(c.Query("scene")), strings.TrimSpace(c.Param("scene")), "global"),
			Route:       c.FullPath(),
			ProjectID:   firstNonEmpty(strings.TrimSpace(c.GetHeader("X-Project-ID")), strings.TrimSpace(c.Query("project_id"))),
			CurrentPage: firstNonEmpty(strings.TrimSpace(c.GetHeader("X-Current-Page")), strings.TrimSpace(c.Query("current_page")), c.Request.Referer()),
			UserContext: map[string]any{
				"uid":   httpx.UIDFromCtx(c),
				"admin": httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)),
			},
			Metadata: map[string]any{},
		}

		if runtimeCtx.ProjectID != "" {
			runtimeCtx.ProjectName = h.lookupProjectName(c.Request.Context(), runtimeCtx.ProjectID)
		}
		if resources := parseSelectedResourcesJSON(firstNonEmpty(c.Query("selected_resources"), c.Query("selectedResources"))); len(resources) > 0 {
			runtimeCtx.SelectedResources = resources
		}

		c.Set(ginRuntimeContextKey, runtimeCtx)
		c.Request = c.Request.WithContext(airuntime.ContextWithRuntimeContext(c.Request.Context(), runtimeCtx))
		c.Next()
	}
}

func (h *HTTPHandler) baseRuntimeContext(c *gin.Context) coreai.RuntimeContext {
	if value, ok := c.Get(ginRuntimeContextKey); ok {
		if runtimeCtx, ok := value.(coreai.RuntimeContext); ok {
			return runtimeCtx
		}
	}
	return airuntime.RuntimeContextFromContext(c.Request.Context())
}

func (h *HTTPHandler) contextWithRuntime(c *gin.Context, raw map[string]any) (context.Context, coreai.RuntimeContext) {
	runtimeCtx := h.normalizeRuntimeContext(c, raw)
	ctx := airuntime.ContextWithRuntimeContext(c.Request.Context(), runtimeCtx)
	return ctx, runtimeCtx
}

func (h *HTTPHandler) lookupProjectName(ctx context.Context, projectID string) string {
	id, err := strconv.Atoi(strings.TrimSpace(projectID))
	if err != nil || id <= 0 || h == nil || h.svcCtx == nil || h.svcCtx.DB == nil {
		return ""
	}
	var project model.Project
	if err := h.svcCtx.DB.WithContext(ctx).Select("id", "name").First(&project, id).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(project.Name)
}

func parseSelectedResourcesJSON(raw string) []coreai.SelectedResource {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []any
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return toSelectedResources(items)
}

func hintRequestParams(c *gin.Context) map[string]any {
	out := make(map[string]any, len(c.Request.URL.Query())+1)
	for key, values := range c.Request.URL.Query() {
		if len(values) == 0 {
			continue
		}
		switch key {
		case "selected_resources", "selectedResources":
			if resources := parseSelectedResourcesJSON(values[0]); len(resources) > 0 {
				rows := make([]any, 0, len(resources))
				for _, resource := range resources {
					rows = append(rows, map[string]any{
						"type":      resource.Type,
						"id":        resource.ID,
						"name":      resource.Name,
						"namespace": resource.Namespace,
					})
				}
				out[key] = rows
			}
		default:
			out[key] = values[0]
		}
	}
	return out
}
