package observability

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetrics_ObserveThoughtChain(t *testing.T) {
	metrics := newMetrics()

	metrics.ObserveThoughtChain(ThoughtChainRecord{
		Scene:    "k8s",
		Status:   "completed",
		Duration: 2 * time.Second,
	})
	metrics.ObserveThoughtChainNode(ThoughtChainNodeRecord{
		Scene:    "k8s",
		Kind:     "plan",
		Status:   "done",
		Duration: 500 * time.Millisecond,
	})
	metrics.ObserveThoughtChainApproval(ThoughtChainApprovalRecord{
		Scene:    "k8s",
		Status:   "approved",
		Duration: 3 * time.Second,
	})

	if got := counterValue(t, metrics.thoughtChains.WithLabelValues("k8s", "completed")); got != 1 {
		t.Fatalf("unexpected thoughtchain counter: got %v want 1", got)
	}
	if got := counterValue(t, metrics.thoughtChainNodes.WithLabelValues("k8s", "plan", "done")); got != 1 {
		t.Fatalf("unexpected node counter: got %v want 1", got)
	}
	if got := counterValue(t, metrics.thoughtApprovals.WithLabelValues("k8s", "approved")); got != 1 {
		t.Fatalf("unexpected approval counter: got %v want 1", got)
	}
	if got := histogramCount(t, metrics.thoughtChainLatency); got != 1 {
		t.Fatalf("unexpected thoughtchain histogram count: got %d want 1", got)
	}
	if got := histogramCount(t, metrics.thoughtNodeLatency); got != 1 {
		t.Fatalf("unexpected node histogram count: got %d want 1", got)
	}
	if got := histogramCount(t, metrics.approvalWaitLatency); got != 1 {
		t.Fatalf("unexpected approval histogram count: got %d want 1", got)
	}
}

func counterValue(t *testing.T, collector interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := collector.Write(metric); err != nil {
		t.Fatalf("write counter metric: %v", err)
	}
	return metric.GetCounter().GetValue()
}

func histogramCount(t *testing.T, collector prometheus.Collector) uint64 {
	t.Helper()
	ch := make(chan prometheus.Metric, 8)
	collector.Collect(ch)
	close(ch)
	for sample := range ch {
		metric := &dto.Metric{}
		if err := sample.Write(metric); err != nil {
			t.Fatalf("write histogram metric: %v", err)
		}
		if histogram := metric.GetHistogram(); histogram != nil {
			return histogram.GetSampleCount()
		}
	}
	t.Fatal("histogram metric not collected")
	return 0
}
