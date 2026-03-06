package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/config"
	"github.com/cy77cc/k8s-manage/internal/rag"
	"github.com/redis/go-redis/v9"
)

type PlatformRunner struct {
	runner       *adk.Runner
	checkpoints  adk.CheckPointStore
	agent        adk.Agent
	Model        model.ToolCallingChatModel
	tools        map[string]tool.InvokableTool
	metas        map[string]tools.ToolMeta
	mcp          *tools.MCPClientManager
	ragRetriever ragPromptRetriever
}

type RunnerConfig struct {
	EnableStreaming bool
	RedisClient     redis.UniversalClient
	CheckPointStore adk.CheckPointStore
}

type ragPromptRetriever interface {
	Retrieve(ctx context.Context, query string, topK int) (*rag.RAGContext, error)
	BuildAugmentedPrompt(query string, context *rag.RAGContext) string
}

func NewPlatformRunner(ctx context.Context, chatModel model.ToolCallingChatModel, deps tools.PlatformDeps, cfg *RunnerConfig) (*PlatformRunner, error) {
	if chatModel == nil {
		return nil, nil
	}

	mcpManager, err := tools.NewMCPClientManager(ctx, tools.MCPConfigFromEnv())
	if err != nil {
		return nil, err
	}
	registered, err := tools.BuildRegisteredToolsWithMCP(deps, mcpManager)
	if err != nil {
		_ = mcpManager.Close()
		return nil, err
	}

	baseTools := make([]tool.BaseTool, 0, len(registered))
	toolMap := make(map[string]tool.InvokableTool, len(registered))
	metaMap := make(map[string]tools.ToolMeta, len(registered))
	for _, item := range registered {
		baseTools = append(baseTools, tools.WrapRegisteredTool(item))
		toolMap[item.Meta.Name] = item.Tool
		metaMap[item.Meta.Name] = item.Meta
	}

	agent, err := newPlatformAgent(ctx, chatModel, baseTools)
	if err != nil {
		_ = mcpManager.Close()
		return nil, err
	}

	store := resolveCheckPointStore(cfg)
	enableStreaming := true
	if cfg != nil {
		enableStreaming = cfg.EnableStreaming
	}

	var ragRetriever ragPromptRetriever
	if deps.DB != nil && config.CFG.Milvus.Enable {
		milvusClient := rag.NewMilvusClient(config.CFG.Milvus)
		if err := milvusClient.EnsureCollections(ctx); err != nil {
			_ = mcpManager.Close()
			return nil, fmt.Errorf("ensure milvus collections: %w", err)
		}
		ragRetriever = rag.NewRAGRetriever(milvusClient, rag.NewEmbedder(config.CFG.Embedder))
	}

	return &PlatformRunner{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{
			EnableStreaming: enableStreaming,
			Agent:           agent,
			CheckPointStore: store,
		}),
		checkpoints:  store,
		agent:        agent,
		Model:        chatModel,
		tools:        toolMap,
		metas:        metaMap,
		mcp:          mcpManager,
		ragRetriever: ragRetriever,
	}, nil
}

func resolveCheckPointStore(cfg *RunnerConfig) adk.CheckPointStore {
	if cfg != nil {
		if cfg.CheckPointStore != nil {
			return cfg.CheckPointStore
		}
		if cfg.RedisClient != nil {
			return NewRedisCheckPointStore(cfg.RedisClient)
		}
	}
	return NewInMemoryCheckPointStore()
}

func (p *PlatformRunner) ToolMetas() []tools.ToolMeta {
	if p == nil {
		return nil
	}
	out := make([]tools.ToolMeta, 0, len(p.metas))
	for _, m := range p.metas {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (p *PlatformRunner) Query(ctx context.Context, sessionID, message string, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	if p == nil || p.runner == nil {
		return errorIterator(fmt.Errorf("runner not initialized"))
	}
	query := p.augmentPrompt(ctx, message)
	callOpts := make([]adk.AgentRunOption, 0, len(opts)+1)
	callOpts = append(callOpts, opts...)
	if sid := strings.TrimSpace(sessionID); sid != "" {
		callOpts = append(callOpts, adk.WithCheckPointID(sid))
	}
	return p.runner.Query(ctx, query, callOpts...)
}

func (p *PlatformRunner) Resume(ctx context.Context, checkpointID string, targets map[string]any, opts ...adk.AgentRunOption) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if p == nil || p.runner == nil {
		return nil, fmt.Errorf("runner not initialized")
	}
	if len(targets) == 0 {
		return p.runner.Resume(ctx, strings.TrimSpace(checkpointID), opts...)
	}
	return p.runner.ResumeWithParams(ctx, strings.TrimSpace(checkpointID), &adk.ResumeParams{
		Targets: targets,
	}, opts...)
}

func (p *PlatformRunner) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	if p == nil {
		return nil, fmt.Errorf("runner not initialized")
	}
	if p.Model == nil {
		return nil, fmt.Errorf("chat model not initialized")
	}
	messages = p.injectRAGIntoMessages(ctx, messages)
	return p.Model.Generate(ctx, messages)
}

func (p *PlatformRunner) RunTool(ctx context.Context, toolName string, params map[string]any) (tools.ToolResult, error) {
	if p == nil {
		return tools.ToolResult{OK: false, Error: "runner not initialized", Source: "platform"}, fmt.Errorf("runner not initialized")
	}
	normalizedName := tools.NormalizeToolName(toolName)
	t, ok := p.tools[normalizedName]
	if !ok {
		return tools.ToolResult{OK: false, Error: "tool not found", Source: "platform"}, fmt.Errorf("tool not found")
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return tools.ToolResult{OK: false, Error: err.Error(), Source: "platform"}, err
	}
	out, err := t.InvokableRun(ctx, string(raw))
	if err != nil {
		return tools.ToolResult{OK: false, Error: err.Error(), Source: "platform"}, nil
	}
	var result tools.ToolResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return tools.ToolResult{OK: true, Data: out, Source: "platform"}, nil
	}
	return result, nil
}

func (p *PlatformRunner) Close() error {
	if p == nil || p.mcp == nil {
		return nil
	}
	return p.mcp.Close()
}

func (p *PlatformRunner) injectRAGIntoMessages(ctx context.Context, messages []*schema.Message) []*schema.Message {
	if p == nil || p.ragRetriever == nil || len(messages) == 0 {
		return messages
	}
	userIdx := -1
	userContent := ""
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg != nil && msg.Role == schema.User {
			userIdx = i
			userContent = strings.TrimSpace(msg.Content)
			break
		}
	}
	if userIdx < 0 || userContent == "" {
		return messages
	}
	contextData, err := p.ragRetriever.Retrieve(ctx, userContent, 6)
	if err != nil {
		return messages
	}
	augmented := p.ragRetriever.BuildAugmentedPrompt(userContent, contextData)
	if strings.TrimSpace(augmented) == "" || augmented == userContent {
		return messages
	}
	out := append([]*schema.Message(nil), messages...)
	copied := *out[userIdx]
	copied.Content = augmented
	out[userIdx] = &copied
	return out
}

func (p *PlatformRunner) augmentPrompt(ctx context.Context, message string) string {
	if p == nil || p.ragRetriever == nil {
		return message
	}
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return message
	}
	contextData, err := p.ragRetriever.Retrieve(ctx, trimmed, 6)
	if err != nil {
		return message
	}
	augmented := p.ragRetriever.BuildAugmentedPrompt(trimmed, contextData)
	if strings.TrimSpace(augmented) == "" {
		return message
	}
	return augmented
}

func errorIterator(err error) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		gen.Send(&adk.AgentEvent{Err: err})
	}()
	return iter
}
