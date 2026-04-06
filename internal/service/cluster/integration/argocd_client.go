package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ArgoSyncResult struct {
	Status   string `json:"status"`
	Revision string `json:"revision"`
}

type ArgoCDClient interface {
	Sync(ctx context.Context, app string) (ArgoSyncResult, error)
}

type HTTPArgoCDClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewHTTPArgoCDClient(baseURL string, token string) *HTTPArgoCDClient {
	return &HTTPArgoCDClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   strings.TrimSpace(token),
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *HTTPArgoCDClient) Sync(ctx context.Context, app string) (ArgoSyncResult, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s/sync", c.baseURL, app)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return ArgoSyncResult{}, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return ArgoSyncResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return ArgoSyncResult{}, fmt.Errorf("argocd sync failed: status=%d", resp.StatusCode)
	}

	var out ArgoSyncResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ArgoSyncResult{}, err
	}
	return out, nil
}
