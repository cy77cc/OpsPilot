package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HarborProject struct {
	Name string `json:"name"`
}

type HarborClient interface {
	ListProjects(ctx context.Context) ([]HarborProject, error)
}

type HTTPHarborClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewHTTPHarborClient(baseURL string, token string) *HTTPHarborClient {
	return &HTTPHarborClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   strings.TrimSpace(token),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *HTTPHarborClient) ListProjects(ctx context.Context) ([]HarborProject, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2.0/projects", nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("harbor list projects failed: status=%d", resp.StatusCode)
	}

	var out []HarborProject
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
