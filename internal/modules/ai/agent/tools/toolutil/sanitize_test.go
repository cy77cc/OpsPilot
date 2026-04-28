package toolutil

import "testing"

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain text", "hello", "hello"},
		{"percent", "100%", `100\%`},
		{"underscore", "my_table", `my\_table`},
		{"backslash", `path\to`, `path\\to`},
		{"all metacharacters", `%_\`, `\%\_\\`},
		{"sql injection attempt", `%'; DROP TABLE--`, `\%'; DROP TABLE--`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeLikePattern(tt.input)
			if got != tt.want {
				t.Errorf("EscapeLikePattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateCommandSafety(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		wantEmpty bool
	}{
		{"safe ls", "ls -la /tmp", true},
		{"safe ps", "ps aux", true},
		{"safe df", "df -h", true},
		{"rm rf root", "rm -rf /", false},
		{"rm rf root via semicolon", "echo ok; rm -rf /", false},
		{"rm rf via pipe", "cat file | rm -rf /", false},
		{"dd if", "dd if=/dev/zero of=/dev/sda", false},
		{"mkfs", "mkfs.ext4 /dev/sda1", false},
		{"chmod 777 root", "chmod 777 /", false},
		{"fork bomb", "(){:|:&};:", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := ValidateCommandSafety(tt.cmd)
			gotEmpty := len(violations) == 0
			if gotEmpty != tt.wantEmpty {
				t.Errorf("ValidateCommandSafety(%q) empty=%v, wantEmpty=%v, violations=%v",
					tt.cmd, gotEmpty, tt.wantEmpty, violations)
			}
		})
	}
}
