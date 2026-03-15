package observability

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Usage struct {
	PromptTokens     int64   `json:"prompt_tokens,omitempty"`
	CompletionTokens int64   `json:"completion_tokens,omitempty"`
	TotalTokens      int64   `json:"total_tokens,omitempty"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	Source           string  `json:"source,omitempty"`
}

type ExecutionRecord struct {
	Operation string
	Scene     string
	ToolName  string
	ToolMode  string
	RiskLevel string
	Status    string
	Duration  time.Duration
	Usage     *Usage
}

type ThoughtChainRecord struct {
	Scene    string
	Status   string
	Duration time.Duration
}

type ThoughtChainNodeRecord struct {
	Scene    string
	Kind     string
	Status   string
	Duration time.Duration
}

type ThoughtChainApprovalRecord struct {
	Scene    string
	Status   string
	Duration time.Duration
}

type Metrics struct {
	toolExecutions      *prometheus.CounterVec
	toolDuration        *prometheus.HistogramVec
	agentExecutions     *prometheus.CounterVec
	agentDuration       *prometheus.HistogramVec
	tokenUsage          *prometheus.CounterVec
	costUsage           *prometheus.CounterVec
	thoughtChains       *prometheus.CounterVec
	thoughtChainLatency *prometheus.HistogramVec
	thoughtChainNodes   *prometheus.CounterVec
	thoughtNodeLatency  *prometheus.HistogramVec
	thoughtApprovals    *prometheus.CounterVec
	approvalWaitLatency *prometheus.HistogramVec
	firstTokenLatency   *prometheus.HistogramVec
}

var (
	defaultMetrics     *Metrics
	defaultMetricsOnce sync.Once
)

func DefaultMetrics() *Metrics {
	defaultMetricsOnce.Do(func() {
		defaultMetrics = newMetrics()
	})
	return defaultMetrics
}

func ObserveToolExecution(record ExecutionRecord) {
	DefaultMetrics().ObserveToolExecution(record)
}

func ObserveAgentExecution(record ExecutionRecord) {
	DefaultMetrics().ObserveAgentExecution(record)
}

func ObserveThoughtChain(record ThoughtChainRecord) {
	DefaultMetrics().ObserveThoughtChain(record)
}

func ObserveThoughtChainNode(record ThoughtChainNodeRecord) {
	DefaultMetrics().ObserveThoughtChainNode(record)
}

func ObserveThoughtChainApproval(record ThoughtChainApprovalRecord) {
	DefaultMetrics().ObserveThoughtChainApproval(record)
}

func ObserveFirstToken(scene string, duration time.Duration) {
	DefaultMetrics().ObserveFirstToken(scene, duration)
}

func newMetrics() *Metrics {
	return &Metrics{
		toolExecutions: registerCounterVec(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "opspilot_ai_tool_executions_total",
			Help: "Total number of AI tool executions by tool and outcome.",
		}, []string{"tool", "mode", "risk", "scene", "status"})),
		toolDuration: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "opspilot_ai_tool_execution_duration_seconds",
			Help:    "Duration of AI tool executions in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"tool", "scene", "status"})),
		agentExecutions: registerCounterVec(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "opspilot_ai_agent_executions_total",
			Help: "Total number of AI agent run or resume operations by outcome.",
		}, []string{"operation", "scene", "status"})),
		agentDuration: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "opspilot_ai_agent_execution_duration_seconds",
			Help:    "Duration of AI agent run or resume operations in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation", "scene", "status"})),
		tokenUsage: registerCounterVec(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "opspilot_ai_tokens_total",
			Help: "Reported AI token usage by scope and source.",
		}, []string{"scope", "name", "scene", "token_type", "source"})),
		costUsage: registerCounterVec(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "opspilot_ai_cost_usd_total",
			Help: "Reported AI execution cost in USD by scope and source.",
		}, []string{"scope", "name", "scene", "source"})),
		thoughtChains: registerCounterVec(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "opspilot_ai_thoughtchain_runs_total",
			Help: "Total number of thoughtChain executions by scene and status.",
		}, []string{"scene", "status"})),
		thoughtChainLatency: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "opspilot_ai_thoughtchain_duration_seconds",
			Help:    "Duration of thoughtChain executions in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"scene", "status"})),
		thoughtChainNodes: registerCounterVec(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "opspilot_ai_thoughtchain_nodes_total",
			Help: "Total number of thoughtChain nodes closed by kind and status.",
		}, []string{"scene", "kind", "status"})),
		thoughtNodeLatency: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "opspilot_ai_thoughtchain_node_duration_seconds",
			Help:    "Duration of thoughtChain nodes in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"scene", "kind", "status"})),
		thoughtApprovals: registerCounterVec(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "opspilot_ai_thoughtchain_approvals_total",
			Help: "Total number of thoughtChain approvals by scene and outcome.",
		}, []string{"scene", "status"})),
		approvalWaitLatency: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "opspilot_ai_thoughtchain_approval_wait_seconds",
			Help:    "Wait time for thoughtChain approvals in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"scene", "status"})),
		firstTokenLatency: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "opspilot_ai_first_token_seconds",
			Help:    "Time to first token in seconds.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"scene"})),
	}
}

func (m *Metrics) ObserveToolExecution(record ExecutionRecord) {
	if m == nil {
		return
	}
	status := normalizeStatus(record.Status)
	scene := normalizeLabel(record.Scene)
	tool := normalizeLabel(record.ToolName)
	mode := normalizeLabel(record.ToolMode)
	risk := normalizeLabel(record.RiskLevel)

	m.toolExecutions.WithLabelValues(tool, mode, risk, scene, status).Inc()
	m.toolDuration.WithLabelValues(tool, scene, status).Observe(durationSeconds(record.Duration))
	m.observeUsage("tool", tool, scene, record.Usage)
}

func (m *Metrics) ObserveAgentExecution(record ExecutionRecord) {
	if m == nil {
		return
	}
	status := normalizeStatus(record.Status)
	scene := normalizeLabel(record.Scene)
	operation := normalizeLabel(record.Operation)

	m.agentExecutions.WithLabelValues(operation, scene, status).Inc()
	m.agentDuration.WithLabelValues(operation, scene, status).Observe(durationSeconds(record.Duration))
	m.observeUsage("agent", operation, scene, record.Usage)
}

func (m *Metrics) ObserveThoughtChain(record ThoughtChainRecord) {
	if m == nil {
		return
	}
	scene := normalizeLabel(record.Scene)
	status := normalizeStatus(record.Status)
	m.thoughtChains.WithLabelValues(scene, status).Inc()
	m.thoughtChainLatency.WithLabelValues(scene, status).Observe(durationSeconds(record.Duration))
}

func (m *Metrics) ObserveThoughtChainNode(record ThoughtChainNodeRecord) {
	if m == nil {
		return
	}
	scene := normalizeLabel(record.Scene)
	kind := normalizeLabel(record.Kind)
	status := normalizeStatus(record.Status)
	m.thoughtChainNodes.WithLabelValues(scene, kind, status).Inc()
	m.thoughtNodeLatency.WithLabelValues(scene, kind, status).Observe(durationSeconds(record.Duration))
}

func (m *Metrics) ObserveThoughtChainApproval(record ThoughtChainApprovalRecord) {
	if m == nil {
		return
	}
	scene := normalizeLabel(record.Scene)
	status := normalizeStatus(record.Status)
	m.thoughtApprovals.WithLabelValues(scene, status).Inc()
	m.approvalWaitLatency.WithLabelValues(scene, status).Observe(durationSeconds(record.Duration))
}

// ObserveFirstToken 记录首 token 延迟。
func (m *Metrics) ObserveFirstToken(scene string, duration time.Duration) {
	if m == nil || duration <= 0 {
		return
	}
	m.firstTokenLatency.WithLabelValues(normalizeLabel(scene)).Observe(duration.Seconds())
}

func (m *Metrics) observeUsage(scope, name, scene string, usage *Usage) {
	usage = normalizeUsage(usage)
	if usage == nil {
		return
	}
	source := normalizeLabel(usage.Source)
	if usage.PromptTokens > 0 {
		m.tokenUsage.WithLabelValues(scope, name, scene, "prompt", source).Add(float64(usage.PromptTokens))
	}
	if usage.CompletionTokens > 0 {
		m.tokenUsage.WithLabelValues(scope, name, scene, "completion", source).Add(float64(usage.CompletionTokens))
	}
	if usage.TotalTokens > 0 {
		m.tokenUsage.WithLabelValues(scope, name, scene, "total", source).Add(float64(usage.TotalTokens))
	}
	if usage.EstimatedCostUSD > 0 {
		m.costUsage.WithLabelValues(scope, name, scene, source).Add(usage.EstimatedCostUSD)
	}
}

func normalizeUsage(usage *Usage) *Usage {
	if usage == nil {
		return nil
	}
	cloned := *usage
	if cloned.TotalTokens <= 0 {
		cloned.TotalTokens = cloned.PromptTokens + cloned.CompletionTokens
	}
	if cloned.TotalTokens <= 0 && cloned.EstimatedCostUSD <= 0 {
		return nil
	}
	cloned.Source = normalizeLabel(cloned.Source)
	return &cloned
}

func durationSeconds(d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return d.Seconds()
}

func normalizeStatus(status string) string {
	return normalizeLabel(status)
}

func normalizeLabel(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func registerCounterVec(counter *prometheus.CounterVec) *prometheus.CounterVec {
	if err := prometheus.Register(counter); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return counter
}

func registerHistogramVec(histogram *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := prometheus.Register(histogram); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return histogram
}
