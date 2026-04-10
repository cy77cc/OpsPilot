package runtime

import "testing"

func TestKernelShouldCreateRun_ConversationBypassesRun(t *testing.T) {
	kernel := NewKernel()
	if kernel.ShouldCreateRun("conversation") {
		t.Fatal("conversation mode should bypass run creation")
	}
	if !kernel.ShouldCreateRun("operation") {
		t.Fatal("operation mode should create runs")
	}
}

func TestKernelResumeTransition_SameRunApprovalResume(t *testing.T) {
	kernel := NewKernel()
	next, err := kernel.ResumeTransition(RunStateWaitingApproval)
	if err != nil {
		t.Fatalf("resume transition: %v", err)
	}
	if next != RunStateResuming {
		t.Fatalf("expected resuming state, got %q", next)
	}
}

func TestKernelResumeTransition_RejectsInvalidSourceState(t *testing.T) {
	kernel := NewKernel()
	if _, err := kernel.ResumeTransition(RunStateCompleted); err == nil {
		t.Fatal("expected invalid resume transition from completed")
	}
}

func TestKernelDefaultExecutionShape_IsSingleAgent(t *testing.T) {
	kernel := NewKernel()
	if got := kernel.DefaultExecutionShape(); got != ExecutionShapeSingleAgent {
		t.Fatalf("expected %q, got %q", ExecutionShapeSingleAgent, got)
	}
}

func TestKernelBuildDispatchDecision_DelegatesOnlyWhenSpecialistAvailable(t *testing.T) {
	kernel := NewKernel()

	withoutSpecialist := kernel.BuildDispatchDecision("kubernetes", false)
	if withoutSpecialist.ExecutionShape != ExecutionShapeSingleAgent {
		t.Fatalf("expected single_agent when specialist unavailable, got %q", withoutSpecialist.ExecutionShape)
	}

	withSpecialist := kernel.BuildDispatchDecision("kubernetes", true)
	if withSpecialist.ExecutionShape != ExecutionShapeDelegatedSpecialist {
		t.Fatalf("expected delegated_specialist when specialist available, got %q", withSpecialist.ExecutionShape)
	}
}
