package observability

import "sync"

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

type Observer interface {
	OnThoughtChainLifecycle(ThoughtChainLifecycleRecord)
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
