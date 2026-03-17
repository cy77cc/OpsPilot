package logic

import (
	"context"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildAugmentedMessage_IncludesSceneContextPromptsAndConstraints(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:logic-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AIScenePrompt{}, &model.AISceneConfig{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	if err := db.Create(&model.AIScenePrompt{
		Scene:      "cluster",
		PromptText: "优先关注集群健康、节点状态和命名空间资源。",
		IsActive:   true,
	}).Error; err != nil {
		t.Fatalf("seed scene prompt: %v", err)
	}
	if err := db.Create(&model.AISceneConfig{
		Scene:            "cluster",
		Description:      "Kubernetes cluster operations",
		AllowedToolsJSON: `["cluster_inspect","k8s_topology"]`,
		BlockedToolsJSON: `["host_batch_exec_apply"]`,
		ConstraintsJSON:  `{"focus":"readonly diagnosis"}`,
	}).Error; err != nil {
		t.Fatalf("seed scene config: %v", err)
	}

	l := &Logic{svcCtx: &svc.ServiceContext{DB: db}}
	message := l.buildAugmentedMessage(context.Background(), "cluster", map[string]any{
		"route":       "/deployment/infrastructure/clusters/42",
		"resource_id": "42",
	}, "检查这个集群为什么不健康")

	for _, fragment := range []string{
		"scene=cluster",
		`scene_context={"resource_id":"42","route":"/deployment/infrastructure/clusters/42"}`,
		"scene_prompts=[",
		"allowed_tools=[\"cluster_inspect\",\"k8s_topology\"]",
		"blocked_tools=[\"host_batch_exec_apply\"]",
		"User request:\n检查这个集群为什么不健康",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("expected augmented message to contain %q, got: %s", fragment, message)
		}
	}
}
