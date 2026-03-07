// Package knowledge provides AI knowledge base functions
package knowledge

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FAQKnowledgeEntry represents a FAQ entry
type FAQKnowledgeEntry struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Content  string `json:"content"`
}

var (
	faqKnowledgeOnce sync.Once
	faqKnowledgeData []FAQKnowledgeEntry
)

// LoadFAQKnowledgeEntries loads FAQ knowledge from JSONL file
func LoadFAQKnowledgeEntries() []FAQKnowledgeEntry {
	faqKnowledgeOnce.Do(func() {
		candidates := []string{
			"docs/ai/ops-faq-100.jsonl",
			filepath.Join("..", "docs", "ai", "ops-faq-100.jsonl"),
			filepath.Join("..", "..", "docs", "ai", "ops-faq-100.jsonl"),
		}
		for _, path := range candidates {
			entries, err := readFAQKnowledgeJSONL(path)
			if err != nil || len(entries) == 0 {
				continue
			}
			faqKnowledgeData = entries
			return
		}
	})
	return faqKnowledgeData
}

func readFAQKnowledgeJSONL(path string) ([]FAQKnowledgeEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	entries := make([]FAQKnowledgeEntry, 0, 128)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item FAQKnowledgeEntry
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" {
			continue
		}
		entries = append(entries, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// MatchFAQKnowledge finds the best matching FAQ entry for a message
func MatchFAQKnowledge(msg string) (*FAQKnowledgeEntry, int) {
	return MatchFAQKnowledgeFromEntries(msg, LoadFAQKnowledgeEntries())
}

// MatchFAQKnowledgeFromEntries finds the best matching FAQ entry from a list
func MatchFAQKnowledgeFromEntries(msg string, entries []FAQKnowledgeEntry) (*FAQKnowledgeEntry, int) {
	if len(entries) == 0 {
		return nil, 0
	}
	query := NormalizeForMatch(msg)
	if query == "" {
		return nil, 0
	}

	bestIdx := -1
	bestScore := 0
	for idx := range entries {
		target := NormalizeForMatch(entries[idx].Question + " " + entries[idx].Answer)
		score := KeywordScore(query, target) + BigramScore(query, target)
		if score > bestScore {
			bestScore = score
			bestIdx = idx
		}
	}
	if bestIdx < 0 || bestScore < 2 {
		return nil, 0
	}
	return &entries[bestIdx], bestScore
}

// NormalizeForMatch normalizes a string for matching
func NormalizeForMatch(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"，", "",
		"。", "",
		"：", "",
		":", "",
		"？", "",
		"?", "",
		"！", "",
		"!", "",
		"（", "",
		"）", "",
		"(", "",
		")", "",
		"-", "",
		"_", "",
		"/", "",
	)
	return replacer.Replace(s)
}

// KeywordScore calculates keyword match score
func KeywordScore(query, target string) int {
	score := 0
	markers := []string{"登录", "告警", "发布", "回滚", "权限", "rbac", "主机", "k8s", "任务", "配置", "值班", "应急", "排查"}
	for _, marker := range markers {
		if strings.Contains(query, marker) && strings.Contains(target, marker) {
			score++
		}
	}
	return score
}

// BigramScore calculates bigram match score
func BigramScore(query, target string) int {
	runes := []rune(query)
	if len(runes) < 2 {
		return 0
	}
	seen := map[string]struct{}{}
	score := 0
	for i := 0; i < len(runes)-1; i++ {
		bg := string(runes[i : i+2])
		if _, ok := seen[bg]; ok {
			continue
		}
		seen[bg] = struct{}{}
		if strings.Contains(target, bg) {
			score++
		}
	}
	return score
}
