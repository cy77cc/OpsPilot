package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
)

// ArgNormalizeConfig controls tool argument normalization behavior.
type ArgNormalizeConfig struct {
	Enabled    bool
	ShadowMode bool
	Reporter   func(context.Context, NormalizationMetadata)
}

// NormalizeResult captures the normalized payload and the emitted metadata.
type NormalizeResult struct {
	NormalizedJSON string
	Metadata       NormalizationMetadata
}

// NormalizationMetadata describes how arguments were normalized.
type NormalizationMetadata struct {
	ToolName         string            `json:"tool_name"`
	Enabled          bool              `json:"enabled"`
	ShadowMode       bool              `json:"shadow_mode"`
	NormalizedKeys   []string          `json:"normalized_keys,omitempty"`
	Coercions        []string          `json:"coercions,omitempty"`
	CoercionFailures []CoercionFailure `json:"coercion_failures,omitempty"`
}

var hostExecBareIPv4Pattern = regexp.MustCompile(`("host_id"\s*:\s*)(\d{1,3}(?:\.\d{1,3}){3})(\s*[,}])`)

// CoercionFailure records a normalization failure with the original provided value.
type CoercionFailure struct {
	Field    string `json:"field"`
	Provided any    `json:"provided"`
	Expected string `json:"expected"`
	Reason   string `json:"reason,omitempty"`
}

type normalizationContextKey struct{}

// WithNormalizationMetadata stores normalization metadata on the context.
func WithNormalizationMetadata(ctx context.Context, meta NormalizationMetadata) context.Context {
	return context.WithValue(ctx, normalizationContextKey{}, meta)
}

// NormalizationMetadataFromContext extracts normalization metadata from the context.
func NormalizationMetadataFromContext(ctx context.Context) (NormalizationMetadata, bool) {
	meta, ok := ctx.Value(normalizationContextKey{}).(NormalizationMetadata)
	return meta, ok
}

type argNormalizationMiddleware struct {
	config  *ArgNormalizeConfig
	schemas map[string]*schema.ParamsOneOf
}

// NewArgNormalizationToolMiddleware creates a compose.ToolMiddleware that normalizes tool args.
func NewArgNormalizationToolMiddleware(ctx context.Context, tools []tool.BaseTool, cfg *ArgNormalizeConfig) (compose.ToolMiddleware, error) {
	if cfg == nil {
		cfg = &ArgNormalizeConfig{
			Enabled:    false,
			ShadowMode: true,
		}
	}

	registry := make(map[string]*schema.ParamsOneOf, len(tools))
	for _, item := range tools {
		if item == nil {
			continue
		}
		info, err := item.Info(ctx)
		if err != nil {
			return compose.ToolMiddleware{}, fmt.Errorf("arg normalizer: load tool info: %w", err)
		}
		if info == nil || info.Name == "" || info.ParamsOneOf == nil {
			continue
		}
		registry[info.Name] = info.ParamsOneOf
	}

	mw := &argNormalizationMiddleware{
		config:  cfg,
		schemas: registry,
	}
	return compose.ToolMiddleware{
		Invokable:  mw.wrapInvokable,
		Streamable: mw.wrapStreamable,
	}, nil
}

// NormalizeToolArgs normalizes raw JSON tool arguments against a tool schema.
func NormalizeToolArgs(toolName, raw string, params *schema.ParamsOneOf) (NormalizeResult, error) {
	result := NormalizeResult{
		NormalizedJSON: raw,
		Metadata: NormalizationMetadata{
			ToolName: toolName,
		},
	}
	if params == nil {
		return result, nil
	}

	js, err := params.ToJSONSchema()
	if err != nil {
		return result, err
	}

	normalized, meta := normalizeRawArgs(toolName, raw, js)
	result.NormalizedJSON = normalized
	result.Metadata = meta
	return result, nil
}

func (m *argNormalizationMiddleware) wrapInvokable(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
	return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
		if input == nil {
			return next(ctx, input)
		}
		normalizedCtx, normalizedArgs, err := m.normalizeCall(ctx, input.Name, input.Arguments)
		if err != nil {
			return nil, err
		}
		if m.config.Enabled {
			input = cloneToolInput(input)
			input.Arguments = normalizedArgs
		}
		return next(normalizedCtx, input)
	}
}

func (m *argNormalizationMiddleware) wrapStreamable(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
	return func(ctx context.Context, input *compose.ToolInput) (*compose.StreamToolOutput, error) {
		if input == nil {
			return next(ctx, input)
		}
		normalizedCtx, normalizedArgs, err := m.normalizeCall(ctx, input.Name, input.Arguments)
		if err != nil {
			return nil, err
		}
		if m.config.Enabled {
			input = cloneToolInput(input)
			input.Arguments = normalizedArgs
		}
		return next(normalizedCtx, input)
	}
}

func (m *argNormalizationMiddleware) normalizeCall(ctx context.Context, toolName, raw string) (context.Context, string, error) {
	params := m.schemas[toolName]
	if params == nil {
		return ctx, raw, nil
	}
	if !m.config.Enabled && !m.config.ShadowMode && m.config.Reporter == nil {
		return ctx, raw, nil
	}

	result, err := NormalizeToolArgs(toolName, raw, params)
	if err != nil {
		return ctx, raw, err
	}
	result.Metadata.Enabled = m.config.Enabled
	result.Metadata.ShadowMode = m.config.ShadowMode

	ctx = WithNormalizationMetadata(ctx, result.Metadata)
	if m.config.Reporter != nil {
		m.config.Reporter(ctx, result.Metadata)
	}
	if m.config.Enabled {
		return ctx, result.NormalizedJSON, nil
	}
	if m.config.ShadowMode {
		return ctx, raw, nil
	}
	return ctx, raw, nil
}

func cloneToolInput(input *compose.ToolInput) *compose.ToolInput {
	if input == nil {
		return nil
	}
	cloned := *input
	if input.CallOptions != nil {
		cloned.CallOptions = append([]tool.Option(nil), input.CallOptions...)
	}
	return &cloned
}

func normalizeRawArgs(toolName, raw string, js *jsonschema.Schema) (string, NormalizationMetadata) {
	meta := NormalizationMetadata{
		ToolName: toolName,
	}
	if js == nil {
		return raw, meta
	}

	decodeRaw := preprocessRawArgs(toolName, raw, &meta)
	decoded, err := decodeJSONWithNumber(decodeRaw)
	if err != nil {
		meta.CoercionFailures = append(meta.CoercionFailures, CoercionFailure{
			Field:    "$root",
			Provided: raw,
			Expected: "json object",
			Reason:   err.Error(),
		})
		return raw, meta
	}

	root, ok := decoded.(map[string]any)
	if !ok {
		meta.CoercionFailures = append(meta.CoercionFailures, CoercionFailure{
			Field:    "$root",
			Provided: decoded,
			Expected: "json object",
			Reason:   "tool arguments must be a JSON object",
		})
		return raw, meta
	}

	if toolName == "host_exec" {
		coerceHostExecIPHostID(root, &meta)
	}

	normalized := normalizeObject("", root, js, &meta, fieldRequirements(js))
	encoded, err := json.Marshal(normalized)
	if err != nil {
		meta.CoercionFailures = append(meta.CoercionFailures, CoercionFailure{
			Field:    "$root",
			Provided: root,
			Expected: "json-serializable normalized object",
			Reason:   err.Error(),
		})
		return raw, meta
	}

	return string(encoded), meta
}

func preprocessRawArgs(toolName, raw string, meta *NormalizationMetadata) string {
	if strings.TrimSpace(toolName) != "host_exec" {
		return raw
	}
	rewritten, changed := quoteBareIPv4HostID(raw)
	if changed {
		appendCoercion(meta, "host_id", "bare-ipv4->quoted-string")
	}
	return rewritten
}

func quoteBareIPv4HostID(raw string) (string, bool) {
	matches := hostExecBareIPv4Pattern.FindAllStringSubmatchIndex(raw, -1)
	if len(matches) == 0 {
		return raw, false
	}

	var (
		builder strings.Builder
		last    int
		changed bool
	)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}
		fullStart, fullEnd := match[0], match[1]
		prefixStart, prefixEnd := match[2], match[3]
		ipStart, ipEnd := match[4], match[5]
		suffixStart, suffixEnd := match[6], match[7]
		ip := raw[ipStart:ipEnd]
		if !isStrictIPv4(ip) {
			continue
		}

		builder.WriteString(raw[last:fullStart])
		builder.WriteString(raw[prefixStart:prefixEnd])
		builder.WriteString(`"`)
		builder.WriteString(ip)
		builder.WriteString(`"`)
		builder.WriteString(raw[suffixStart:suffixEnd])
		last = fullEnd
		changed = true
	}
	if !changed {
		return raw, false
	}
	builder.WriteString(raw[last:])
	return builder.String(), true
}

func isStrictIPv4(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.Count(raw, ".") != 3 {
		return false
	}
	ip := net.ParseIP(raw)
	return ip != nil && ip.To4() != nil
}

func coerceHostExecIPHostID(root map[string]any, meta *NormalizationMetadata) {
	if root == nil {
		return
	}
	rawHostID, ok := root["host_id"]
	if !ok {
		return
	}
	hostID, ok := rawHostID.(string)
	if !ok {
		return
	}
	hostID = strings.TrimSpace(hostID)
	if !isStrictIPv4(hostID) {
		return
	}
	if existingTarget, ok := root["target"].(string); !ok || strings.TrimSpace(existingTarget) == "" {
		root["target"] = hostID
		appendCoercion(meta, "target", "derived-from-host_id")
	}
	delete(root, "host_id")
	appendCoercion(meta, "host_id", "ipv4->target")
}

func decodeJSONWithNumber(raw string) (any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty arguments")
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()

	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected trailing JSON content")
		}
		return nil, err
	}
	return out, nil
}

func normalizeObject(path string, obj map[string]any, js *jsonschema.Schema, meta *NormalizationMetadata, required map[string]bool) map[string]any {
	if js == nil {
		return obj
	}

	props := orderedProperties(js)
	if len(props) == 0 {
		return normalizeAdditionalProperties(path, obj, js, required, meta)
	}

	index := buildPropertyIndex(props)
	grouped := make(map[string][]keyCandidate, len(obj))
	out := make(map[string]any, len(obj))

	rawKeys := make([]string, 0, len(obj))
	for rawKey := range obj {
		rawKeys = append(rawKeys, rawKey)
	}
	sort.Strings(rawKeys)

	for _, rawKey := range rawKeys {
		rawValue := obj[rawKey]
		canonical, rank, matched := index.resolve(rawKey)
		if !matched {
			if js.AdditionalProperties != nil && js.AdditionalProperties != jsonschema.FalseSchema {
				normalizedValue, omitted := normalizeValue(joinPath(path, rawKey), rawValue, js.AdditionalProperties, false, meta)
				if !omitted {
					out[rawKey] = normalizedValue
				}
			} else {
				out[rawKey] = rawValue
			}
			continue
		}

		if rawKey != canonical {
			appendNormalizedKey(meta, rawKey, canonical)
		}
		grouped[canonical] = append(grouped[canonical], keyCandidate{
			rawKey:    rawKey,
			canonical: canonical,
			rank:      rank,
			value:     rawValue,
		})
	}

	canonicalKeys := make([]string, 0, len(grouped))
	for canonical := range grouped {
		canonicalKeys = append(canonicalKeys, canonical)
	}
	sort.Strings(canonicalKeys)

	for _, canonical := range canonicalKeys {
		candidates := grouped[canonical]
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].rank != candidates[j].rank {
				return candidates[i].rank > candidates[j].rank
			}
			return candidates[i].rawKey < candidates[j].rawKey
		})

		selected := candidates[0]
		propSchema := propSchemaForKey(js, canonical)
		normalizedValue, omitted := normalizeValue(joinPath(path, canonical), selected.value, propSchema, required[canonical], meta)
		if !omitted {
			out[canonical] = normalizedValue
		}
	}

	return out
}

type keyCandidate struct {
	rawKey    string
	canonical string
	rank      int
	value     any
}

type propertyIndex struct {
	exact      map[string]string
	ignoreCase map[string][]string
	normalized map[string][]string
}

func buildPropertyIndex(props []string) *propertyIndex {
	idx := &propertyIndex{
		exact:      make(map[string]string, len(props)),
		ignoreCase: make(map[string][]string, len(props)),
		normalized: make(map[string][]string, len(props)),
	}
	for _, prop := range props {
		idx.exact[prop] = prop
		idx.ignoreCase[strings.ToLower(prop)] = append(idx.ignoreCase[strings.ToLower(prop)], prop)
		idx.normalized[normalizeKey(prop)] = append(idx.normalized[normalizeKey(prop)], prop)
	}
	for _, keys := range idx.ignoreCase {
		sort.Strings(keys)
	}
	for _, keys := range idx.normalized {
		sort.Strings(keys)
	}
	return idx
}

func (idx *propertyIndex) resolve(rawKey string) (canonical string, rank int, matched bool) {
	if canonical, ok := idx.exact[rawKey]; ok {
		return canonical, 3, true
	}

	lower := strings.ToLower(rawKey)
	if candidates := idx.ignoreCase[lower]; len(candidates) > 0 {
		return candidates[0], 2, true
	}

	norm := normalizeKey(rawKey)
	if candidates := idx.normalized[norm]; len(candidates) > 0 {
		return candidates[0], 1, true
	}

	return "", 0, false
}

func orderedProperties(js *jsonschema.Schema) []string {
	if js == nil || js.Properties == nil {
		return nil
	}
	keys := make([]string, 0, js.Properties.Len())
	for pair := js.Properties.Oldest(); pair != nil; pair = pair.Next() {
		keys = append(keys, pair.Key)
	}
	sort.Strings(keys)
	return keys
}

func propSchemaForKey(js *jsonschema.Schema, key string) *jsonschema.Schema {
	if js == nil || js.Properties == nil {
		return nil
	}
	if v, ok := js.Properties.Get(key); ok {
		return v
	}
	return nil
}

func normalizeAdditionalProperties(path string, obj map[string]any, js *jsonschema.Schema, required map[string]bool, meta *NormalizationMetadata) map[string]any {
	if js == nil {
		return obj
	}
	if js.AdditionalProperties == nil || js.AdditionalProperties == jsonschema.FalseSchema {
		return obj
	}
	out := make(map[string]any, len(obj))
	for k, v := range obj {
		normalized, omitted := normalizeValue(joinPath(path, k), v, js.AdditionalProperties, required[k], meta)
		if !omitted {
			out[k] = normalized
		}
	}
	return out
}

func normalizeValue(path string, raw any, js *jsonschema.Schema, required bool, meta *NormalizationMetadata) (any, bool) {
	if js == nil {
		return raw, false
	}

	switch schemaType(js) {
	case "object":
		obj, ok := raw.(map[string]any)
		if !ok {
			return raw, false
		}
		return normalizeObject(path, obj, js, meta, fieldRequirements(js)), false
	case "array":
		arr, ok := raw.([]any)
		if !ok {
			return raw, false
		}
		itemSchema := js.Items
		out := make([]any, 0, len(arr))
		for idx, item := range arr {
			normalized, omitted := normalizeValue(joinPath(path, strconv.Itoa(idx)), item, itemSchema, true, meta)
			if !omitted {
				out = append(out, normalized)
			}
		}
		return out, false
	case "string":
		return normalizeStringValue(path, raw, js, required, meta)
	case "integer":
		return normalizeIntegerValue(path, raw, required, meta)
	case "number":
		return normalizeNumberValue(path, raw, required, meta)
	case "boolean":
		return normalizeBoolValue(path, raw, required, meta)
	case "null":
		return nil, false
	default:
		return raw, false
	}
}

func normalizeStringValue(path string, raw any, js *jsonschema.Schema, required bool, meta *NormalizationMetadata) (any, bool) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" && !required {
			appendCoercion(meta, path, "empty-string->omitted")
			return nil, true
		}
		if normalized, ok := matchEnumInsensitive(js.Enum, v); ok {
			if normalized != v {
				appendCoercion(meta, path, fmt.Sprintf("string(enum):%s->%s", v, normalized))
			}
			return normalized, false
		}
		if len(js.Enum) > 0 {
			appendFailure(meta, path, v, "string enum", "enum value is not a unique match")
		}
		return v, false
	case json.Number:
		if !required && strings.TrimSpace(v.String()) == "" {
			appendCoercion(meta, path, "empty-string->omitted")
			return nil, true
		}
		appendCoercion(meta, path, "number->string")
		return v.String(), false
	case bool:
		appendCoercion(meta, path, "bool->string")
		return strconv.FormatBool(v), false
	case float64:
		appendCoercion(meta, path, "number->string")
		return strconv.FormatFloat(v, 'f', -1, 64), false
	case int, int8, int16, int32, int64:
		appendCoercion(meta, path, "number->string")
		return fmt.Sprint(v), false
	default:
		return raw, false
	}
}

func normalizeIntegerValue(path string, raw any, required bool, meta *NormalizationMetadata) (any, bool) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" && !required {
			appendCoercion(meta, path, "empty-string->omitted")
			return nil, true
		}
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			appendFailure(meta, path, v, "int", err.Error())
			return raw, false
		}
		appendCoercion(meta, path, "string->int")
		return json.Number(strconv.FormatInt(n, 10)), false
	case json.Number:
		if _, err := v.Int64(); err == nil {
			return v, false
		}
		appendFailure(meta, path, v, "int", "invalid integer number")
		return raw, false
	case float64:
		if float64(int64(v)) == v {
			appendCoercion(meta, path, "number->int")
			return json.Number(strconv.FormatInt(int64(v), 10)), false
		}
		appendFailure(meta, path, v, "int", "non-integral number")
		return raw, false
	default:
		return raw, false
	}
}

func normalizeNumberValue(path string, raw any, required bool, meta *NormalizationMetadata) (any, bool) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" && !required {
			appendCoercion(meta, path, "empty-string->omitted")
			return nil, true
		}
		if _, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil {
			appendFailure(meta, path, v, "number", err.Error())
			return raw, false
		}
		appendCoercion(meta, path, "string->float")
		return json.Number(strings.TrimSpace(v)), false
	case json.Number:
		if _, err := v.Float64(); err == nil {
			return v, false
		}
		appendFailure(meta, path, v, "number", "invalid number")
		return raw, false
	case float64:
		return json.Number(strconv.FormatFloat(v, 'f', -1, 64)), false
	case int, int8, int16, int32, int64:
		appendCoercion(meta, path, "number->number")
		return json.Number(fmt.Sprint(v)), false
	default:
		return raw, false
	}
}

func normalizeBoolValue(path string, raw any, required bool, meta *NormalizationMetadata) (any, bool) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" && !required {
			appendCoercion(meta, path, "empty-string->omitted")
			return nil, true
		}
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			appendFailure(meta, path, v, "bool", err.Error())
			return raw, false
		}
		appendCoercion(meta, path, "string->bool")
		return b, false
	case bool:
		return v, false
	default:
		return raw, false
	}
}

func matchEnumInsensitive(values []any, raw string) (string, bool) {
	if len(values) == 0 {
		return "", false
	}

	exact := make([]string, 0, len(values))
	insensitive := make([]string, 0, len(values))
	for _, item := range values {
		s, ok := item.(string)
		if !ok {
			continue
		}
		if s == raw {
			exact = append(exact, s)
			continue
		}
		if strings.EqualFold(s, raw) {
			insensitive = append(insensitive, s)
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) > 1 {
		return "", false
	}
	if len(insensitive) == 1 {
		return insensitive[0], true
	}
	return "", false
}

func appendCoercion(meta *NormalizationMetadata, path, rule string) {
	if meta == nil || rule == "" {
		return
	}
	meta.Coercions = append(meta.Coercions, fmt.Sprintf("%s:%s", path, rule))
}

func appendFailure(meta *NormalizationMetadata, path string, provided any, expected, reason string) {
	if meta == nil {
		return
	}
	meta.CoercionFailures = append(meta.CoercionFailures, CoercionFailure{
		Field:    path,
		Provided: provided,
		Expected: expected,
		Reason:   reason,
	})
}

func appendNormalizedKey(meta *NormalizationMetadata, rawKey, canonical string) {
	if meta == nil || rawKey == "" || canonical == "" || rawKey == canonical {
		return
	}
	label := fmt.Sprintf("%s->%s", rawKey, canonical)
	for _, existing := range meta.NormalizedKeys {
		if existing == label {
			return
		}
	}
	meta.NormalizedKeys = append(meta.NormalizedKeys, label)
}

func schemaType(js *jsonschema.Schema) string {
	if js == nil {
		return ""
	}
	if js.Type != "" {
		return strings.ToLower(js.Type)
	}
	if len(js.TypeEnhanced) > 0 {
		return strings.ToLower(js.TypeEnhanced[0])
	}
	return ""
}

func fieldRequirements(js *jsonschema.Schema) map[string]bool {
	req := make(map[string]bool, len(js.Required))
	for _, name := range js.Required {
		req[name] = true
	}
	return req
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	if child == "" {
		return parent
	}
	return parent + "." + child
}

func normalizeKey(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return ""
	}

	var out []rune
	var prevUnderscore bool
	var prevLowerOrDigit bool
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if unicode.IsUpper(r) && prevLowerOrDigit && !prevUnderscore {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			prevUnderscore = false
			prevLowerOrDigit = unicode.IsLower(r) || unicode.IsDigit(r)
		default:
			if len(out) > 0 && !prevUnderscore {
				out = append(out, '_')
				prevUnderscore = true
			}
			prevLowerOrDigit = false
		}
	}
	normalized := strings.Trim(string(out), "_")
	normalized = strings.ReplaceAll(normalized, "__", "_")
	for strings.Contains(normalized, "__") {
		normalized = strings.ReplaceAll(normalized, "__", "_")
	}
	return normalized
}
