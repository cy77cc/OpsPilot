package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// SearchToolsInput is the input for the search_tools dynamic discovery tool.
type SearchToolsInput struct {
	Query string `json:"query" jsonschema:"description=Search query to find relevant tools by name, description, or domain keywords"`
}

const searchToolsMaxResults = 10

// NewSearchToolsTool creates a tool that lets the agent discover tools on-demand.
// This reduces context token usage by up to 85% while improving tool selection accuracy.
// Reference: Anthropic "Building Multi-Agent Systems" — Tool Discovery pattern.
func NewSearchToolsTool(catalog Catalog) (tool.InvokableTool, error) {
	return utils.InferTool(
		"search_tools",
		"Search available tools by keyword or capability. Use this to discover tools before calling them. Returns matching tool names, descriptions, and domains.",
		func(ctx context.Context, input SearchToolsInput) (string, error) {
			query := strings.TrimSpace(input.Query)
			results := catalog.Search(query, searchToolsMaxResults, "")

			if len(results) == 0 {
				return "No tools found matching the query.", nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Found %d tool(s):\n", len(results)))
			for _, r := range results {
				sb.WriteString(fmt.Sprintf("- %s: %s [%s]\n", r.ToolName, r.Description, r.Domain))
			}
			return sb.String(), nil
		},
	)
}
