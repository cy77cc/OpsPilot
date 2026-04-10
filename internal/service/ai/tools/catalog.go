package tools

import (
	"sort"
	"strings"
)

// ToolMetadata describes minimal searchable tool metadata.
type ToolMetadata struct {
	ToolName         string `json:"tool_name"`
	Domain           string `json:"domain"`
	Capability       string `json:"capability"`
	RiskLevel        string `json:"risk_level"`
	OutputMode       string `json:"output_mode"`
	Description      string `json:"description"`
	DirectlyCallable bool   `json:"directly_callable"`
	AccessPath       string `json:"access_path"`
}

// Catalog is an in-memory metadata-backed searchable tool catalog.
type Catalog struct {
	entries []ToolMetadata
}

// NewCatalog builds an in-memory searchable catalog from provided metadata.
func NewCatalog(entries []ToolMetadata) Catalog {
	return Catalog{entries: entries}
}

func entrySearchText(entry ToolMetadata) string {
	return strings.ToLower(strings.Join([]string{
		entry.ToolName,
		entry.Domain,
		entry.Capability,
		entry.RiskLevel,
		entry.Description,
		entry.AccessPath,
	}, " "))
}

func scoreEntry(query string, entry ToolMetadata) int {
	if strings.TrimSpace(query) == "" {
		return 0
	}
	text := entrySearchText(entry)
	score := 0
	for _, token := range strings.Fields(strings.ToLower(query)) {
		if strings.Contains(text, token) {
			score++
		}
	}
	return score
}

// Search returns top matched tool metadata items, filtered by domain before truncation.
func (c Catalog) Search(query string, limit int, domain string) []ToolMetadata {
	if limit <= 0 {
		limit = 5
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	type scored struct {
		tool  ToolMetadata
		score int
	}
	scoredEntries := make([]scored, 0, len(c.entries))
	for _, entry := range c.entries {
		if domain != "" && !strings.EqualFold(entry.Domain, domain) {
			continue
		}
		score := scoreEntry(query, entry)
		if strings.TrimSpace(query) != "" && score == 0 {
			continue
		}
		scoredEntries = append(scoredEntries, scored{tool: entry, score: score})
	}
	sort.SliceStable(scoredEntries, func(i, j int) bool {
		if scoredEntries[i].score == scoredEntries[j].score {
			return scoredEntries[i].tool.ToolName < scoredEntries[j].tool.ToolName
		}
		return scoredEntries[i].score > scoredEntries[j].score
	})
	if len(scoredEntries) > limit {
		scoredEntries = scoredEntries[:limit]
	}
	out := make([]ToolMetadata, 0, len(scoredEntries))
	for _, item := range scoredEntries {
		out = append(out, item.tool)
	}
	return out
}
