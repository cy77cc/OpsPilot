package toolutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type unavailableTool struct {
	name    string
	message string
}

// UnavailableInvokableTool returns a non-panicking placeholder tool that
// preserves the tool name in the catalog but fails deterministically when
// invoked.
func UnavailableInvokableTool(name string, err error) tool.InvokableTool {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = "unavailable_tool"
	}
	message := fmt.Sprintf("tool %q is unavailable", trimmedName)
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = fmt.Sprintf("%s: %s", message, strings.TrimSpace(err.Error()))
	}
	return &unavailableTool{
		name:    trimmedName,
		message: message,
	}
}

func (t *unavailableTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: "Temporarily unavailable tool placeholder.",
	}, nil
}

func (t *unavailableTool) InvokableRun(context.Context, string, ...tool.Option) (string, error) {
	return "", fmt.Errorf("%s", t.message)
}
