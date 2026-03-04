package experts

import (
	"context"
	"strings"
	"testing"
)

func TestResultAggregatorTemplate(t *testing.T) {
	agg := NewResultAggregator(AggregationTemplate, nil)
	out, err := agg.Aggregate(context.Background(), []ExpertResult{
		{ExpertName: "k8s_expert", Output: "pod crashloop"},
		{ExpertName: "monitor_expert", Output: "latency spike"},
	}, "check")
	if err != nil {
		t.Fatalf("aggregate template: %v", err)
	}
	if !strings.Contains(out, "k8s_expert") || !strings.Contains(out, "monitor_expert") {
		t.Fatalf("unexpected aggregate output: %s", out)
	}
}
