package runtime

import "testing"

func TestResumeIdentityRequiresCanonicalKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		planID    string
		stepID    string
		want      string
	}{
		{
			name:      "builds identity from trimmed fields",
			sessionID: " session-1 ",
			planID:    " plan-1 ",
			stepID:    " step-1 ",
			want:      "session-1:plan-1:step-1",
		},
		{
			name:      "requires session id",
			planID:    "plan-1",
			stepID:    "step-1",
			want:      "",
		},
		{
			name:      "requires plan id",
			sessionID: "session-1",
			stepID:    "step-1",
			want:      "",
		},
		{
			name:      "requires step id",
			sessionID: "session-1",
			planID:    "plan-1",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ResumeIdentity(tt.sessionID, tt.planID, tt.stepID); got != tt.want {
				t.Fatalf("ResumeIdentity(%q, %q, %q) = %q, want %q", tt.sessionID, tt.planID, tt.stepID, got, tt.want)
			}
		})
	}
}

func TestResolvedSceneEffectiveAllowedToolsPrefersOverrideAndReturnsClone(t *testing.T) {
	t.Parallel()

	scene := ResolvedScene{
		AllowedTools: []string{" tool-a ", "tool-b"},
		SceneConfig: SceneConfig{
			AllowedTools: []string{"tool-c"},
		},
	}

	got := scene.EffectiveAllowedTools()
	if len(got) != 2 || got[0] != "tool-a" || got[1] != "tool-b" {
		t.Fatalf("EffectiveAllowedTools() = %#v", got)
	}

	got[0] = "changed"
	if scene.AllowedTools[0] != " tool-a " {
		t.Fatalf("EffectiveAllowedTools() did not return a clone, original = %#v", scene.AllowedTools)
	}

	scene.AllowedTools = nil
	got = scene.EffectiveAllowedTools()
	if len(got) != 1 || got[0] != "tool-c" {
		t.Fatalf("EffectiveAllowedTools() fallback = %#v", got)
	}
}
