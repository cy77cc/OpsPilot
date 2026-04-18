package sequenceguard

import (
	"hash/fnv"
	"sync"
)

const stripeCount = 64

var stripes [stripeCount]sync.Mutex

// LockKey serializes in-process sequence allocation for a stable key such as
// a chat session ID or run ID. It returns an unlock closure.
func LockKey(key string) func() {
	idx := stripeIndex(key)
	stripes[idx].Lock()
	return func() {
		stripes[idx].Unlock()
	}
}

func stripeIndex(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32() % stripeCount
}
