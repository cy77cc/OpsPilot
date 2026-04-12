package tools

import (
	"sort"
	"strings"
)

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

type Catalog struct {
	entries []ToolMetadata
}

func NewCatalog(entries []ToolMetadata) Catalog {
	return Catalog{entries: entries}
}

func (c Catalog) Search(query string, limit int, domain string) []ToolMetadata {
	if limit <= 0 {
		limit = 5
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	type scored struct {
		entry ToolMetadata
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
		scoredEntries = append(scoredEntries, scored{entry: entry, score: score})
	}
	sort.SliceStable(scoredEntries, func(i, j int) bool {
		if scoredEntries[i].score == scoredEntries[j].score {
			return scoredEntries[i].entry.ToolName < scoredEntries[j].entry.ToolName
		}
		return scoredEntries[i].score > scoredEntries[j].score
	})
	if len(scoredEntries) > limit {
		scoredEntries = scoredEntries[:limit]
	}
	out := make([]ToolMetadata, 0, len(scoredEntries))
	for _, item := range scoredEntries {
		out = append(out, item.entry)
	}
	return out
}

func scoreEntry(query string, entry ToolMetadata) int {
	if strings.TrimSpace(query) == "" {
		return 0
	}
	text := strings.ToLower(strings.Join([]string{
		entry.ToolName,
		entry.Domain,
		entry.Capability,
		entry.RiskLevel,
		entry.Description,
		entry.AccessPath,
	}, " "))
	score := 0
	for _, token := range strings.Fields(strings.ToLower(query)) {
		if strings.Contains(text, token) {
			score++
		}
	}
	return score
}
