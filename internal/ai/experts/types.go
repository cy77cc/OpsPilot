package experts

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type DomainWeight struct {
	Name   string  `yaml:"name"`
	Weight float64 `yaml:"weight"`
}

type Expert struct {
	Name         string
	DisplayName  string
	Persona      string
	ToolPatterns []string
	Domains      []DomainWeight
	Keywords     []string
	Capabilities []string
	RiskLevel    string

	Agent *react.Agent
	Tools map[string]tool.InvokableTool
}

type ExpertConfig struct {
	Name         string         `yaml:"name"`
	DisplayName  string         `yaml:"display_name"`
	Persona      string         `yaml:"persona"`
	ToolPatterns []string       `yaml:"tool_patterns"`
	Domains      []DomainWeight `yaml:"domains"`
	Keywords     []string       `yaml:"keywords"`
	Capabilities []string       `yaml:"capabilities"`
	RiskLevel    string         `yaml:"risk_level"`
}

type ExpertsFile struct {
	Version string         `yaml:"version"`
	Experts []ExpertConfig `yaml:"experts"`
}

type RankedExpert struct {
	Expert *Expert
	Score  float64
}

type ExpertRegistry interface {
	GetExpert(name string) (*Expert, bool)
	ListExperts() []*Expert
	Reload() error
	MatchByKeywords(content string) []*RankedExpert
	MatchByDomain(domain string) []*RankedExpert
}

type RouteRequest struct {
	Message        string
	Scene          string
	History        []*schema.Message
	RuntimeContext map[string]any
}

type RouteDecision struct {
	PrimaryExpert string
	HelperExperts []string
	Strategy      ExecutionStrategy
	Confidence    float64
	Source        string
}

type ExecutionStrategy string

const (
	StrategySingle     ExecutionStrategy = "single"
	StrategySequential ExecutionStrategy = "sequential"
	StrategyParallel   ExecutionStrategy = "parallel"
)

type SceneMapping struct {
	PrimaryExpert string            `yaml:"primary_expert"`
	HelperExperts []string          `yaml:"helper_experts"`
	Strategy      ExecutionStrategy `yaml:"strategy"`
	ContextHints  []string          `yaml:"context_hints"`
	Description   string            `yaml:"description"`
	Keywords      []string          `yaml:"keywords"`
	Tools         []string          `yaml:"tools"`
}

type SceneMappingsFile struct {
	Version  string                  `yaml:"version"`
	Mappings map[string]SceneMapping `yaml:"mappings"`
}

type ExecuteRequest struct {
	Message        string
	Decision       *RouteDecision
	RuntimeContext map[string]any
	History        []*schema.Message
}

type ExpertTrace struct {
	ExpertName string
	Input      string
	Output     string
	Duration   time.Duration
	Status     string
}

type ExecuteResult struct {
	Response string
	Traces   []ExpertTrace
	Metadata map[string]any
}

type ExecutionPlan struct {
	Steps []ExecutionStep
}

type ExecutionStep struct {
	ExpertName  string
	Task        string
	DependsOn   []int
	ContextFrom []int
}

type ExpertResult struct {
	ExpertName string
	Output     string
	Error      error
	Duration   time.Duration
}

type AggregationMode string

const (
	AggregationTemplate AggregationMode = "template"
	AggregationLLM      AggregationMode = "llm"
)


type AggregatorLLM interface {
	Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error)
	Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (outStream *schema.StreamReader[*schema.Message], err error)
}
