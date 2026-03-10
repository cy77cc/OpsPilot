// Package ai 提供了 K8s 管理平台的核心 AI Agent 实现。
// 包含主要的 AIAgent 结构体，负责编排工具执行、流式响应以及与 Eino 框架的 react.Agent 集成。
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	airag "github.com/cy77cc/k8s-manage/internal/ai/rag"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/core"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/cicd"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/deployment"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/governance"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/host"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/infrastructure"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/kubernetes"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/monitor"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/service"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RunnerConfig 存储 AIAgent 的配置选项。
// 控制流式行为、Redis 连接、执行限制、安全切面、检查点存储和心跳间隔。
type RunnerConfig struct {
	// EnableStreaming 是否启用流式响应。
	EnableStreaming bool
	// RedisClient 提供 Redis 连接，用于会话状态和检查点存储。
	RedisClient redis.UniversalClient
	// MaxStep 限制 Agent 推理的最大步数。
	MaxStep int
	// CheckPointStore 启用跨执行中断的状态持久化。
	CheckPointStore compose.CheckPointStore
	// HeartbeatInterval 定义流式处理期间心跳事件的间隔。
	HeartbeatInterval time.Duration
}

// AIAgent 是基于 react.Agent 实现的 AI 助手。
// 它负责管理工具列表、执行流式对话、处理审批中断，并集成 RAG 检索增强。
type AIAgent struct {
	// model 是支持工具调用的聊天模型。
	model einomodel.ToolCallingChatModel
	// tools 存储所有已注册的工具。
	tools []tool.BaseTool
	// agent 是底层的 ReAct Agent 实现。
	agent *react.Agent
	// retriever 用于 RAG 知识检索。
	retriever airag.Retriever
	// knowledge 是 Milvus 索引器，用于知识库管理。
	knowledge *airag.MilvusIndexer
	// feedback 用于收集会话反馈。
	feedback *airag.SessionFeedbackCollector
	// maxStep 是最大推理步数。
	maxStep int
	// checkpointStore 用于存储执行检查点。
	checkpointStore compose.CheckPointStore
	// heartbeatInterval 是心跳事件间隔。
	heartbeatInterval time.Duration
}

// NewAIAgent 创建一个新的 AI Agent 实例。
//
// 参数:
//   - ctx: 上下文，用于初始化模型和工具。
//   - model: 支持工具调用的聊天模型。
//   - deps: 平台依赖项（数据库、K8s 客户端等）。
//   - cfg: Agent 配置选项，可为 nil。
//
// 返回:
//   - *AIAgent: 创建的 Agent 实例。
//   - error: 初始化过程中的错误。
func NewAIAgent(ctx context.Context, model einomodel.ToolCallingChatModel, deps core.PlatformDeps, cfg *RunnerConfig) (*AIAgent, error) {
	// 直接收集所有工具，简化 Registry 和 Builder 层
	var invokables []tool.InvokableTool
	invokables = append(invokables, host.NewHostTools(ctx, deps)...)
	invokables = append(invokables, deployment.NewDeploymentTools(ctx, deps)...)
	invokables = append(invokables, infrastructure.NewInfrastructureTools(ctx, deps)...)
	invokables = append(invokables, service.NewServiceTools(ctx, deps)...)
	invokables = append(invokables, monitor.NewMonitorTools(ctx, deps)...)
	invokables = append(invokables, governance.NewGovernanceTools(ctx, deps)...)
	invokables = append(invokables, kubernetes.NewKubernetesTools(ctx, deps)...)
	invokables = append(invokables, cicd.NewCICDTools(ctx, deps)...)

	allTools := make([]tool.BaseTool, 0, len(invokables))
	for _, t := range invokables {
		allTools = append(allTools, t)
	}

	maxStep := 15
	if cfg != nil && cfg.MaxStep > 0 {
		maxStep = cfg.MaxStep
	}

	// 构建工具节点配置
	toolsConfig := compose.ToolsNodeConfig{
		Tools: allTools,
	}

	// 创建 react.Agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig:      toolsConfig,
		MaxStep:          maxStep,
	})
	if err != nil {
		return nil, fmt.Errorf("create react agent: %w", err)
	}

	a := &AIAgent{
		model:             model,
		tools:             allTools,
		agent:             agent,
		maxStep:           maxStep,
		heartbeatInterval: 10 * time.Second, // default
	}

	// 应用配置
	if cfg != nil {
		if cfg.HeartbeatInterval > 0 {
			a.heartbeatInterval = cfg.HeartbeatInterval
		}
		a.checkpointStore = cfg.CheckPointStore
	}

	// 初始化 RAG 组件
	knowledgeIndexer := airag.NewMilvusIndexer(nil)
	a.retriever = airag.NewNamespaceRetriever(knowledgeIndexer)
	a.knowledge = knowledgeIndexer

	if cfg != nil && cfg.RedisClient != nil {
		// sessionState 可用于存储会话历史
		// feedback collector 目前不需要 SessionQAExtractor
		a.feedback = airag.NewFeedbackCollector(knowledgeIndexer, nil)
	}

	return a, nil
}

// Generate 非流式生成方法，实现兼容接口。
//
// 参数:
//   - ctx: 上下文。
//   - messages: 消息列表。
//
// 返回:
//   - *schema.Message: 生成的消息。
//   - error: 生成错误。
func (a *AIAgent) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	if a.agent == nil {
		return nil, fmt.Errorf("agent not initialized")
	}
	return a.agent.Generate(ctx, messages)
}

// RunTool 直接执行指定的工具。
// 不经过 Agent 推理，直接调用工具实现。
//
// 参数:
//   - ctx: 上下文。
//   - toolName: 工具名称。
//   - params: 工具参数。
//
// 返回:
//   - core.ToolResult: 工具执行结果。
//   - error: 执行错误。
func (a *AIAgent) RunTool(ctx context.Context, toolName string, params map[string]any) (core.ToolResult, error) {
	var targetTool tool.InvokableTool
	for _, t := range a.tools {
		if it, ok := t.(tool.InvokableTool); ok {
			info, _ := it.Info(ctx)
			if info.Name == toolName {
				targetTool = it
				break
			}
		}
	}

	if targetTool == nil {
		return core.ToolResult{}, fmt.Errorf("tool not found: %s", toolName)
	}

	raw, err := json.Marshal(params)
	if err != nil {
		return core.ToolResult{}, err
	}

	content, err := targetTool.InvokableRun(ctx, string(raw))
	if err != nil {
		return core.ToolResult{OK: false, Error: err.Error(), Source: "tool"}, err
	}

	// 尝试解析为标准 ToolResult 格式
	var result core.ToolResult
	if json.Unmarshal([]byte(content), &result) == nil && (result.OK || result.Error != "" || result.Source != "") {
		return result, nil
	}

	// 新格式：将 Output 结构体包装为 ToolResult
	var outputData map[string]any
	if json.Unmarshal([]byte(content), &outputData) == nil {
		return core.ToolResult{
			OK:     true,
			Data:   outputData,
			Source: "agent",
		}, nil
	}

	// 兜底：原始字符串
	return core.ToolResult{
		OK:     true,
		Data:   map[string]any{"content": content},
		Source: "agent",
	}, nil
}

// Stream 流式执行对话。
// 这是 AIAgent 的核心方法，处理消息、执行工具调用、发送 SSE 事件。
//
// 参数:
//   - ctx: 上下文。
//   - sessionID: 会话 ID，为空时自动生成。
//   - message: 用户消息。
//   - userID: 用户 ID，用于权限检查。
//   - runtimeCtx: 运行时上下文，包含场景、命名空间等信息。
//   - emit: SSE 事件发射函数，返回 false 表示客户端已断开。
//
// 返回:
//   - *StreamResult: 流式执行结果。
//   - error: 执行错误。
//
// SSE 事件类型:
//   - delta: 内容片段。
//   - thinking_delta: 思考链内容。
//   - tool_call: 工具调用。
//   - tool_result: 工具执行结果。
//   - approval_required: 需要审批。
//   - heartbeat: 心跳。
func (a *AIAgent) Stream(ctx context.Context, sessionID, message string, userID uint64, runtimeCtx map[string]any, emit func(event string, payload map[string]any) bool) (*StreamResult, error) {
	if a.agent == nil {
		return nil, fmt.Errorf("agent not initialized")
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "sess-" + uuid.NewString()
	}

	// 构建消息
	messages := []*schema.Message{schema.UserMessage(message)}

	// RAG 增强
	if a.retriever != nil {
		namespace := "global"
		if runtimeCtx != nil {
			if ns, ok := runtimeCtx["namespace"].(string); ok && ns != "" {
				namespace = ns
			}
		}
		if entries, err := a.retriever.Retrieve(ctx, namespace, message, 4); err == nil && len(entries) > 0 {
			augmented := airag.BuildAugmentedPrompt(message, entries)
			messages = []*schema.Message{schema.UserMessage(augmented)}
		}
	}

	// 注入工具上下文
	toolCtx := ctx
	if userID != 0 {
		toolCtx = tools.WithToolUser(toolCtx, userID, "")
	}
	if runtimeCtx != nil {
		toolCtx = tools.WithToolRuntimeContext(toolCtx, runtimeCtx)
	}

	// 注入回调处理器用于 tool_result 事件
	if emit != nil {
		callbackHandler := NewStreamingCallbacks(emit)
		toolCtx = callbacks.InitCallbacks(toolCtx, nil, callbackHandler)
	}

	// 流式调用
	stream, err := a.agent.Stream(toolCtx, messages)
	if err != nil {
		return nil, fmt.Errorf("stream agent: %w", err)
	}
	defer stream.Close()

	result := &StreamResult{SessionID: sessionID}

	// 设置心跳定时器
	heartbeatInterval := a.heartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = 10 * time.Second
	}
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// 创建通道用于异步读取流
	type streamResult struct {
		chunk *schema.Message
		err   error
	}
	streamChan := make(chan streamResult, 1)

	// 启动 goroutine 读取流
	go func() {
		defer close(streamChan)
		for {
			chunk, err := stream.Recv()
			streamChan <- streamResult{chunk: chunk, err: err}
			if err != nil {
				return
			}
		}
	}()

	// 主循环：处理流和心跳
	for {
		select {
		case <-toolCtx.Done():
			return nil, toolCtx.Err()

		case t := <-heartbeatTicker.C:
			if emit != nil {
				emit("heartbeat", map[string]any{"ts": t.UTC().Format(time.RFC3339Nano)})
			}

		case res, ok := <-streamChan:
			if !ok {
				// 流已关闭
				return result, nil
			}

			if res.err != nil {
				if res.err == io.EOF {
					return result, nil
				}

				// 检查是否为中断错误（审批需求）
				if info, ok := compose.IsInterruptRerunError(res.err); ok {
					// 提取审批信息
					if approvalInfo, ok := info.(*tools.ApprovalInfo); ok && emit != nil {
						emit("approval_required", map[string]any{
							"tool_name": approvalInfo.ToolName,
							"arguments": jsonStringToMap(approvalInfo.ArgumentsInJSON),
							"risk":      approvalInfo.Risk,
							"preview":   approvalInfo.Preview,
						})
						// 设置 CheckpointID 用于后续恢复
						result.CheckpointID = "checkpoint-" + uuid.NewString()
					}
					return result, nil
				}

				return nil, fmt.Errorf("receive stream: %w", res.err)
			}

			chunk := res.chunk
			if chunk == nil {
				continue
			}

			// 处理内容
			if chunk.Content != "" {
				result.Content += chunk.Content
				if emit != nil {
					emit("delta", map[string]any{"contentChunk": chunk.Content})
				}
			}

			// 处理思考链
			if chunk.ReasoningContent != "" {
				result.Thinking += chunk.ReasoningContent
				if emit != nil {
					emit("thinking_delta", map[string]any{"contentChunk": chunk.ReasoningContent})
				}
			}

			// 处理工具调用
			for _, tc := range chunk.ToolCalls {
				callID := tc.ID
				if callID == "" {
					callID = "call-" + uuid.NewString()
				}

				if emit != nil {
					emit("tool_call", map[string]any{
						"call_id": callID,
						"tool":    tc.Function.Name,
						"params":  jsonStringToMap(tc.Function.Arguments),
					})
				}

				result.ToolCalls = append(result.ToolCalls, ToolCallInfo{
					ID:        callID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
		}
	}
}

// Resume 从中断点恢复执行。
// 当工具执行需要审批时会中断，用户审批后调用此方法恢复。
//
// 参数:
//   - ctx: 上下文。
//   - checkpointID: 检查点 ID，标识中断位置。
//   - approval: 审批结果，可以是 *tools.ApprovalResult 或 map[string]any。
//   - emit: SSE 事件发射函数。
//
// 返回:
//   - *StreamResult: 恢复执行后的结果。
//   - error: 恢复错误。
func (a *AIAgent) Resume(ctx context.Context, checkpointID string, approval any, emit func(event string, payload map[string]any) bool) (*StreamResult, error) {
	if a.agent == nil {
		return nil, fmt.Errorf("agent not initialized")
	}

	// 设置恢复上下文
	resumeCtx := ctx
	switch v := approval.(type) {
	case *tools.ApprovalResult:
		resumeCtx = compose.ResumeWithData(ctx, checkpointID, v)
	case map[string]any:
		result := &tools.ApprovalResult{}
		if approved, ok := v["approved"].(bool); ok {
			result.Approved = approved
		}
		if reason, ok := v["reason"].(string); ok {
			result.DisapproveReason = &reason
		}
		resumeCtx = compose.ResumeWithData(ctx, checkpointID, result)
	}

	// 继续流式执行
	stream, err := a.agent.Stream(resumeCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("stream agent: %w", err)
	}
	defer stream.Close()

	result := &StreamResult{CheckpointID: checkpointID}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("receive stream: %w", err)
		}
		if chunk == nil {
			continue
		}

		if chunk.Content != "" {
			result.Content += chunk.Content
			if emit != nil {
				emit("delta", map[string]any{"contentChunk": chunk.Content})
			}
		}

		for _, tc := range chunk.ToolCalls {
			callID := tc.ID
			if callID == "" {
				callID = "call-" + uuid.NewString()
			}
			if emit != nil {
				emit("tool_call", map[string]any{
					"call_id": callID,
					"tool":    tc.Function.Name,
					"params":  jsonStringToMap(tc.Function.Arguments),
				})
			}
		}
	}

	return result, nil
}

func (a *AIAgent) ResumePayload(ctx context.Context, checkpointID string, targets map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("ResumePayload not implemented in simplified refactoring")
}

// AddKnowledge 向知识库添加知识条目。
//
// 参数:
//   - ctx: 上下文。
//   - namespace: 命名空间，用于隔离不同租户的知识。
//   - question: 问题内容。
//   - answer: 答案内容。
//
// 返回:
//   - *airag.KnowledgeEntry: 创建的知识条目。
//   - error: 添加错误。
func (a *AIAgent) AddKnowledge(ctx context.Context, namespace, question, answer string) (*airag.KnowledgeEntry, error) {
	if a == nil || a.knowledge == nil {
		return nil, ErrToolNotFound
	}
	entry, err := a.knowledge.AddUserKnowledge(ctx, namespace, question, answer)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// CollectFeedback 收集会话反馈。
// 反馈可用于改进知识库或作为训练数据。
//
// 参数:
//   - ctx: 上下文。
//   - sessionID: 会话 ID。
//   - namespace: 命名空间。
//   - feedback: 反馈类型（如点赞/点踩）。
//   - question: 问题内容。
//   - answer: 答案内容。
//
// 返回:
//   - *airag.KnowledgeEntry: 创建的知识条目（如果有）。
//   - error: 收集错误。
func (a *AIAgent) CollectFeedback(ctx context.Context, sessionID, namespace string, feedback airag.Feedback, question, answer string) (*airag.KnowledgeEntry, error) {
	if a == nil || a.knowledge == nil {
		return nil, ErrToolNotFound
	}
	if strings.TrimSpace(question) != "" || strings.TrimSpace(answer) != "" {
		entry := airag.KnowledgeEntry{
			ID:        "feedback-" + uuid.NewString(),
			Source:    airag.SourceFeedback,
			Namespace: strings.TrimSpace(namespace),
			Question:  strings.TrimSpace(question),
			Answer:    strings.TrimSpace(answer),
		}
		if err := a.knowledge.Index(ctx, []airag.KnowledgeEntry{entry}); err != nil {
			return nil, err
		}
		return &entry, nil
	}
	if a.feedback == nil {
		return nil, fmt.Errorf("feedback collector is not initialized")
	}
	return a.feedback.Collect(ctx, sessionID, namespace, feedback)
}

// StreamResult 存储流式执行的最终结果。
type StreamResult struct {
	// SessionID 是会话 ID。
	SessionID string
	// Content 是累积的响应内容。
	Content string
	// Thinking 是累积的思考链内容。
	Thinking string
	// ToolCalls 是执行的工具调用列表。
	ToolCalls []ToolCallInfo
	// CheckpointID 是检查点 ID，非空表示需要恢复。
	CheckpointID string
}

// ToolCallInfo 存储单个工具调用的信息。
type ToolCallInfo struct {
	// ID 是工具调用 ID。
	ID string
	// Name 是工具名称。
	Name string
	// Arguments 是工具参数的 JSON 字符串。
	Arguments string
}

// jsonStringToMap 将 JSON 字符串解析为 map。
// 解析失败时返回 nil。
//
// 参数:
//   - s: JSON 字符串。
//
// 返回:
//   - map[string]any: 解析后的 map。
func jsonStringToMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}
