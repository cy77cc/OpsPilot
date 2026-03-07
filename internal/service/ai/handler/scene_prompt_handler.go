package handler

import (
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ScenePromptItem 场景提示词响应项
type ScenePromptItem struct {
	ID           uint64 `json:"id"`
	PromptText   string `json:"prompt_text"`
	PromptType   string `json:"prompt_type"`
	DisplayOrder int    `json:"display_order"`
}

// scenePrompts 获取场景快捷提示词
func (h *AIHandler) scenePrompts(c *gin.Context) {
	scene := c.Param("scene")
	if scene == "" {
		httpx.Fail(c, xcode.ParamError, "scene is required")
		return
	}

	// 标准化场景名称
	scene = normalizeSceneKey(scene)

	// 从数据库获取提示词
	var prompts []model.AIScenePrompt
	var err error

	if h.svcCtx.DB != nil {
		err = h.svcCtx.DB.Where("scene = ? AND is_active = ?", scene, true).
			Order("display_order ASC").
			Find(&prompts).Error
	}

	if err != nil && err != gorm.ErrRecordNotFound {
		httpx.Fail(c, xcode.ServerError, "failed to get prompts")
		return
	}

	// 转换为响应格式
	items := make([]ScenePromptItem, 0, len(prompts))
	for _, p := range prompts {
		items = append(items, ScenePromptItem{
			ID:           p.ID,
			PromptText:   p.PromptText,
			PromptType:   p.PromptType,
			DisplayOrder: p.DisplayOrder,
		})
	}

	// 如果数据库没有，尝试从配置获取
	if len(items) == 0 {
		items = getDefaultPromptsForScene(scene)
	}

	httpx.OK(c, gin.H{
		"scene":   scene,
		"prompts": items,
	})
}

// getDefaultPromptsForScene 获取场景默认提示词
func getDefaultPromptsForScene(scene string) []ScenePromptItem {
	// 从场景配置获取
	meta, exists := sceneMetaByKey(scene)
	if !exists {
		return []ScenePromptItem{
			{PromptText: "有什么可以帮助你的？", PromptType: "quick_action"},
		}
	}

	// 根据场景工具生成默认提示词
	prompts := make([]ScenePromptItem, 0)
	switch scene {
	case "deployment:clusters":
		prompts = append(prompts,
			ScenePromptItem{PromptText: "查看所有集群状态", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "帮我检查集群健康度", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "部署应用到指定集群", PromptType: "quick_action"},
		)
	case "deployment:hosts":
		prompts = append(prompts,
			ScenePromptItem{PromptText: "查看主机列表", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "执行主机健康检查", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "在主机上执行命令", PromptType: "quick_action"},
		)
	case "services:list", "services:catalog":
		prompts = append(prompts,
			ScenePromptItem{PromptText: "查看服务目录", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "搜索服务", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "创建新服务", PromptType: "quick_action"},
		)
	case "deployment:metrics", "monitor":
		prompts = append(prompts,
			ScenePromptItem{PromptText: "查看监控指标", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "查看告警列表", PromptType: "quick_action"},
			ScenePromptItem{PromptText: "分析服务性能", PromptType: "quick_action"},
		)
	default:
		// 根据工具生成通用提示词
		if len(meta.Tools) > 0 {
			prompts = append(prompts,
				ScenePromptItem{PromptText: "帮我了解当前页面功能", PromptType: "quick_action"},
				ScenePromptItem{PromptText: "查看相关资源列表", PromptType: "quick_action"},
			)
		}
		if meta.Description != "" {
			prompts = append(prompts,
				ScenePromptItem{PromptText: "我需要帮助: " + meta.Description, PromptType: "quick_action"},
			)
		}
	}

	if len(prompts) == 0 {
		prompts = append(prompts,
			ScenePromptItem{PromptText: "有什么可以帮助你的？", PromptType: "quick_action"},
		)
	}

	return prompts
}

// ensureScenePromptsTable 确保表存在
func ensureScenePromptsTable(db *gorm.DB) error {
	return db.AutoMigrate(&model.AIScenePrompt{})
}
