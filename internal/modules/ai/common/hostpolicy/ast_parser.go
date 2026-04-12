package hostpolicy

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ParsedCommand contains the AST-derived summary used by policy validation.
type ParsedCommand struct {
	BaseCommands           []string
	HasRedirection         bool
	HasBackground          bool
	HasCommandSubstitution bool
}

// ParseCommand parses shell text into a policy-friendly summary.
func ParseCommand(command string) (*ParsedCommand, error) {
	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, fmt.Errorf("parse command: %w", err)
	}

	parsed := &ParsedCommand{}
	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.Redirect:
			parsed.HasRedirection = true
		case *syntax.Stmt:
			if n.Background {
				parsed.HasBackground = true
			}
		case *syntax.CmdSubst:
			parsed.HasCommandSubstitution = true
		case *syntax.CallExpr:
			name := firstLiteralArg(n)
			if name != "" {
				parsed.BaseCommands = append(parsed.BaseCommands, strings.ToLower(name))
			}
		}
		return true
	})
	return parsed, nil
}

func firstLiteralArg(call *syntax.CallExpr) string {
	if call == nil || len(call.Args) == 0 {
		return ""
	}
	for _, part := range call.Args[0].Parts {
		lit, ok := part.(*syntax.Lit)
		if ok {
			return strings.TrimSpace(lit.Value)
		}
	}
	return ""
}
