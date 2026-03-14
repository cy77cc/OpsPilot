package observability

import "testing"

type testObserver struct {
	records []ThoughtChainLifecycleRecord
}

func (o *testObserver) OnThoughtChainLifecycle(record ThoughtChainLifecycleRecord) {
	o.records = append(o.records, record)
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
