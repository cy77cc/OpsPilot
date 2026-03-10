package rewrite

import "testing"

func TestHeuristicRewriteTreatsAllHostsAsExplicitTarget(t *testing.T) {
	out := heuristicRewrite(Input{Message: "查看所有主机的状态"})
	if out.OperationMode != "query" {
		t.Fatalf("OperationMode = %s, want query", out.OperationMode)
	}
	if len(out.AmbiguityFlags) != 0 {
		t.Fatalf("AmbiguityFlags = %v, want empty", out.AmbiguityFlags)
	}
	if len(out.DomainHints) != 1 || out.DomainHints[0] != "hostops" {
		t.Fatalf("DomainHints = %v, want [hostops]", out.DomainHints)
	}
}

func TestRewriteConstrainsModelOutputForAllHosts(t *testing.T) {
	parsed := Output{
		NormalizedGoal: "查看所有主机的状态",
		OperationMode:  "query",
		DomainHints:    []string{"k8s", "hostops", "delivery"},
		AmbiguityFlags: []string{"resource_target_not_explicit"},
		Narrative:      "bad narrative",
	}
	fallback := heuristicRewrite(Input{Message: "查看所有主机的状态"})
	out := normalizeOutput(Input{Message: "查看所有主机的状态"}, parsed, fallback)
	if len(out.DomainHints) != 1 || out.DomainHints[0] != "hostops" {
		t.Fatalf("DomainHints = %v, want [hostops]", out.DomainHints)
	}
	if len(out.AmbiguityFlags) != 0 {
		t.Fatalf("AmbiguityFlags = %v, want empty", out.AmbiguityFlags)
	}
	if out.Narrative == "" || out.Narrative == "bad narrative" {
		t.Fatalf("Narrative = %q, want rebuilt narrative", out.Narrative)
	}
}
