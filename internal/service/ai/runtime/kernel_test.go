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
