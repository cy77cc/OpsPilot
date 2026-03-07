package classifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type Intent string

const (
	IntentSimple  Intent = "simple"
	IntentAgentic Intent = "agentic"
)

const classifierPrompt = `你是一个意图分类器。根据用户输入，判断是否需要调用工具来完成任务。

需要调用工具的情况：
- 查询 K8s 资源（Pod、Service、Deployment 等）
- 查看日志、事件
- 执行主机命令
- 部署服务
- 查看监控指标、告警
- 操作资源（创建、删除、更新）

不需要调用工具的情况：
- 问候、闲聊
- 知识问答（如 "什么是 Pod"）
- 总结、解释已有信息
- 简单的澄清问题

用户输入: %s

只输出一个词: "simple" 或 "agentic"`

type IntentClassifier struct {
	model model.ToolCallingChatModel
}

func NewIntentClassifier(m model.ToolCallingChatModel) *IntentClassifier {
	return &IntentClassifier{model: m}
}

func (c *IntentClassifier) Classify(ctx context.Context, query string) (Intent, error) {
	if c == nil || c.model == nil {
		return IntentSimple, nil
	}

	resp, err := c.model.Generate(ctx, []*schema.Message{
		schema.UserMessage(fmt.Sprintf(classifierPrompt, strings.TrimSpace(query))),
	})
	if err != nil {
		return IntentSimple, nil
	}
	return parseIntent(resp), nil
}

func parseIntent(msg *schema.Message) Intent {
	if msg == nil {
		return IntentSimple
	}
	content := strings.ToLower(strings.TrimSpace(msg.Content))
	switch {
	case strings.Contains(content, string(IntentAgentic)):
		return IntentAgentic
	case strings.Contains(content, string(IntentSimple)):
		return IntentSimple
	default:
		return IntentSimple
	}
}
