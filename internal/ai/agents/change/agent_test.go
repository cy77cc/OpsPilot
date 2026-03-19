package change

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

func TestFormatExecutedSteps_CompressesHistory(t *testing.T) {
	steps := make([]planexecute.ExecutedStep, 0, 8)
	for i := 1; i <= 8; i++ {
		steps = append(steps, planexecute.ExecutedStep{
			Step:   fmt.Sprintf("step-%d", i),
			Result: strings.Repeat(fmt.Sprintf("result-%d ", i), 80),
		})
	}

	got := formatExecutedSteps(steps)

	if !strings.Contains(got, "Completed 8 step(s). Showing the latest 5 step(s).") {
		t.Fatalf("expected compact summary header, got %q", got)
	}
	if strings.Contains(got, "step-1") || strings.Contains(got, "step-2") || strings.Contains(got, "step-3") {
		t.Fatalf("expected older steps to be omitted, got %q", got)
	}
	for _, want := range []string{"step-4", "step-5", "step-6", "step-7", "step-8"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in compact history, got %q", want, got)
		}
	}
	if !strings.Contains(got, "...<truncated>") {
		t.Fatalf("expected long tool results to be truncated, got %q", got)
	}
}
