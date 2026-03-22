package middleware

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/eino-contrib/jsonschema"
	"github.com/wk8/go-ordered-map/v2"
)

type fakeBaseTool struct {
	info *schema.ToolInfo
}

func (f *fakeBaseTool) Info(context.Context) (*schema.ToolInfo, error) {
	return f.info, nil
}

type argNormalizeApprovalEvaluator struct {
	args string
}

func (c *argNormalizeApprovalEvaluator) Evaluate(_ context.Context, _ string, args string, _ common.ApprovalEvalMeta) (*common.ApprovalDecision, error) {
	c.args = args
	return &common.ApprovalDecision{RequiresApproval: false}, nil
}

func mustToolInfo(name string, params map[string]*schema.ParameterInfo) *schema.ToolInfo {
	return &schema.ToolInfo{
		Name:        name,
		Desc:        "test tool",
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}
}

func mustNormalizedJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var got map[string]any
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decode normalized json: %v", err)
	}
	return got
}

func requireStringSliceContains(t *testing.T, got []string, want string) {
	t.Helper()
	for _, item := range got {
		if item == want {
			return
		}
	}
	t.Fatalf("expected %q in %v", want, got)
}

func TestNormalizeArgs_FieldPriority(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("priority_tool", map[string]*schema.ParameterInfo{
			"user_name": {Type: schema.String},
		}),
	}

	mw, err := NewArgNormalizationToolMiddleware(context.Background(), []tool.BaseTool{schemaTool}, &ArgNormalizeConfig{Enabled: true})
	if err != nil {
		t.Fatalf("build middleware: %v", err)
	}

	var captured string
	var capturedMeta NormalizationMetadata
	endpoint := func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
		captured = input.Arguments
		if meta, ok := NormalizationMetadataFromContext(ctx); ok {
			capturedMeta = meta
		}
		return &compose.ToolOutput{Result: "ok"}, nil
	}

	wrapped := mw.Invokable(endpoint)
	_, err = wrapped(context.Background(), &compose.ToolInput{
		Name:      "priority_tool",
		Arguments: `{"UserName":"A","user_name":"B"}`,
		CallID:    "call-1",
	})
	if err != nil {
		t.Fatalf("invoke middleware: %v", err)
	}

	got := mustNormalizedJSON(t, captured)
	if got["user_name"] != "B" {
		t.Fatalf("expected exact field to win, got %v", got["user_name"])
	}
	if capturedMeta.ToolName != "priority_tool" {
		t.Fatalf("expected metadata tool name priority_tool, got %q", capturedMeta.ToolName)
	}
	requireStringSliceContains(t, capturedMeta.NormalizedKeys, "UserName->user_name")
	if len(capturedMeta.CoercionFailures) != 0 {
		t.Fatalf("expected no coercion failures, got %v", capturedMeta.CoercionFailures)
	}
}

func TestNormalizeArgs_TypeCoercion(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("coercion_tool", map[string]*schema.ParameterInfo{
			"count":   {Type: schema.Integer, Required: true},
			"ratio":   {Type: schema.Number},
			"enabled": {Type: schema.Boolean},
			"label":   {Type: schema.String},
		}),
	}

	result, err := NormalizeToolArgs("coercion_tool", `{"count":"2","ratio":"1.5","enabled":"true","label":2}`, schemaTool.info.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	got := mustNormalizedJSON(t, result.NormalizedJSON)
	if got["count"] != json.Number("2") {
		t.Fatalf("expected count to coerce to number 2, got %T %v", got["count"], got["count"])
	}
	if got["ratio"] != json.Number("1.5") {
		t.Fatalf("expected ratio to coerce to number 1.5, got %T %v", got["ratio"], got["ratio"])
	}
	if got["enabled"] != true {
		t.Fatalf("expected enabled to coerce to true, got %T %v", got["enabled"], got["enabled"])
	}
	if got["label"] != "2" {
		t.Fatalf("expected label to coerce to string 2, got %T %v", got["label"], got["label"])
	}
	requireStringSliceContains(t, result.Metadata.Coercions, "count:string->int")
	requireStringSliceContains(t, result.Metadata.Coercions, "enabled:string->bool")
	requireStringSliceContains(t, result.Metadata.Coercions, "label:number->string")
}

func TestNormalizeArgs_EnumCaseInsensitive(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("enum_tool", map[string]*schema.ParameterInfo{
			"state": {Type: schema.String, Enum: []string{"open", "closed"}},
		}),
	}

	result, err := NormalizeToolArgs("enum_tool", `{"state":"OPEN"}`, schemaTool.info.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	got := mustNormalizedJSON(t, result.NormalizedJSON)
	if got["state"] != "open" {
		t.Fatalf("expected enum to normalize to canonical case, got %v", got["state"])
	}
	requireStringSliceContains(t, result.Metadata.Coercions, "state:string(enum):OPEN->open")
}

func TestNormalizeArgs_EmptyStringOptionalNumber(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("optional_tool", map[string]*schema.ParameterInfo{
			"limit": {Type: schema.Integer},
		}),
	}

	result, err := NormalizeToolArgs("optional_tool", `{"limit":""}`, schemaTool.info.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	got := mustNormalizedJSON(t, result.NormalizedJSON)
	if _, ok := got["limit"]; ok {
		t.Fatalf("expected optional empty scalar to be omitted, got %v", got["limit"])
	}
	if len(result.Metadata.CoercionFailures) != 0 {
		t.Fatalf("expected no coercion failures, got %v", result.Metadata.CoercionFailures)
	}
}

func TestNormalizeArgs_CoercionFailureMetadataUsesOriginalValue(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("failure_tool", map[string]*schema.ParameterInfo{
			"count": {Type: schema.Integer, Required: true},
		}),
	}

	result, err := NormalizeToolArgs("failure_tool", `{"count":"nope"}`, schemaTool.info.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	if len(result.Metadata.CoercionFailures) != 1 {
		t.Fatalf("expected 1 coercion failure, got %v", result.Metadata.CoercionFailures)
	}
	failure := result.Metadata.CoercionFailures[0]
	if failure.Field != "count" {
		t.Fatalf("expected failure field count, got %q", failure.Field)
	}
	if failure.Expected != "int" {
		t.Fatalf("expected failure expected int, got %q", failure.Expected)
	}
	if got, ok := failure.Provided.(string); !ok || got != "nope" {
		t.Fatalf("expected original provided value nope, got %#v", failure.Provided)
	}
}

func TestArgNormalizationMiddleware_OrderBeforeApproval(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("ordered_tool", map[string]*schema.ParameterInfo{
			"user_name": {Type: schema.String},
		}),
	}

	normalizer, err := NewArgNormalizationToolMiddleware(context.Background(), []tool.BaseTool{schemaTool}, &ArgNormalizeConfig{Enabled: true})
	if err != nil {
		t.Fatalf("build normalizer: %v", err)
	}

	approvalCapture := &argNormalizeApprovalEvaluator{}
	approval := NewApprovalToolMiddleware(&ApprovalMiddlewareConfig{
		Orchestrator:     approvalCapture,
		NeedsApproval:    func(string) bool { return true },
		PreviewGenerator: DefaultPreviewGenerator,
		DefaultTimeout:   common.DefaultApprovalTimeout,
		ToolConfigs:      DefaultToolConfigs(),
	})

	var endpointArgs string
	endpoint := func(_ context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
		endpointArgs = input.Arguments
		return &compose.ToolOutput{Result: "ok"}, nil
	}

	wrapped := normalizer.Invokable(approval.Invokable(endpoint))
	_, err = wrapped(context.Background(), &compose.ToolInput{
		Name:      "ordered_tool",
		Arguments: `{"UserName":"A","user_name":"B"}`,
		CallID:    "call-2",
	})
	if err != nil {
		t.Fatalf("invoke middleware chain: %v", err)
	}

	if got := approvalCapture.args; got != `{"user_name":"B"}` {
		t.Fatalf("expected approval to see normalized args, got %s", got)
	}
	if endpointArgs != `{"user_name":"B"}` {
		t.Fatalf("expected endpoint to see normalized args, got %s", endpointArgs)
	}
}

func TestNormalizeArgs_KeyOrderDeterministic(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("deterministic_tool", map[string]*schema.ParameterInfo{
			"a_field": {Type: schema.String},
			"b_field": {Type: schema.String},
		}),
	}

	result, err := NormalizeToolArgs("deterministic_tool", `{"bField":"2","aField":"1"}`, schemaTool.info.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	if !strings.Contains(result.NormalizedJSON, "\"a_field\":\"1\"") || !strings.Contains(result.NormalizedJSON, "\"b_field\":\"2\"") {
		t.Fatalf("expected deterministic normalized output, got %s", result.NormalizedJSON)
	}
}

func TestNormalizeArgs_UnsupportedSchemaDoesNotFail(t *testing.T) {
	result, err := NormalizeToolArgs("unknown_tool", `{"value":"x"}`, nil)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}
	if result.NormalizedJSON != `{"value":"x"}` {
		t.Fatalf("expected raw args to remain unchanged, got %s", result.NormalizedJSON)
	}
}

func TestNormalizeArgs_InvalidJSONRecordsFailure(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("broken_tool", map[string]*schema.ParameterInfo{
			"value": {Type: schema.String},
		}),
	}

	result, err := NormalizeToolArgs("broken_tool", `{broken`, schemaTool.info.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}
	if len(result.Metadata.CoercionFailures) == 0 {
		t.Fatal("expected invalid JSON to be recorded as a failure")
	}
}

func TestNormalizeArgs_StableSchemaOrdering(t *testing.T) {
	schemaTool := &fakeBaseTool{
		info: mustToolInfo("order_tool", map[string]*schema.ParameterInfo{
			"user_name":  {Type: schema.String},
			"cluster_id": {Type: schema.Integer},
		}),
	}

	result, err := NormalizeToolArgs("order_tool", `{"clusterId":"1","userName":"alice"}`, schemaTool.info.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	if !strings.Contains(result.NormalizedJSON, "\"cluster_id\":1") {
		t.Fatalf("expected cluster_id to be normalized, got %s", result.NormalizedJSON)
	}
	if !strings.Contains(result.NormalizedJSON, "\"user_name\":\"alice\"") {
		t.Fatalf("expected user_name to be normalized, got %s", result.NormalizedJSON)
	}
}

func TestNormalizeArgs_EnumAmbiguityKeepsOriginal(t *testing.T) {
	toolInfo := &schema.ToolInfo{
		Name: "enum_ambiguity",
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&jsonschema.Schema{
			Type: "object",
			Properties: orderedmap.New[string, *jsonschema.Schema](
				orderedmap.WithInitialData[string, *jsonschema.Schema](
					orderedmap.Pair[string, *jsonschema.Schema]{
						Key: "state",
						Value: &jsonschema.Schema{
							Type: "string",
							Enum: []any{"OPEN", "open"},
						},
					},
				),
			),
		}),
	}

	result, err := NormalizeToolArgs("enum_ambiguity", `{"state":"Open"}`, toolInfo.ParamsOneOf)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}
	got := mustNormalizedJSON(t, result.NormalizedJSON)
	if got["state"] != "Open" {
		t.Fatalf("expected ambiguous enum to remain original, got %v", got["state"])
	}
	if len(result.Metadata.CoercionFailures) == 0 {
		t.Fatal("expected ambiguous enum to record a failure")
	}
}
