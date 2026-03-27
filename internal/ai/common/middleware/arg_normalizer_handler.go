package middleware

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type argNormalizationHandler struct {
	*adk.BaseChatModelAgentMiddleware
	config  *ArgNormalizeConfig
	schemas map[string]*schema.ParamsOneOf
}

// NewArgNormalizationHandler creates a ChatModelAgent middleware that normalizes tool arguments.
func NewArgNormalizationHandler(ctx context.Context, tools []tool.BaseTool, cfg *ArgNormalizeConfig) (adk.ChatModelAgentMiddleware, error) {
	if cfg == nil {
		cfg = &ArgNormalizeConfig{
			Enabled:    false,
			ShadowMode: true,
		}
	}

	registry := make(map[string]*schema.ParamsOneOf, len(tools))
	for _, item := range tools {
		if item == nil {
			continue
		}
		info, err := item.Info(ctx)
		if err != nil {
			return nil, err
		}
		if info == nil || info.Name == "" || info.ParamsOneOf == nil {
			continue
		}
		registry[info.Name] = info.ParamsOneOf
	}

	return &argNormalizationHandler{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		config:                       cfg,
		schemas:                      registry,
	}, nil
}

func (m *argNormalizationHandler) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx == nil {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		nextCtx, normalizedArgs, err := m.normalizeCall(ctx, tCtx.Name, args)
		if err != nil {
			return "", err
		}
		return endpoint(nextCtx, normalizedArgs, opts...)
	}, nil
}

func (m *argNormalizationHandler) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	if tCtx == nil {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		nextCtx, normalizedArgs, err := m.normalizeCall(ctx, tCtx.Name, args)
		if err != nil {
			return nil, err
		}
		return endpoint(nextCtx, normalizedArgs, opts...)
	}, nil
}

func (m *argNormalizationHandler) normalizeCall(ctx context.Context, toolName, raw string) (context.Context, string, error) {
	params := m.schemas[toolName]
	if params == nil {
		return ctx, raw, nil
	}
	if !m.config.Enabled && !m.config.ShadowMode && m.config.Reporter == nil {
		return ctx, raw, nil
	}

	result, err := NormalizeToolArgs(toolName, raw, params)
	if err != nil {
		return ctx, raw, err
	}
	result.Metadata.Enabled = m.config.Enabled
	result.Metadata.ShadowMode = m.config.ShadowMode

	ctx = WithNormalizationMetadata(ctx, result.Metadata)
	if m.config.Reporter != nil {
		m.config.Reporter(ctx, result.Metadata)
	}
	if m.config.Enabled {
		return ctx, result.NormalizedJSON, nil
	}
	return ctx, raw, nil
}
