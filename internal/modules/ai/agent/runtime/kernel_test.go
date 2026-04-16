package runtime

import "testing"

func TestRunState_IsDelegationAware(t *testing.T) {
	if !RunStateDelegating.Known() {
		t.Fatal("expected delegating to be a known run state")
	}
	if !RunStateWaitingSubagent.Known() {
		t.Fatal("expected waiting_subagent to be a known run state")
	}
}

func TestBuildDispatchDecision_MonitoringSpecialistOnly(t *testing.T) {
	kernel := NewKernel()

	delegated := kernel.BuildDispatchDecision("monitoring", true)
	if delegated.ExecutionShape != ExecutionShapeDelegatedSpecialist {
		t.Fatalf("expected delegated specialist shape, got %q", delegated.ExecutionShape)
	}

	singleNoSpecialist := kernel.BuildDispatchDecision("monitoring", false)
	if singleNoSpecialist.ExecutionShape != ExecutionShapeSingleAgent {
		t.Fatalf("expected single agent shape without specialist, got %q", singleNoSpecialist.ExecutionShape)
	}

	singleOtherScene := kernel.BuildDispatchDecision("host", true)
	if singleOtherScene.ExecutionShape != ExecutionShapeSingleAgent {
		t.Fatalf("expected single agent shape for non-monitoring scene, got %q", singleOtherScene.ExecutionShape)
	}
}
