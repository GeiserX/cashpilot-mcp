package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base   string
	hc     *http.Client
	apiKey string
}

func NewClient(base, apiKey string) *Client {
	return &Client{
		base:   strings.TrimRight(base, "/"),
		hc:     &http.Client{Timeout: 30 * time.Second},
		apiKey: apiKey,
	}
}

func (c *Client) buildURL(path string, q url.Values) string {
	u := c.base + path
	if q != nil && len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CashPilot error %d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) doGet(ctx context.Context, path string, q url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.buildURL(path, q), nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) doPost(ctx context.Context, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.buildURL(path, nil), body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req)
}

// ---------------------------------------------------------------------------
// Resource methods (GET, no params)
// ---------------------------------------------------------------------------

func (c *Client) GetEarningsSummary(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/earnings/summary", nil)
}

func (c *Client) GetEarningsBreakdown(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/earnings/breakdown", nil)
}

func (c *Client) GetServicesDeployed(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/services/deployed", nil)
}

func (c *Client) GetServicesCatalog(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/services", nil)
}

func (c *Client) GetFleetSummary(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/fleet/summary", nil)
}

func (c *Client) GetWorkers(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/workers", nil)
}

func (c *Client) GetHealthScores(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/health/scores", nil)
}

func (c *Client) GetCollectorAlerts(ctx context.Context) ([]byte, error) {
	return c.doGet(ctx, "/api/collector-alerts", nil)
}

// ---------------------------------------------------------------------------
// Tool methods (GET with params, POST, DELETE)
// ---------------------------------------------------------------------------

func (c *Client) GetEarningsDaily(ctx context.Context, days int) ([]byte, error) {
	q := url.Values{}
	q.Set("days", fmt.Sprintf("%d", days))
	return c.doGet(ctx, "/api/earnings/daily", q)
}

func (c *Client) GetEarningsHistory(ctx context.Context, period string) ([]byte, error) {
	q := url.Values{}
	q.Set("period", period)
	return c.doGet(ctx, "/api/earnings/history", q)
}

func (c *Client) GetServiceLogs(ctx context.Context, slug string, lines int) ([]byte, error) {
	q := url.Values{}
	q.Set("lines", fmt.Sprintf("%d", lines))
	path := fmt.Sprintf("/api/services/%s/logs", url.PathEscape(slug))
	return c.doGet(ctx, path, q)
}

func (c *Client) RestartService(ctx context.Context, slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s/restart", url.PathEscape(slug))
	return c.doPost(ctx, path, nil)
}

func (c *Client) StopService(ctx context.Context, slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s/stop", url.PathEscape(slug))
	return c.doPost(ctx, path, nil)
}

func (c *Client) StartService(ctx context.Context, slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s/start", url.PathEscape(slug))
	return c.doPost(ctx, path, nil)
}

func (c *Client) DeployService(ctx context.Context, slug string, body []byte) ([]byte, error) {
	path := fmt.Sprintf("/api/deploy/%s", url.PathEscape(slug))
	return c.doPost(ctx, path, bytes.NewReader(body))
}

func (c *Client) RemoveService(ctx context.Context, slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s", url.PathEscape(slug))
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.buildURL(path, nil), nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) TriggerCollection(ctx context.Context) ([]byte, error) {
	return c.doPost(ctx, "/api/collect", nil)
}

func (c *Client) GetCompose(ctx context.Context, slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/compose/%s", url.PathEscape(slug))
	return c.doGet(ctx, path, nil)
}
