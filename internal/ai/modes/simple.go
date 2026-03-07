package modes

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/types"
)

const simpleChatInstruction = `你是一个专业的运维助手。

当前模式是简单问答模式：
- 直接回答用户问题
- 不要调用工具
- 不要编造执行结果
- 回答保持简洁、准确、可操作`

type SimpleChatMode struct {
	model model.ToolCallingChatModel
}

func NewSimpleChatMode(m model.ToolCallingChatModel) *SimpleChatMode {
	return &SimpleChatMode{model: m}
}

func (m *SimpleChatMode) Execute(ctx context.Context, message string, gen *adk.AsyncGenerator[*types.AgentResult]) {
	if gen == nil {
		return
	}
	if m == nil || m.model == nil {
		gen.Send(&types.AgentResult{Type: "error", Content: "chat model not initialized"})
		return
	}

	resp, err := m.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(simpleChatInstruction),
		schema.UserMessage(strings.TrimSpace(message)),
	})
	if err != nil {
		gen.Send(&types.AgentResult{Type: "error", Content: err.Error()})
		return
	}

	content := ""
	if resp != nil {
		content = strings.TrimSpace(resp.Content)
	}
	if content == "" {
		content = "无输出。"
	}
	gen.Send(&types.AgentResult{Type: "text", Content: content})
}
