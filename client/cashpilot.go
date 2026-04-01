package client

import (
	"bytes"
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

// ---------------------------------------------------------------------------
// Resource methods (GET, no params)
// ---------------------------------------------------------------------------

func (c *Client) GetEarningsSummary() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/earnings/summary", nil), nil)
	return c.do(req)
}

func (c *Client) GetEarningsBreakdown() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/earnings/breakdown", nil), nil)
	return c.do(req)
}

func (c *Client) GetServicesDeployed() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/services/deployed", nil), nil)
	return c.do(req)
}

func (c *Client) GetServicesCatalog() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/services", nil), nil)
	return c.do(req)
}

func (c *Client) GetFleetSummary() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/fleet/summary", nil), nil)
	return c.do(req)
}

func (c *Client) GetWorkers() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/workers", nil), nil)
	return c.do(req)
}

func (c *Client) GetHealthScores() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/health/scores", nil), nil)
	return c.do(req)
}

func (c *Client) GetCollectorAlerts() ([]byte, error) {
	req, _ := http.NewRequest("GET", c.buildURL("/api/collector-alerts", nil), nil)
	return c.do(req)
}

// ---------------------------------------------------------------------------
// Tool methods (GET with params, POST, DELETE)
// ---------------------------------------------------------------------------

func (c *Client) GetEarningsDaily(days int) ([]byte, error) {
	q := url.Values{}
	q.Set("days", fmt.Sprintf("%d", days))
	req, _ := http.NewRequest("GET", c.buildURL("/api/earnings/daily", q), nil)
	return c.do(req)
}

func (c *Client) GetEarningsHistory(period string) ([]byte, error) {
	q := url.Values{}
	q.Set("period", period)
	req, _ := http.NewRequest("GET", c.buildURL("/api/earnings/history", q), nil)
	return c.do(req)
}

func (c *Client) GetServiceLogs(slug string, lines int) ([]byte, error) {
	q := url.Values{}
	q.Set("lines", fmt.Sprintf("%d", lines))
	path := fmt.Sprintf("/api/services/%s/logs", url.PathEscape(slug))
	req, _ := http.NewRequest("GET", c.buildURL(path, q), nil)
	return c.do(req)
}

func (c *Client) RestartService(slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s/restart", url.PathEscape(slug))
	req, _ := http.NewRequest("POST", c.buildURL(path, nil), nil)
	return c.do(req)
}

func (c *Client) StopService(slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s/stop", url.PathEscape(slug))
	req, _ := http.NewRequest("POST", c.buildURL(path, nil), nil)
	return c.do(req)
}

func (c *Client) StartService(slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s/start", url.PathEscape(slug))
	req, _ := http.NewRequest("POST", c.buildURL(path, nil), nil)
	return c.do(req)
}

func (c *Client) DeployService(slug string, body []byte) ([]byte, error) {
	path := fmt.Sprintf("/api/deploy/%s", url.PathEscape(slug))
	req, _ := http.NewRequest("POST", c.buildURL(path, nil), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) RemoveService(slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/services/%s", url.PathEscape(slug))
	req, _ := http.NewRequest("DELETE", c.buildURL(path, nil), nil)
	return c.do(req)
}

func (c *Client) TriggerCollection() ([]byte, error) {
	req, _ := http.NewRequest("POST", c.buildURL("/api/collect", nil), nil)
	return c.do(req)
}

func (c *Client) GetCompose(slug string) ([]byte, error) {
	path := fmt.Sprintf("/api/compose/%s", url.PathEscape(slug))
	req, _ := http.NewRequest("GET", c.buildURL(path, nil), nil)
	return c.do(req)
}
