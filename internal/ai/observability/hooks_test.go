package observability

import "testing"

type testObserver struct {
	records     []ThoughtChainLifecycleRecord
	toolRecords []ToolExecutionLifecycleRecord
}

func (o *testObserver) OnThoughtChainLifecycle(record ThoughtChainLifecycleRecord) {
	o.records = append(o.records, record)
}

func (o *testObserver) OnToolExecutionLifecycle(record ToolExecutionLifecycleRecord) {
	o.toolRecords = append(o.toolRecords, record)
}

func TestObserveThoughtChainLifecycle_RegisterAndUnregister(t *testing.T) {
	observer := &testObserver{}
	unregister := RegisterObserver(observer)
	t.Cleanup(unregister)

	ObserveThoughtChainLifecycle(ThoughtChainLifecycleRecord{
		Event:     "chain_started",
		TraceID:   "trace-1",
		SessionID: "sess-1",
		ChainID:   "chain-1",
		NodeID:    "plan:chain-1",
		Scene:     "deployment:hosts",
		Tool:      "host_list_inventory",
		Kind:      "tool",
		Status:    "loading",
	})

	if len(observer.records) != 1 {
		t.Fatalf("expected one lifecycle record, got %d", len(observer.records))
	}
	if got := observer.records[0].SessionID; got != "sess-1" {
		t.Fatalf("expected session id to propagate, got %q", got)
	}

	unregister()
	ObserveThoughtChainLifecycle(ThoughtChainLifecycleRecord{Event: "chain_completed"})
	if len(observer.records) != 1 {
		t.Fatalf("expected observer to stop receiving records after unregister, got %d", len(observer.records))
	}
}

func TestObserveToolExecutionLifecycle(t *testing.T) {
	observer := &testObserver{}
	unregister := RegisterObserver(observer)
	t.Cleanup(unregister)

	ObserveToolExecutionLifecycle(ToolExecutionLifecycleRecord{
		Event:     "tool_call",
		TraceID:   "trace-1",
		SessionID: "sess-1",
		ToolName:  "get_pods",
		Status:    "pending",
	})

	if len(observer.toolRecords) != 1 {
		t.Fatalf("expected one tool lifecycle record, got %d", len(observer.toolRecords))
	}
	if got := observer.toolRecords[0].ToolName; got != "get_pods" {
		t.Fatalf("expected tool name to propagate, got %q", got)
	}

	unregister()
	ObserveToolExecutionLifecycle(ToolExecutionLifecycleRecord{Event: "tool_result"})
	if len(observer.toolRecords) != 1 {
		t.Fatalf("expected observer to stop receiving records after unregister, got %d", len(observer.toolRecords))
	}
}
