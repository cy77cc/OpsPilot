package diagnosis

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

func TestFormatExecutedSteps_CompressesHistory(t *testing.T) {
	steps := make([]planexecute.ExecutedStep, 0, 7)
	for i := 1; i <= 7; i++ {
		steps = append(steps, planexecute.ExecutedStep{
			Step:   fmt.Sprintf("step-%d", i),
			Result: strings.Repeat(fmt.Sprintf("observation-%d ", i), 80),
		})
	}

	got := formatExecutedSteps(steps)

	if !strings.Contains(got, "Completed 7 step(s). Showing the latest 5 step(s).") {
		t.Fatalf("expected compact summary header, got %q", got)
	}
	if strings.Contains(got, "step-1") || strings.Contains(got, "step-2") {
		t.Fatalf("expected older steps to be omitted, got %q", got)
	}
	for _, want := range []string{"step-3", "step-4", "step-5", "step-6", "step-7"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in compact history, got %q", want, got)
		}
	}
	if !strings.Contains(got, "...<truncated>") {
		t.Fatalf("expected long tool results to be truncated, got %q", got)
	}
}
