package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	adkdeep "github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/middleware"
	agenttodo "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/todo"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools"
	aiclient "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/client"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

const defaultScene = "ai"

func NewOpsPilotAgent(ctx context.Context, scene string) (adk.ResumableAgent, error) {
	registry := NewDefaultRegistry()
	return createDeepAgent(ctx, registry, scene)
}

func NewOpsPilotAgentFromContext(ctx context.Context) (adk.ResumableAgent, error) {
	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	return NewOpsPilotAgent(ctx, sceneMeta.Scene)
}

func SceneRequiresReadOnlyExecution(scene string) bool {
	spec, ok := NewDefaultRegistry().Lookup(scene)
	return ok && spec.ReadOnly
}

func createDeepAgent(ctx context.Context, registry *Registry, scene string) (adk.ResumableAgent, error) {
	if registry == nil {
		registry = NewDefaultRegistry()
	}

	normalizedScene := strings.TrimSpace(scene)
	if normalizedScene == "" {
		normalizedScene = defaultScene
	}

	chatModel, err := aiclient.GetDefaultChatModel(ctx, nil, aiclient.ChatModelConfig{})
	if err != nil {
		return nil, fmt.Errorf("create chat model: %w", err)
	}

	sceneTools := tools.BuildToolsForSceneWithMode(ctx, normalizedScene, false)
	if len(sceneTools) == 0 {
		return nil, fmt.Errorf("no tools available for scene: %s", normalizedScene)
	}

	mainHandlers, err := middleware.BuildAgentHandlers(ctx, normalizedScene, sceneTools)
	if err != nil {
		return nil, fmt.Errorf("build deep agent handlers: %w", err)
	}

	todoMiddleware, err := agenttodo.NewWriteOpsTodosMiddleware()
	if err != nil {
		return nil, fmt.Errorf("build write ops todos middleware: %w", err)
	}

	summaryMiddleware, err := summarization.New(ctx, &summarization.Config{
		Model:   chatModel,
		Trigger: &summarization.TriggerCondition{ContextTokens: 24000},
		PreserveUserMessages: &summarization.PreserveUserMessages{
			Enabled:   true,
			MaxTokens: 8000,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build summarization middleware: %w", err)
	}

	subAgents, err := buildDeepSubAgents(ctx, registry)
	if err != nil {
		return nil, err
	}

	return adkdeep.New(ctx, &adkdeep.Config{
		Name:                   "deep_main",
		Description:            "OpsPilot deep orchestrator for governed operations and specialist delegation.",
		ChatModel:              chatModel,
		// Instruction:            buildDeepInstruction(normalizedScene),
		SubAgents:              subAgents,
		ToolsConfig:            adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: sceneTools}},
		// WithoutWriteTodos:      true,
		// WithoutGeneralSubAgent: true,
		Handlers:               append([]adk.ChatModelAgentMiddleware{summaryMiddleware}, append(mainHandlers, todoMiddleware)...),
		MaxIteration:           32,
	})
}

func createNamedSceneAgent(
	ctx context.Context,
	name, scene, description, instruction string,
	readOnly bool,
) (adk.ResumableAgent, error) {
	svcCtx, ok := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
	if !ok || svcCtx == nil {
		return nil, fmt.Errorf("service context not found")
	}

	chatModel, err := aiclient.GetDefaultChatModel(ctx, nil, aiclient.ChatModelConfig{})
	if err != nil {
		return nil, fmt.Errorf("create chat model: %w", err)
	}

	sceneTools := tools.BuildToolsForSceneWithMode(ctx, scene, readOnly)
	if len(sceneTools) == 0 {
		return nil, fmt.Errorf("no tools available for scene: %s", scene)
	}

	handlers, err := middleware.BuildAgentHandlers(ctx, scene, sceneTools)
	if err != nil {
		return nil, fmt.Errorf("build agent handlers: %w", err)
	}
	summaryMiddleware, err := summarization.New(ctx, &summarization.Config{
		Model:   chatModel,
		Trigger: &summarization.TriggerCondition{ContextTokens: 24000},
		PreserveUserMessages: &summarization.PreserveUserMessages{
			Enabled:   true,
			MaxTokens: 8000,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build specialist summarization middleware: %w", err)
	}

	agentName := strings.TrimSpace(name)
	if agentName == "" {
		agentName = strings.TrimSpace(scene)
	}

	agentDescription := strings.TrimSpace(description)
	if agentDescription == "" {
		agentDescription = fmt.Sprintf("%s operations specialist", strings.TrimSpace(scene))
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        agentName,
		Description: agentDescription,
		Instruction: strings.TrimSpace(instruction),
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: sceneTools}},
		Handlers:    append([]adk.ChatModelAgentMiddleware{summaryMiddleware}, handlers...),
	})
	if err != nil {
		return nil, fmt.Errorf("create chat model agent: %w", err)
	}

	return agent, nil
}

func buildDeepSubAgents(ctx context.Context, registry *Registry) ([]adk.Agent, error) {
	if registry == nil {
		registry = NewDefaultRegistry()
	}
	entries := registry.Entries()
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Scene < entries[j].Scene
	})

	subAgents := make([]adk.Agent, 0, len(entries))
	for _, entry := range entries {
		spec := entry.Spec
		agentName := strings.TrimSpace(spec.Name)
		if agentName == "" {
			continue
		}
		domain := strings.TrimSpace(spec.Domain)
		if domain == "" {
			domain = strings.TrimSpace(entry.Scene)
		}
		desc := strings.TrimSpace(spec.Description)
		if desc == "" {
			desc = fmt.Sprintf("%s specialist. Keep results compact and prefer summary-only returns.", agentName)
		}
		instruction := strings.TrimSpace(spec.Instruction)
		if instruction == "" {
			instruction = fmt.Sprintf(`You are the %s specialist.

Scope: %s
Rules:
1. Keep results compact and summary-oriented.
2. Use read-only analysis only.
3. Return concrete findings and next checks.`, agentName, domain)
		}
		specialist, err := createNamedSceneAgent(ctx, agentName, domain, desc, instruction, true)
		if err != nil {
			return nil, fmt.Errorf("create deep sub-agent %s: %w", agentName, err)
		}
		subAgents = append(subAgents, specialist)
	}
	return subAgents, nil
}

func buildDeepInstruction(scene string) string {
	trimmedScene := strings.TrimSpace(scene)
	if trimmedScene == "" {
		trimmedScene = defaultScene
	}
	return fmt.Sprintf(`You are the Deep main agent for OpsPilot.

Current scene hint: %s.

Execution policy:
1. Default to solving directly with the current toolset.
2. Use task sub-agents only when needed for context isolation, parallel research, or specialist tool selection.
3. Sub-agents are read-only and must return compact summaries.
4. Any write, mutation, or governed action must be performed by you through approval-aware tools.
5. Keep user-facing output concise, structured, and actionable.
`, trimmedScene)
}
