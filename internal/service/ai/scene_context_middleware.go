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

// ginRuntimeContextKey 是 gin.Context 中存储 RuntimeContext 的键名。
const ginRuntimeContextKey = "ai.runtime_context"

// SceneContextMiddleware 在每个请求前构建 RuntimeContext 并注入到两处：
//   - gin.Context（用于同一 handler 内快速读取）
//   - http.Request.Context（用于传递给下游服务）
//
// 场景 key 优先级：query param scene > path param scene > "global"
// ProjectName 通过 projectID 懒加载查询数据库。
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

// baseRuntimeContext 从 gin.Context 读取 SceneContextMiddleware 预设的 RuntimeContext。
// 若中间件未运行（如单元测试），则从 request context 中读取。
func (h *HTTPHandler) baseRuntimeContext(c *gin.Context) coreai.RuntimeContext {
	if value, ok := c.Get(ginRuntimeContextKey); ok {
		if runtimeCtx, ok := value.(coreai.RuntimeContext); ok {
			return runtimeCtx
		}
	}
	return airuntime.RuntimeContextFromContext(c.Request.Context())
}

// contextWithRuntime 将请求体中的 context 字段与中间件捕获的 RuntimeContext 合并，
// 返回注入了合并后 RuntimeContext 的 Go context 和合并后的 RuntimeContext。
func (h *HTTPHandler) contextWithRuntime(c *gin.Context, raw map[string]any) (context.Context, coreai.RuntimeContext) {
	runtimeCtx := h.normalizeRuntimeContext(c, raw)
	ctx := airuntime.ContextWithRuntimeContext(c.Request.Context(), runtimeCtx)
	return ctx, runtimeCtx
}

// lookupProjectName 根据 projectID 查询项目名称，查询失败或 ID 无效时返回空字符串。
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

// parseSelectedResourcesJSON 将 URL query 中 JSON 格式的 selected_resources 解析为结构体列表。
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

// hintRequestParams 将 URL query 参数转换为 map，供 HintResolver 使用。
// selected_resources 参数特殊处理：解析为结构化列表而非原始字符串。
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
