package middleware

import "testing"

func TestCommandClassForTool_HostExec(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     string
		want     string
	}{
		{
			name:     "readonly command via AST",
			toolName: "host_exec",
			args:     `{"command": "df -h"}`,
			want:     "readonly",
		},
		{
			name:     "non-readonly command",
			toolName: "host_exec",
			args:     `{"command": "systemctl restart nginx"}`,
			want:     "service_control",
		},
		{
			name:     "empty command",
			toolName: "host_exec",
			args:     `{"command": ""}`,
			want:     "unknown",
		},
		{
			name:     "delete tool name",
			toolName: "k8s_delete_pod",
			args:     `{}`,
			want:     "write",
		},
		{
			name:     "read tool name",
			toolName: "k8s_query",
			args:     `{}`,
			want:     "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandClassForTool(tt.toolName, tt.args)
			if got != tt.want {
				t.Errorf("commandClassForTool(%q, %q) = %q, want %q", tt.toolName, tt.args, got, tt.want)
			}
		})
	}
}
