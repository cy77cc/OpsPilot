package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	SessionKeyRuntimeContext = "ai.runtime_context"
	SessionKeyResolvedScene  = "ai.runtime_scene"
	SessionKeySessionID      = "ai.session_id"
	SessionKeyPlanID         = "ai.plan_id"
	SessionKeyTurnID         = "ai.turn_id"
)

type runtimeContextKey struct{}

type ContextProcessor struct {
	resolver        *SceneConfigResolver
	plannerPrompt   prompt.ChatTemplate
	executorPrompt  prompt.ChatTemplate
	replannerPrompt prompt.ChatTemplate
}

func NewContextProcessor(resolver *SceneConfigResolver) *ContextProcessor {
	return &ContextProcessor{
		resolver:        resolver,
		plannerPrompt:   plannerInputPrompt,
		executorPrompt:  executorInputPrompt,
		replannerPrompt: replannerInputPrompt,
	}
}

func (p *ContextProcessor) ResolveScene(ctx context.Context) ResolvedScene {
	if value, ok := adk.GetSessionValue(ctx, SessionKeyResolvedScene); ok {
		if scene, ok := value.(ResolvedScene); ok {
			return scene
		}
	}
	runtimeCtx := p.RuntimeContextFromSession(ctx)
	if p == nil || p.resolver == nil {
		return defaultSceneConfigResolver().Resolve(runtimeCtx.Scene)
	}
	return p.resolver.Resolve(runtimeCtx.Scene)
}

func (p *ContextProcessor) RuntimeContextFromSession(ctx context.Context) RuntimeContext {
	value, ok := adk.GetSessionValue(ctx, SessionKeyRuntimeContext)
	if !ok {
		runtimeCtx, _ := ctx.Value(runtimeContextKey{}).(RuntimeContext)
		return runtimeCtx
	}
	runtimeCtx, _ := value.(RuntimeContext)
	return runtimeCtx
}

func ContextWithRuntimeContext(ctx context.Context, runtimeCtx RuntimeContext) context.Context {
	return context.WithValue(ctx, runtimeContextKey{}, runtimeCtx)
}

func RuntimeContextFromContext(ctx context.Context) RuntimeContext {
	runtimeCtx, _ := ctx.Value(runtimeContextKey{}).(RuntimeContext)
	return runtimeCtx
}

func (p *ContextProcessor) BuildPlannerInput(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
	scene := p.ResolveScene(ctx)
	runtimeCtx := p.RuntimeContextFromSession(ctx)
	msgs, err := p.plannerPrompt.Format(ctx, map[string]any{
		"scene_name":         firstNonEmpty(runtimeCtx.SceneName, scene.SceneConfig.Name, scene.Domain),
		"scene_description":  scene.SceneConfig.Description,
		"scene_constraints":  formatLines(scene.Constraints),
		"tool_names":         formatList(scene.EffectiveAllowedTools()),
		"selected_resources": formatResources(runtimeCtx.SelectedResources),
		"context_summary":    p.runtimeSummary(runtimeCtx),
		"user_input":         formatMessages(userInput),
		"examples":           formatList(scene.ExampleIDs),
	})
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (p *ContextProcessor) BuildExecutorInput(ctx context.Context, in *planexecute.ExecutionContext, tools []tool.BaseTool) ([]adk.Message, error) {
	scene := p.ResolveScene(ctx)
	runtimeCtx := p.RuntimeContextFromSession(ctx)
	planContent, err := in.Plan.MarshalJSON()
	if err != nil {
		return nil, err
	}
	msgs, err := p.executorPrompt.Format(ctx, map[string]any{
		"scene_name":         firstNonEmpty(runtimeCtx.SceneName, scene.SceneConfig.Name, scene.Domain),
		"scene_description":  scene.SceneConfig.Description,
		"scene_constraints":  formatLines(scene.Constraints),
		"context_summary":    p.runtimeSummary(runtimeCtx),
		"tool_names":         formatList(p.FilterTools(ctx, scene, tools)),
		"selected_resources": formatResources(runtimeCtx.SelectedResources),
		"plan":               string(planContent),
		"executed_steps":     formatExecutedSteps(in.ExecutedSteps),
		"step":               in.Plan.FirstStep(),
		"user_input":         formatMessages(in.UserInput),
	})
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (p *ContextProcessor) BuildReplannerInput(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	scene := p.ResolveScene(ctx)
	runtimeCtx := p.RuntimeContextFromSession(ctx)
	planContent, err := in.Plan.MarshalJSON()
	if err != nil {
		return nil, err
	}
	msgs, err := p.replannerPrompt.Format(ctx, map[string]any{
		"scene_name":        firstNonEmpty(runtimeCtx.SceneName, scene.SceneConfig.Name, scene.Domain),
		"scene_description": scene.SceneConfig.Description,
		"scene_constraints": formatLines(scene.Constraints),
		"context_summary":   p.runtimeSummary(runtimeCtx),
		"plan":              string(planContent),
		"executed_steps":    formatExecutedSteps(in.ExecutedSteps),
		"user_input":        formatMessages(in.UserInput),
	})
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (p *ContextProcessor) FilterTools(ctx context.Context, scene ResolvedScene, tools []tool.BaseTool) []string {
	allowed := make(map[string]struct{})
	blocked := make(map[string]struct{})
	for _, name := range scene.EffectiveAllowedTools() {
		allowed[normalizeToolName(name)] = struct{}{}
	}
	for _, name := range scene.BlockedTools {
		blocked[normalizeToolName(name)] = struct{}{}
	}
	for _, name := range scene.SceneConfig.BlockedTools {
		blocked[normalizeToolName(name)] = struct{}{}
	}

	out := make([]string, 0, len(tools))
	for _, tl := range tools {
		if tl == nil {
			continue
		}
		info, err := tl.Info(ctx)
		if err != nil || info == nil {
			continue
		}
		name := strings.TrimSpace(info.Name)
		if name == "" {
			continue
		}
		normalized := normalizeToolName(name)
		if _, denied := blocked[normalized]; denied {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[normalized]; !ok {
				continue
			}
		}
		out = append(out, name)
	}
	return out
}

func (p *ContextProcessor) runtimeSummary(runtimeCtx RuntimeContext) string {
	lines := []string{
		fmt.Sprintf("Scene: %s", firstNonEmpty(runtimeCtx.SceneName, runtimeCtx.Scene, "global")),
		fmt.Sprintf("Route: %s", firstNonEmpty(runtimeCtx.Route, "/")),
		fmt.Sprintf("Project: %s", firstNonEmpty(runtimeCtx.ProjectName, runtimeCtx.ProjectID, "unknown")),
		fmt.Sprintf("Page: %s", firstNonEmpty(runtimeCtx.CurrentPage, "unknown")),
	}
	if len(runtimeCtx.UserContext) > 0 {
		lines = append(lines, fmt.Sprintf("User Context: %v", runtimeCtx.UserContext))
	}
	return strings.Join(lines, "\n")
}

func formatResources(resources []SelectedResource) string {
	if len(resources) == 0 {
		return "none"
	}
	rows := make([]string, 0, len(resources))
	for _, resource := range resources {
		rows = append(rows, strings.TrimSpace(fmt.Sprintf("%s:%s:%s", resource.Type, resource.Namespace, firstNonEmpty(resource.Name, resource.ID))))
	}
	return strings.Join(rows, "\n")
}

func formatMessages(input []adk.Message) string {
	if len(input) == 0 {
		return ""
	}
	rows := make([]string, 0, len(input))
	for _, msg := range input {
		rows = append(rows, msg.Content)
	}
	return strings.Join(rows, "\n")
}

func formatExecutedSteps(results []planexecute.ExecutedStep) string {
	if len(results) == 0 {
		return "none"
	}
	rows := make([]string, 0, len(results))
	for _, result := range results {
		rows = append(rows, fmt.Sprintf("Step: %s\nResult: %s", result.Step, result.Result))
	}
	return strings.Join(rows, "\n\n")
}

func formatLines(lines []string) string {
	if len(lines) == 0 {
		return "none"
	}
	return "- " + strings.Join(lines, "\n- ")
}

func normalizeToolName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	return strings.ToLower(name)
}

func formatList(items []string) string {
	items = cloneStrings(items)
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

var plannerInputPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are the runtime planner for an operations assistant.

Use the runtime context, scene constraints, and selected resources to produce a practical step-by-step plan.

Scene: {scene_name}
Description: {scene_description}
Constraints:
{scene_constraints}

Available tools: {tool_names}
Relevant examples: {examples}`),
	schema.UserMessage(`Runtime context:
{context_summary}

Selected resources:
{selected_resources}

User request:
{user_input}`))

var executorInputPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are executing one plan step inside a scene-aware runtime.

Scene: {scene_name}
Description: {scene_description}
Constraints:
{scene_constraints}

Available tools: {tool_names}`),
	schema.UserMessage(`Runtime context:
{context_summary}

Selected resources:
{selected_resources}

Original user request:
{user_input}

Plan:
{plan}

Executed steps:
{executed_steps}

Current step:
{step}`))

var replannerInputPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are revising a running operations plan.

Scene: {scene_name}
Description: {scene_description}
Constraints:
{scene_constraints}`),
	schema.UserMessage(`Runtime context:
{context_summary}

Original user request:
{user_input}

Current plan:
{plan}

Executed steps:
{executed_steps}

Decide whether to submit the final result, continue with the remaining plan, or create a revised plan.`))
