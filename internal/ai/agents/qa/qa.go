// Package qa 实现基于 RAG 的知识库问答助手。
//
// 架构：ChatModelAgent + search_knowledge 工具（封装 RAGRetriever）
// 模型通过调用 search_knowledge 工具主动检索相关文档，再综合生成回答。
package qa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/prompt"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/history"
)

// searchKnowledgeInput 是 search_knowledge 工具的输入参数。
type searchKnowledgeInput struct {
	// Query 检索关键词，应提炼自用户问题的核心意图。
	Query string `json:"query" jsonschema_description:"search query extracted from the user's question"`
	// TopK 返回的最大结果数，默认 6。
	TopK int `json:"top_k,omitempty" jsonschema_description:"max results to return, default 6"`
}

// searchKnowledgeOutput 是 search_knowledge 工具的输出结构。
type searchKnowledgeOutput struct {
	// Context 格式化后的增强上下文文本，直接供模型引用。
	Context string `json:"context"`
}

// newSearchKnowledgeTool 将 RAGRetriever 封装为 Eino 工具。
//
// 参数:
//   - retriever: RAG 检索器实例，nil 时工具返回空上下文（不报错，降级为纯模型回答）
//
// 返回: Eino InvokableTool
// func newSearchKnowledgeTool(retriever *rag.RAGRetriever) (tool.BaseTool, error) {
// 	return einoutils.InferTool(
// 		"search_knowledge",
// 		"Search the OpsPilot knowledge base for documentation, troubleshooting cases, and platform asset information. Call this before answering any technical question.",
// 		func(ctx context.Context, input *searchKnowledgeInput) (*searchKnowledgeOutput, error) {
// 			if retriever == nil || strings.TrimSpace(input.Query) == "" {
// 				return &searchKnowledgeOutput{Context: ""}, nil
// 			}
// 			topK := input.TopK
// 			if topK <= 0 {
// 				topK = 6
// 			}
// 			ragCtx, err := retriever.Retrieve(ctx, input.Query, topK)
// 			if err != nil {
// 				// 检索失败时降级为空上下文，不中断问答流程
// 				return &searchKnowledgeOutput{Context: ""}, nil
// 			}
// 			augmented := retriever.BuildAugmentedPrompt(input.Query, ragCtx)
// 			return &searchKnowledgeOutput{Context: augmented}, nil
// 		},
// 	)
// }

// NewQAAgent 创建 QA 问答 Agent 实例。
//
// 参数:
//   - ctx:       上下文
//   - retriever: RAG 检索器（nil 时降级为纯模型问答，不报错）
//
// 返回: Eino Agent 和初始化错误
func NewQAAgent(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("qa agent: init model: %w", err)
	}

	// TODO
	// searchTool, err := newSearchKnowledgeTool(retriever)
	// if err != nil {
	// 	return nil, fmt.Errorf("qa agent: build search tool: %w", err)
	// }
	toolset := []tool.BaseTool{
		history.LoadSessionHistory(ctx),
	}
	normalizerMW, err := tools.EnabledArgNormalizationToolMiddleware(ctx, toolset)
	if err != nil {
		return nil, fmt.Errorf("qa agent: init tool normalization middleware: %w", err)
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "QAAgent",
		Description: "Knowledge base Q&A assistant for Kubernetes and platform operations",
		Instruction: prompt.QA_SYSTEM,
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               toolset,
				ToolCallMiddlewares: []compose.ToolMiddleware{normalizerMW},
			},
		},
		MaxIterations: 5,
	})
}

// formatRAGContext 将 RAGContext 序列化为可读字符串，供调试使用。
func formatRAGContext(ragCtx any) string {
	b, err := json.Marshal(ragCtx)
	if err != nil {
		return ""
	}
	return string(b)
}
