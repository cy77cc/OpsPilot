package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TrivySummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

type TrivyScanResult struct {
	Summary TrivySummary `json:"summary"`
}

type TrivyClient interface {
	ScanImage(ctx context.Context, image string) (TrivyScanResult, error)
}

type HTTPTrivyClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPTrivyClient(baseURL string) *HTTPTrivyClient {
	return &HTTPTrivyClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *HTTPTrivyClient) ScanImage(ctx context.Context, image string) (TrivyScanResult, error) {
	payload := map[string]string{"image": image}
	buf, err := json.Marshal(payload)
	if err != nil {
		return TrivyScanResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/scan", bytes.NewReader(buf))
	if err != nil {
		return TrivyScanResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return TrivyScanResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return TrivyScanResult{}, fmt.Errorf("trivy scan failed: status=%d", resp.StatusCode)
	}

	var out TrivyScanResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TrivyScanResult{}, err
	}
	return out, nil
}
