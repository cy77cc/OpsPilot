package toolutil

import (
	"regexp"
	"strings"
)

// EscapeLikePattern escapes SQL LIKE metacharacters (%, _, \) in user input
// so they are treated as literal characters rather than wildcards.
func EscapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

var dangerousCommandPatterns = regexp.MustCompile(
	`(?:^|;\s*|&&\s*|\|\s*|\x60)\s*rm\s+-rf?\s+/` +
		`|(?:^|;\s*|&&\s*|\|\s*)\s*dd\s+if=` +
		`|(?:^|;\s*|&&\s*|\|\s*)\s*mkfs` +
		`|>\s*/dev/sd` +
		`|(?:^|;\s*|&&\s*|\|\s*)\s*chmod\s+777\s+/` +
		`|:\(\)\{|^\(\)\{`, // fork bomb
)

// ValidateCommandSafety checks a shell command for dangerous patterns that
// could cause destructive operations on the local machine. Returns a list of
// violation descriptions; an empty slice means the command passed validation.
func ValidateCommandSafety(cmd string) []string {
	if dangerousCommandPatterns.MatchString(cmd) {
		return []string{"command contains potentially destructive patterns (rm -rf /, dd, mkfs, fork bomb, etc.)"}
	}
	return nil
}
