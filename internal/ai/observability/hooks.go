package observability

import "sync"

// ThoughtChainLifecycleRecord 记录思维链生命周期事件。
// 保留用于向后兼容和 metrics 收集。
type ThoughtChainLifecycleRecord struct {
	Event     string
	TraceID   string
	SessionID string
	ChainID   string
	NodeID    string
	Scene     string
	Tool      string
	Kind      string
	Status    string
}

// ToolExecutionLifecycleRecord 记录工具执行生命周期事件。
// 用于简化后的 tool_call -> tool_approval -> tool_result 流程。
type ToolExecutionLifecycleRecord struct {
	Event     string
	TraceID   string
	SessionID string
	ToolName  string
	Status    string
}

type Observer interface {
	OnThoughtChainLifecycle(ThoughtChainLifecycleRecord)
	OnToolExecutionLifecycle(ToolExecutionLifecycleRecord)
}

var (
	observersMu sync.RWMutex
	observers   []Observer
)

func RegisterObserver(observer Observer) func() {
	if observer == nil {
		return func() {}
	}
	observersMu.Lock()
	observers = append(observers, observer)
	index := len(observers) - 1
	observersMu.Unlock()
	return func() {
		observersMu.Lock()
		defer observersMu.Unlock()
		if index < 0 || index >= len(observers) || observers[index] != observer {
			for i := range observers {
				if observers[i] == observer {
					observers = append(observers[:i], observers[i+1:]...)
					return
				}
			}
			return
		}
		observers = append(observers[:index], observers[index+1:]...)
	}
}

func ObserveThoughtChainLifecycle(record ThoughtChainLifecycleRecord) {
	observersMu.RLock()
	snapshot := append([]Observer(nil), observers...)
	observersMu.RUnlock()
	for _, observer := range snapshot {
		observer.OnThoughtChainLifecycle(record)
	}
}

func ObserveToolExecutionLifecycle(record ToolExecutionLifecycleRecord) {
	observersMu.RLock()
	snapshot := append([]Observer(nil), observers...)
	observersMu.RUnlock()
	for _, observer := range snapshot {
		observer.OnToolExecutionLifecycle(record)
	}
}
