package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer creates an httptest server that records the last request and
// responds with the given status/body.
func newTestServer(t *testing.T, status int, body string) (*httptest.Server, *http.Request) {
	t.Helper()
	var lastReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReq = r.Clone(r.Context())
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, lastReq
}

// helper that creates a server returning the given request details
func newCapturingServer(t *testing.T, status int, body string) (*httptest.Server, func() *http.Request) {
	t.Helper()
	var captured *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, func() *http.Request { return captured }
}

// ---------------------------------------------------------------------------
// NewClient
// ---------------------------------------------------------------------------

func TestNewClient_trims_trailing_slash(t *testing.T) {
	c := NewClient("http://example.com/", "key")
	if c.base != "http://example.com" {
		t.Errorf("base = %q, want trailing slash removed", c.base)
	}
}

func TestNewClient_preserves_base_without_trailing_slash(t *testing.T) {
	c := NewClient("http://example.com", "key")
	if c.base != "http://example.com" {
		t.Errorf("base = %q, want %q", c.base, "http://example.com")
	}
}

func TestNewClient_stores_api_key(t *testing.T) {
	c := NewClient("http://x", "secret-123")
	if c.apiKey != "secret-123" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "secret-123")
	}
}

// ---------------------------------------------------------------------------
// buildURL
// ---------------------------------------------------------------------------

func TestBuildURL_without_query(t *testing.T) {
	c := NewClient("http://host:8080", "")
	got := c.buildURL("/api/test", nil)
	if got != "http://host:8080/api/test" {
		t.Errorf("buildURL = %q", got)
	}
}

func TestBuildURL_with_query(t *testing.T) {
	c := NewClient("http://host:8080", "")
	q := make(map[string][]string)
	q["days"] = []string{"7"}
	got := c.buildURL("/api/test", q)
	if got != "http://host:8080/api/test?days=7" {
		t.Errorf("buildURL = %q", got)
	}
}

// ---------------------------------------------------------------------------
// Authorization header
// ---------------------------------------------------------------------------

func TestDo_sets_authorization_header_when_api_key_present(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"ok":true}`)
	c := NewClient(srv.URL, "my-key")

	_, err := c.GetEarningsSummary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req == nil {
		t.Fatal("no request captured")
	}
	auth := req.Header.Get("Authorization")
	if auth != "Bearer my-key" {
		t.Errorf("Authorization = %q, want %q", auth, "Bearer my-key")
	}
}

func TestDo_omits_authorization_header_when_api_key_empty(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"ok":true}`)
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsSummary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req == nil {
		t.Fatal("no request captured")
	}
	auth := req.Header.Get("Authorization")
	if auth != "" {
		t.Errorf("Authorization = %q, want empty", auth)
	}
}

// ---------------------------------------------------------------------------
// HTTP error handling
// ---------------------------------------------------------------------------

func TestDo_returns_error_on_4xx_status(t *testing.T) {
	srv, _ := newCapturingServer(t, 404, "not found")
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected error for 404 status")
	}
	if !strings.Contains(err.Error(), "CashPilot error 404") {
		t.Errorf("error = %q, want it to contain status code", err.Error())
	}
}

func TestDo_returns_error_on_5xx_status(t *testing.T) {
	srv, _ := newCapturingServer(t, 500, "internal server error")
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
	if !strings.Contains(err.Error(), "CashPilot error 500") {
		t.Errorf("error = %q, want it to contain status code", err.Error())
	}
}

func TestDo_returns_body_in_error_message(t *testing.T) {
	srv, _ := newCapturingServer(t, 403, "forbidden-detail")
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "forbidden-detail") {
		t.Errorf("error = %q, want it to contain response body", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Resource methods (GET, no params)
// ---------------------------------------------------------------------------

func TestGetEarningsSummary_sends_correct_request(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"total":42}`)
	c := NewClient(srv.URL, "")

	body, err := c.GetEarningsSummary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"total":42}` {
		t.Errorf("body = %q", string(body))
	}
	req := getReq()
	if req.Method != "GET" {
		t.Errorf("method = %q, want GET", req.Method)
	}
	if req.URL.Path != "/api/earnings/summary" {
		t.Errorf("path = %q", req.URL.Path)
	}
}

func TestGetEarningsBreakdown_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsBreakdown(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getReq().URL.Path != "/api/earnings/breakdown" {
		t.Errorf("path = %q", getReq().URL.Path)
	}
}

func TestGetServicesDeployed_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	_, err := c.GetServicesDeployed(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getReq().URL.Path != "/api/services/deployed" {
		t.Errorf("path = %q", getReq().URL.Path)
	}
}

func TestGetServicesCatalog_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	_, err := c.GetServicesCatalog(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getReq().URL.Path != "/api/services" {
		t.Errorf("path = %q", getReq().URL.Path)
	}
}

func TestGetFleetSummary_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{}`)
	c := NewClient(srv.URL, "")

	_, err := c.GetFleetSummary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getReq().URL.Path != "/api/fleet/summary" {
		t.Errorf("path = %q", getReq().URL.Path)
	}
}

func TestGetWorkers_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	_, err := c.GetWorkers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getReq().URL.Path != "/api/workers" {
		t.Errorf("path = %q", getReq().URL.Path)
	}
}

func TestGetHealthScores_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{}`)
	c := NewClient(srv.URL, "")

	_, err := c.GetHealthScores(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getReq().URL.Path != "/api/health/scores" {
		t.Errorf("path = %q", getReq().URL.Path)
	}
}

func TestGetCollectorAlerts_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	_, err := c.GetCollectorAlerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getReq().URL.Path != "/api/collector-alerts" {
		t.Errorf("path = %q", getReq().URL.Path)
	}
}

// ---------------------------------------------------------------------------
// Tool methods (GET with params)
// ---------------------------------------------------------------------------

func TestGetEarningsDaily_sends_days_query_param(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsDaily(context.Background(), 14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.URL.Path != "/api/earnings/daily" {
		t.Errorf("path = %q", req.URL.Path)
	}
	if req.URL.Query().Get("days") != "14" {
		t.Errorf("days param = %q, want %q", req.URL.Query().Get("days"), "14")
	}
}

func TestGetEarningsHistory_sends_period_query_param(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsHistory(context.Background(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.URL.Path != "/api/earnings/history" {
		t.Errorf("path = %q", req.URL.Path)
	}
	if req.URL.Query().Get("period") != "month" {
		t.Errorf("period param = %q", req.URL.Query().Get("period"))
	}
}

func TestGetServiceLogs_sends_slug_and_lines(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `"logs..."`)
	c := NewClient(srv.URL, "")

	_, err := c.GetServiceLogs(context.Background(), "honeygain", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.URL.Path != "/api/services/honeygain/logs" {
		t.Errorf("path = %q", req.URL.Path)
	}
	if req.URL.Query().Get("lines") != "100" {
		t.Errorf("lines param = %q", req.URL.Query().Get("lines"))
	}
}

// ---------------------------------------------------------------------------
// Tool methods (POST)
// ---------------------------------------------------------------------------

func TestRestartService_sends_post_to_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"status":"ok"}`)
	c := NewClient(srv.URL, "")

	_, err := c.RestartService(context.Background(), "earnapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST", req.Method)
	}
	if req.URL.Path != "/api/services/earnapp/restart" {
		t.Errorf("path = %q", req.URL.Path)
	}
}

func TestStopService_sends_post_to_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"status":"ok"}`)
	c := NewClient(srv.URL, "")

	_, err := c.StopService(context.Background(), "peer2profit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST", req.Method)
	}
	if req.URL.Path != "/api/services/peer2profit/stop" {
		t.Errorf("path = %q", req.URL.Path)
	}
}

func TestStartService_sends_post_to_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"status":"ok"}`)
	c := NewClient(srv.URL, "")

	_, err := c.StartService(context.Background(), "repocket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST", req.Method)
	}
	if req.URL.Path != "/api/services/repocket/start" {
		t.Errorf("path = %q", req.URL.Path)
	}
}

func TestDeployService_sends_post_with_body(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"deployed":true}`)
	c := NewClient(srv.URL, "")

	_, err := c.DeployService(context.Background(), "myservice", []byte(`{"env":{"KEY":"VAL"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST", req.Method)
	}
	if req.URL.Path != "/api/deploy/myservice" {
		t.Errorf("path = %q", req.URL.Path)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
}

func TestTriggerCollection_sends_post(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"triggered":true}`)
	c := NewClient(srv.URL, "")

	_, err := c.TriggerCollection(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST", req.Method)
	}
	if req.URL.Path != "/api/collect" {
		t.Errorf("path = %q", req.URL.Path)
	}
}

// ---------------------------------------------------------------------------
// Tool methods (DELETE)
// ---------------------------------------------------------------------------

func TestRemoveService_sends_delete_to_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{"removed":true}`)
	c := NewClient(srv.URL, "")

	_, err := c.RemoveService(context.Background(), "proxylite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", req.Method)
	}
	if req.URL.Path != "/api/services/proxylite" {
		t.Errorf("path = %q", req.URL.Path)
	}
}

// ---------------------------------------------------------------------------
// GetCompose
// ---------------------------------------------------------------------------

func TestGetCompose_sends_correct_path(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `version: "3"`)
	c := NewClient(srv.URL, "")

	_, err := c.GetCompose(context.Background(), "honeygain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	if req.Method != "GET" {
		t.Errorf("method = %q, want GET", req.Method)
	}
	if req.URL.Path != "/api/compose/honeygain" {
		t.Errorf("path = %q", req.URL.Path)
	}
}

// ---------------------------------------------------------------------------
// Slug URL-encoding
// ---------------------------------------------------------------------------

func TestServiceLogs_url_encodes_slug_with_special_chars(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `""`)
	c := NewClient(srv.URL, "")

	_, err := c.GetServiceLogs(context.Background(), "my/service", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := getReq()
	// url.PathEscape("my/service") = "my%2Fservice"
	if !strings.Contains(req.URL.RawPath, "my%2Fservice") && !strings.Contains(req.RequestURI, "my%2Fservice") {
		t.Errorf("slug not URL-encoded in path: raw=%q uri=%q", req.URL.RawPath, req.RequestURI)
	}
}

// ---------------------------------------------------------------------------
// Connection error
// ---------------------------------------------------------------------------

func TestDoGet_returns_error_when_server_unreachable(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "") // nothing listening on port 1

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected connection error")
	}
}

// ---------------------------------------------------------------------------
// NewRequestWithContext error paths (invalid URL)
// ---------------------------------------------------------------------------

func TestDoGet_returns_error_when_url_is_invalid(t *testing.T) {
	c := NewClient("http://host\x7f:8080", "") // control char makes URL invalid
	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid URL in doGet")
	}
}

func TestDoPost_returns_error_when_url_is_invalid(t *testing.T) {
	c := NewClient("http://host\x7f:8080", "") // control char makes URL invalid
	_, err := c.TriggerCollection(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid URL in doPost")
	}
}

func TestRemoveService_returns_error_when_url_is_invalid(t *testing.T) {
	c := NewClient("http://host\x7f:8080", "") // control char makes URL invalid
	_, err := c.RemoveService(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for invalid URL in RemoveService")
	}
}
