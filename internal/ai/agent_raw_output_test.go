// Package ai_test 提供直接观察 Agent 原生输出的测试工具。
//
// TestAgentRawOutput 是一个调试测试，用于打印 ADK Agent 事件的完整结构，
// 帮助开发者理解 plan-execute 管线的内部流转。
package ai_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/spf13/viper"
)

// TestAgentRawOutput 直接运行 Agent 并打印原始事件输出。
//
// 此测试用于调试和观察 ADK Agent 的内部事件流。
// 运行前需要设置环境变量 LLM_API_KEY 或 QWEN_API_KEY。
//
// 运行方式（在项目根目录）:
//
//	export LLM_API_KEY=your-api-key
//	cd /root/project/k8s-manage && go test -v -run TestAgentRawOutput ./internal/ai/...
func TestAgentRawOutput(t *testing.T) {
	// 尝试加载配置
	configLoaded := tryLoadConfig(t)
	if !configLoaded {
		t.Skip("跳过测试：未找到配置文件。请在项目根目录运行。")
	}

	// 检查 LLM 配置
	if !config.CFG.LLM.Enable {
		t.Skip("跳过测试：LLM 未启用")
	}

	// 检查 API Key 是否有效（不是占位符）
	apiKey := config.CFG.LLM.APIKey
	if strings.HasPrefix(apiKey, "${") || apiKey == "" {
		// 尝试从环境变量获取
		if envKey := os.Getenv("QWEN_API_KEY"); envKey != "" {
			apiKey = envKey
			config.CFG.LLM.APIKey = apiKey
		} else if envKey := os.Getenv("LLM_API_KEY"); envKey != "" {
			apiKey = envKey
			config.CFG.LLM.APIKey = apiKey
		}
	}
	if strings.HasPrefix(config.CFG.LLM.APIKey, "${") || config.CFG.LLM.APIKey == "" {
		t.Skip("跳过测试：LLM API Key 未设置。请设置环境变量 LLM_API_KEY 或 QWEN_API_KEY")
	}

	t.Logf("使用 API Key: %s...%s",
		config.CFG.LLM.APIKey[:min(4, len(config.CFG.LLM.APIKey))],
		config.CFG.LLM.APIKey[max(0, len(config.CFG.LLM.APIKey)-4):])

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 创建 Agent（PlatformDeps 可以为空，仅用于工具执行）
	deps := agents.Deps{
		PlatformDeps:     common.PlatformDeps{},
		ContextProcessor: airuntime.NewContextProcessor(airuntime.NewSceneConfigResolver(nil)),
	}

	agent, err := agents.NewAgent(ctx, deps)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		CheckPointStore: airuntime.NewCheckpointStore(nil, ""),
		EnableStreaming: true,
	})

	// 用户问题
	userQuery := "你好"
	t.Logf("用户问题: %s", userQuery)
	t.Log("=" + strings.Repeat("=", 60))

	// 运行 Agent
	iter := runner.Query(ctx, userQuery)

	// 收集并打印所有原始事件
	eventCount := 0
	for {
		event, ok := iter.Next()
		if !ok {
			t.Log("\n" + strings.Repeat("=", 61))
			t.Log("事件流结束")
			break
		}

		eventCount++
		t.Logf("\n--- Event #%d ---", eventCount)
		printAgentEvent(t, event)
	}

	t.Logf("共收到 %d 个事件", eventCount)
}

// TestAgentRawOutputWithScene 测试带场景上下文的 Agent 原生输出。
//
// 运行方式:
//
//	export LLM_API_KEY=your-api-key
//	cd /root/project/k8s-manage && go test -v -run TestAgentRawOutputWithScene ./internal/ai/...
func TestAgentRawOutputWithScene(t *testing.T) {
	if !tryLoadConfig(t) {
		t.Skip("跳过测试：未找到配置文件")
	}

	if !config.CFG.LLM.Enable {
		t.Skip("跳过测试：LLM 未启用")
	}

	// 检查 API Key 是否有效
	if strings.HasPrefix(config.CFG.LLM.APIKey, "${") || config.CFG.LLM.APIKey == "" {
		// 尝试从环境变量获取
		if envKey := os.Getenv("QWEN_API_KEY"); envKey != "" {
			config.CFG.LLM.APIKey = envKey
		} else if envKey := os.Getenv("LLM_API_KEY"); envKey != "" {
			config.CFG.LLM.APIKey = envKey
		}
	}
	if strings.HasPrefix(config.CFG.LLM.APIKey, "${") || config.CFG.LLM.APIKey == "" {
		t.Skip("跳过测试：LLM API Key 未设置。请设置环境变量 LLM_API_KEY 或 QWEN_API_KEY")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sceneResolver := airuntime.NewSceneConfigResolver(nil)
	contextProcessor := airuntime.NewContextProcessor(sceneResolver)

	deps := agents.Deps{
		PlatformDeps:     common.PlatformDeps{},
		ContextProcessor: contextProcessor,
	}

	agent, err := agents.NewAgent(ctx, deps)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}

	checkpointStore := airuntime.NewCheckpointStore(nil, "")
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		CheckPointStore: checkpointStore,
		EnableStreaming: true,
	})

	// 带场景上下文的运行时配置
	runtimeCtx := airuntime.RuntimeContext{
		Scene: "deployment:hosts",
		UserContext: map[string]any{
			"environment": "development",
		},
	}

	sessionValues := map[string]any{
		airuntime.SessionKeyRuntimeContext: runtimeCtx,
		airuntime.SessionKeyResolvedScene:  sceneResolver.Resolve(runtimeCtx.Scene),
		airuntime.SessionKeySessionID:      "test-session-001",
		airuntime.SessionKeyPlanID:         "test-plan-001",
		airuntime.SessionKeyTurnID:         "test-turn-001",
	}

	userQuery := "查询主机列表"
	t.Logf("用户问题: %s (场景: %s)", userQuery, runtimeCtx.Scene)
	t.Log("=" + strings.Repeat("=", 60))

	iter := runner.Query(ctx, userQuery,
		adk.WithSessionValues(sessionValues),
	)

	eventCount := 0
	for {
		event, ok := iter.Next()
		if !ok {
			t.Log("\n" + strings.Repeat("=", 61))
			t.Log("事件流结束")
			break
		}

		eventCount++
		t.Logf("\n--- Event #%d ---", eventCount)
		printAgentEvent(t, event)
	}

	t.Logf("共收到 %d 个事件", eventCount)
}

// tryLoadConfig 尝试从多个路径加载配置。
func tryLoadConfig(t *testing.T) bool {
	t.Helper()
	configPaths := []string{
		"configs/config.yaml",
		"../configs/config.yaml",
		"../../configs/config.yaml",
		"../../../configs/config.yaml",
	}
	for _, path := range configPaths {
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			continue
		}
		if err := viper.Unmarshal(&config.CFG); err != nil {
			continue
		}
		// 配置加载后，用环境变量补充缺失的 API Key
		applyEnvVars()
		return true
	}
	return false
}

// applyEnvVars 用环境变量补充配置。
func applyEnvVars() {
	// LLM API Key 优先使用环境变量
	// 支持 QWEN_API_KEY 和 LLM_API_KEY 两种环境变量名
	apiKey := os.Getenv("QWEN_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if apiKey != "" {
		config.CFG.LLM.APIKey = apiKey
	}

	// 如果配置中的 API Key 是占位符（以 ${ 开头），也用环境变量替换
	if strings.HasPrefix(config.CFG.LLM.APIKey, "${") {
		config.CFG.LLM.APIKey = apiKey
	}

	// 如果有 API Key 但 LLM 未启用，自动启用
	if config.CFG.LLM.APIKey != "" && !config.CFG.LLM.Enable {
		config.CFG.LLM.Enable = true
	}
}

// printAgentEvent 格式化打印 Agent 事件的详细结构。
func printAgentEvent(t *testing.T, event *adk.AgentEvent) {
	if event == nil {
		t.Log("  [nil event]")
		return
	}

	// 打印基本信息
	t.Logf("  AgentName: %q", event.AgentName)

	// 打印运行路径
	if len(event.RunPath) > 0 {
		t.Logf("  RunPath: %d steps", len(event.RunPath))
	}

	// 打印错误
	if event.Err != nil {
		t.Errorf("  Error: %v", event.Err)
		return
	}

	// 打印输出
	if event.Output != nil {
		t.Log("  Output:")
		printAgentOutput(t, event.Output, "    ")
	}

	// 打印动作（中断、退出等）
	if event.Action != nil {
		t.Log("  Action:")
		printAgentAction(t, event.Action, "    ")
	}
}

// printAgentOutput 打印 AgentOutput 的详细结构。
func printAgentOutput(t *testing.T, output *adk.AgentOutput, indent string) {
	if output == nil {
		return
	}

	// 消息输出
	if output.MessageOutput != nil {
		msg := output.MessageOutput
		t.Logf("%sMessageOutput:", indent)
		t.Logf("%s  Role: %s", indent, msg.Role)
		t.Logf("%s  IsStreaming: %v", indent, msg.IsStreaming)
		t.Logf("%s  ToolName: %q", indent, msg.ToolName)

		// 非流式消息
		if msg.Message != nil {
			content := msg.Message.Content
			t.Logf("%s  Message.Content: %q", indent, truncateString(content, 200))
			if len(msg.Message.ToolCalls) > 0 {
				t.Logf("%s  Message.ToolCalls: %d 个", indent, len(msg.Message.ToolCalls))
				for i, tc := range msg.Message.ToolCalls {
					t.Logf("%s    [%d] ID: %s, Name: %s", indent, i, tc.ID, tc.Function.Name)
					if tc.Function.Arguments != "" {
						t.Logf("%s         Args: %s", indent, truncateString(tc.Function.Arguments, 100))
					}
				}
			}
		}

		// 流式消息
		if msg.MessageStream != nil && msg.IsStreaming {
			chunks := collectStreamChunks(t, msg.MessageStream)
			if len(chunks) > 0 {
				fullContent := strings.Join(chunks, "")
				t.Logf("%s  StreamContent: %q", indent, truncateString(fullContent, 300))
			}
		}
	}

	// 自定义输出
	if output.CustomizedOutput != nil {
		customJSON, err := json.MarshalIndent(output.CustomizedOutput, indent+"  ", "  ")
		if err == nil {
			t.Logf("%sCustomizedOutput: %s", indent, string(customJSON))
		}
	}
}

// printAgentAction 打印 AgentAction 的详细结构。
func printAgentAction(t *testing.T, action *adk.AgentAction, indent string) {
	if action == nil {
		return
	}

	// 退出标记
	if action.Exit {
		t.Logf("%sExit: true", indent)
	}

	// 中断信息
	if action.Interrupted != nil {
		t.Logf("%sInterrupted:", indent)
		t.Logf("%s  Data: %v", indent, action.Interrupted.Data)
		if len(action.Interrupted.InterruptContexts) > 0 {
			t.Logf("%s  InterruptContexts: %d 个", indent, len(action.Interrupted.InterruptContexts))
			for i, ctx := range action.Interrupted.InterruptContexts {
				t.Logf("%s    [%d] ID: %s", indent, i, ctx.ID)
				if ctx.Info != nil {
					infoJSON, err := json.MarshalIndent(ctx.Info, indent+"      ", "  ")
					if err == nil {
						t.Logf("%s      Info: %s", indent, string(infoJSON))
					}
				}
			}
		}
	}

	// 转移到其他 Agent
	if action.TransferToAgent != nil {
		t.Logf("%sTransferToAgent: %s", indent, action.TransferToAgent.DestAgentName)
	}

	// 中断循环
	if action.BreakLoop != nil {
		t.Logf("%sBreakLoop: from=%s", indent, action.BreakLoop.From)
	}

	// 自定义动作
	if action.CustomizedAction != nil {
		customJSON, _ := json.MarshalIndent(action.CustomizedAction, indent+"  ", "  ")
		t.Logf("%sCustomizedAction: %s", indent, string(customJSON))
	}
}

// collectStreamChunks 收集流式消息的所有分片。
func collectStreamChunks(t *testing.T, stream adk.MessageStream) []string {
	if stream == nil {
		return nil
	}

	var chunks []string
	for {
		frame, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Logf("    Stream error: %v", err)
			break
		}
		if frame != nil && frame.Content != "" {
			chunks = append(chunks, frame.Content)
		}
	}
	return chunks
}

// truncateString 截断字符串，用于日志输出。
func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestPrintAgentEventStructure 展示 AgentEvent 的完整结构（文档用途）。
func TestPrintAgentEventStructure(t *testing.T) {
	// AgentEvent 是 ADK 返回的核心事件结构，包含以下主要字段：
	//
	// type AgentEvent struct {
	//     AgentName string        // 产生事件的 Agent 名称（planner/executor/replanner）
	//     RunPath   []RunStep     // 执行路径（嵌套 agent 时有效）
	//     Output    *AgentOutput  // 输出内容（消息、自定义输出等）
	//     Action    *AgentAction  // 动作（中断、退出、转移等）
	//     Err       error         // 错误信息
	// }
	//
	// type AgentOutput struct {
	//     MessageOutput   *MessageVariant  // LLM 消息输出
	//     CustomizedOutput any             // 自定义输出（如计划）
	// }
	//
	// type MessageVariant struct {
	//     Message       Message        // 完整消息（非流式）
	//     MessageStream MessageStream  // 流式消息
	//     Role          Role           // 角色 (Assistant/Tool)
	//     IsStreaming   bool           // 是否流式
	//     ToolName      string         // 工具名称（tool 消息时）
	// }
	//
	// type AgentAction struct {
	//     Exit            bool                // 是否退出
	//     Interrupted     *InterruptInfo      // 中断信息（审批等待等）
	//     TransferToAgent *TransferToAgentAction // 转移到其他 agent
	//     BreakLoop       *BreakLoopAction    // 中断循环
	//     CustomizedAction any                // 自定义动作
	// }
	//
	// 典型事件流：
	// 1. planner 输出计划 JSON (MessageOutput, Role=assistant)
	// 2. executor 输出工具调用 (MessageOutput.ToolCalls)
	// 3. 工具执行结果 (MessageOutput, Role=tool)
	// 4. replanner 调整计划 (MessageOutput)
	// 5. 最终回答 (MessageOutput, 流式文本)
	// 6. 完成 (Action.Exit=true)
	t.Log("参见源码注释")
}
