// Package metricsnoop provides a no-op metrics collector for testing.
package metricsnoop

import (
	"sync"
	"time"
)

// Metrics holds observed metric values.
type Metrics struct {
	ToolCallCount       int
	ToolCallErrors      int
	ToolCallDurationSum time.Duration
}

// Collector implements ObservabilityMetrics for testing.
type Collector struct {
	mu      sync.Mutex
	metrics Metrics
}

// New creates a new noop metrics collector.
func New() *Collector {
	return &Collector{}
}

func (c *Collector) RecordToolCall(toolName string, duration time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.ToolCallCount++
	c.metrics.ToolCallDurationSum += duration
	if err != nil {
		c.metrics.ToolCallErrors++
	}
}

func (c *Collector) Snapshot() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics
}
