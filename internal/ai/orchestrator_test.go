package ai

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/config"
)

// TestOrchestratorRunWithRealModel 使用真实模型验证编排器可执行性。
func TestOrchestratorRunWithRealModel(t *testing.T) {
	t.Parallel()

	setupRealModelConfigForTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	o, err := NewOrchestrator(ctx, common.PlatformDeps{})
	if err != nil {
		t.Fatalf("NewOrchestrator() error = %v", err)
	}

	reply, err := o.Run(ctx, "你好")
	fmt.Println(reply)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.TrimSpace(reply) == "" {
		t.Fatalf("Run() reply is empty")
	}
}

// TestOrchestratorRunWithCheckpointRealModel 使用真实模型验证带 checkpoint 的运行路径。
func TestOrchestratorRunWithCheckpointRealModel(t *testing.T) {
	t.Parallel()

	setupRealModelConfigForTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	o, err := NewOrchestrator(ctx, common.PlatformDeps{})
	if err != nil {
		t.Fatalf("NewOrchestrator() error = %v", err)
	}

	reply, interrupted, err := o.RunWithCheckPoint(ctx, "请用一句话回答：Go 语言支持协程吗？", "orchestrator-real-model-cp")
	if err != nil {
		t.Fatalf("RunWithCheckPoint() error = %v", err)
	}
	if interrupted {
		t.Fatalf("RunWithCheckPoint() interrupted = true, want false")
	}
	if strings.TrimSpace(reply) == "" {
		t.Fatalf("RunWithCheckPoint() reply is empty")
	}
}

// TestOrchestratorValidation 覆盖基础参数校验分支。
func TestOrchestratorValidation(t *testing.T) {
	t.Parallel()

	o := &Orchestrator{}
	if _, err := o.Run(context.Background(), "hello"); err == nil || !strings.Contains(err.Error(), "orchestrator runner is nil") {
		t.Fatalf("Run() error = %v, want runner nil", err)
	}
}

// setupRealModelConfigForTest 为真实模型测试设置配置并在条件不满足时跳过。
func setupRealModelConfigForTest(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skip real model test in short mode")
	}

	provider := strings.TrimSpace(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		provider = "qwen"
	}

	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://coding.dashscope.aliyuncs.com/v1"
	}

	model := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if model == "" {
		model = "qwen3.5-plus"
	}

	apiKey := strings.TrimSpace("sk-sp-95f6a7ca251f49f2975801ca53426f61")
	if strings.EqualFold(provider, "qwen") && apiKey == "" {
		t.Skip("skip real model test: LLM_API_KEY is empty")
	}

	oldCfg := config.CFG
	t.Cleanup(func() {
		config.CFG = oldCfg
	})

	config.CFG.LLM.Enable = true
	config.CFG.LLM.Provider = provider
	config.CFG.LLM.BaseURL = baseURL
	config.CFG.LLM.Model = model
	config.CFG.LLM.APIKey = apiKey
}
