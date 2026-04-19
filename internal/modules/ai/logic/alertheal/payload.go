package alertheal

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

const (
	protocolAlertmanager = "alertmanager"
	protocolUnified      = "opspilot.alert.v1"
)

// NormalizedAlert 是跨协议统一后的告警载荷。
type NormalizedAlert struct {
	Source          string
	Protocol        string
	Fingerprint     string
	Status          string
	Severity        string
	Title           string
	Target          string
	LabelsJSON      string
	AnnotationsJSON string
	RawPayloadJSON  string
	StartsAt        *time.Time
	EndsAt          *time.Time
}

// DedupeKey 生成 source/fingerprint/status 维度去重键。
func DedupeKey(source, fingerprint, status string) string {
	return strings.TrimSpace(source) + ":" + strings.TrimSpace(fingerprint) + ":" + strings.TrimSpace(status)
}

// NormalizePayload 将不同协议的原始 payload 归一化为统一结构。
func NormalizePayload(protocol string, raw []byte) ([]NormalizedAlert, error) {
	compactRaw, err := compactJSON(raw)
	if err != nil {
		return nil, ErrInvalidPayload
	}

	var envelope struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, ErrInvalidPayload
	}

	effectiveProtocol := normalizeProtocol(protocol)
	if effectiveProtocol != protocolAlertmanager && effectiveProtocol != protocolUnified {
		if normalizeProtocol(envelope.Kind) == protocolUnified {
			effectiveProtocol = protocolUnified
		}
	}

	switch effectiveProtocol {
	case protocolAlertmanager:
		return normalizeAlertmanager(raw, compactRaw)
	case protocolUnified:
		return normalizeUnified(raw, compactRaw)
	default:
		return nil, ErrInvalidPayload
	}
}

func normalizeAlertmanager(raw []byte, compactRaw string) ([]NormalizedAlert, error) {
	var payload struct {
		Receiver string `json:"receiver"`
		Alerts   []struct {
			Status      string            `json:"status"`
			Fingerprint string            `json:"fingerprint"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			StartsAt    *time.Time        `json:"startsAt"`
			EndsAt      *time.Time        `json:"endsAt"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, ErrInvalidPayload
	}
	if len(payload.Alerts) == 0 {
		return nil, ErrInvalidPayload
	}

	out := make([]NormalizedAlert, 0, len(payload.Alerts))
	for _, alert := range payload.Alerts {
		source := protocolAlertmanager
		labels := map[string]string{}
		for k, v := range alert.Labels {
			labels[k] = v
		}
		if receiver := strings.TrimSpace(payload.Receiver); receiver != "" {
			labels["am_receiver"] = receiver
		}

		title := strings.TrimSpace(labels["alertname"])
		out = append(out, NormalizedAlert{
			Source:          source,
			Protocol:        protocolAlertmanager,
			Fingerprint:     strings.TrimSpace(alert.Fingerprint),
			Status:          strings.TrimSpace(alert.Status),
			Severity:        strings.TrimSpace(labels["severity"]),
			Title:           title,
			Target:          firstNonEmpty(labels["instance"], labels["pod"], labels["node"]),
			LabelsJSON:      mustJSON(labels, "{}"),
			AnnotationsJSON: mustJSON(alert.Annotations, "{}"),
			RawPayloadJSON:  compactRaw,
			StartsAt:        alert.StartsAt,
			EndsAt:          alert.EndsAt,
		})
	}
	return out, nil
}

func normalizeUnified(raw []byte, compactRaw string) ([]NormalizedAlert, error) {
	var payload struct {
		Kind   string `json:"kind"`
		Source string `json:"source"`
		Alerts []struct {
			Status      string         `json:"status"`
			Fingerprint string         `json:"fingerprint"`
			Severity    string         `json:"severity"`
			Title       string         `json:"title"`
			Target      string         `json:"target"`
			Labels      map[string]any `json:"labels"`
			Annotations map[string]any `json:"annotations"`
			StartsAt    string         `json:"startsAt"`
			EndsAt      string         `json:"endsAt"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, ErrInvalidPayload
	}
	if len(payload.Alerts) == 0 {
		return nil, ErrInvalidPayload
	}

	source := strings.TrimSpace(payload.Source)
	if source == "" {
		source = protocolUnified
	}

	out := make([]NormalizedAlert, 0, len(payload.Alerts))
	for _, alert := range payload.Alerts {
		startsAt, err := parseRFC3339Ptr(alert.StartsAt)
		if err != nil {
			return nil, ErrInvalidPayload
		}
		endsAt, err := parseRFC3339Ptr(alert.EndsAt)
		if err != nil {
			return nil, ErrInvalidPayload
		}

		out = append(out, NormalizedAlert{
			Source:          source,
			Protocol:        protocolUnified,
			Fingerprint:     strings.TrimSpace(alert.Fingerprint),
			Status:          strings.TrimSpace(alert.Status),
			Severity:        strings.TrimSpace(alert.Severity),
			Title:           strings.TrimSpace(alert.Title),
			Target:          strings.TrimSpace(alert.Target),
			LabelsJSON:      mustJSON(alert.Labels, "{}"),
			AnnotationsJSON: mustJSON(alert.Annotations, "{}"),
			RawPayloadJSON:  compactRaw,
			StartsAt:        startsAt,
			EndsAt:          endsAt,
		})
	}
	return out, nil
}

func normalizeProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func compactJSON(raw []byte) (string, error) {
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		return "", err
	}
	return out.String(), nil
}

func mustJSON(v any, fallback string) string {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return fallback
	}
	return string(b)
}

func parseRFC3339Ptr(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
