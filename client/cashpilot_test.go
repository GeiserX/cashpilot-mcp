package client

import (
	"context"
	"errors"
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
	srv, _ := newCapturingServer(t, 404, `{"detail":"not found"}`)
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected error for 404 status")
	}
	if !strings.Contains(err.Error(), "CashPilot answered 404") {
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
	if !strings.Contains(err.Error(), "CashPilot answered 500") {
		t.Errorf("error = %q, want it to contain status code", err.Error())
	}
}

func TestDo_quotes_fastapi_detail_in_error_message(t *testing.T) {
	srv, _ := newCapturingServer(t, 403, `{"detail":"forbidden-detail"}`)
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "forbidden-detail") || !strings.Contains(err.Error(), "read-only key") {
		t.Errorf("error = %q, want the detail and the role hint", err.Error())
	}
}

func TestDo_never_echoes_a_non_json_error_body(t *testing.T) {
	srv, _ := newCapturingServer(t, 502, "<html><body>proxy at 10.1.2.3 failed</body></html>")
	c := NewClient(srv.URL, "")

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "10.1.2.3") || strings.Contains(err.Error(), "<html>") {
		t.Errorf("error leaks the raw body: %q", err.Error())
	}
}

func TestDo_adds_worker_hint_when_several_workers_are_online(t *testing.T) {
	srv, _ := newCapturingServer(t, 400, `{"detail":"worker_id is required (multiple workers online)"}`)
	c := NewClient(srv.URL, "")

	_, err := c.RestartService(context.Background(), "honeygain", Target{})
	if err == nil || !strings.Contains(err.Error(), "list_workers") {
		t.Fatalf("error = %v, want a pointer to list_workers", err)
	}
}

func TestDo_refuses_an_oversized_response(t *testing.T) {
	srv, _ := newCapturingServer(t, 200, strings.Repeat("x", MaxResponseBytes+1))
	c := NewClient(srv.URL, "")

	if _, err := c.GetEarningsSummary(context.Background()); err == nil {
		t.Fatal("expected an error for a response over the cap")
	}
}

func TestDo_transport_error_hides_the_cashpilot_url(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "")

	_, err := c.GetEarningsSummary(context.Background())
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), "127.0.0.1") {
		t.Errorf("error leaks the URL: %q", err.Error())
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

	_, err := c.GetHealthScores(context.Background(), 0)
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

	_, err := c.GetServiceLogs(context.Background(), "honeygain", 100, Target{})
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

	_, err := c.RestartService(context.Background(), "earnapp", Target{})
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

	_, err := c.StopService(context.Background(), "peer2profit", Target{})
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

	_, err := c.StartService(context.Background(), "repocket", Target{})
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

	_, err := c.DeployService(context.Background(), "myservice", []byte(`{"env":{"KEY":"VAL"}}`), Target{})
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

	_, err := c.RemoveService(context.Background(), "proxylite", Target{})
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

	_, err := c.GetCompose(context.Background(), "honeygain", "")
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

func TestServiceMethods_refuse_invalid_slugs_without_a_request(t *testing.T) {
	sent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { sent = true }))
	t.Cleanup(srv.Close)
	c := NewClient(srv.URL, "")
	ctx := context.Background()
	for _, slug := range []string{"", ".", "..", "my/service", "../fleet", "Honeygain", "a?b", "a#b", "a b", strings.Repeat("a", 65)} {
		calls := map[string]func() error{
			"logs":    func() error { _, err := c.GetServiceLogs(ctx, slug, 1, Target{}); return err },
			"restart": func() error { _, err := c.RestartService(ctx, slug, Target{}); return err },
			"stop":    func() error { _, err := c.StopService(ctx, slug, Target{}); return err },
			"start":   func() error { _, err := c.StartService(ctx, slug, Target{}); return err },
			"deploy":  func() error { _, err := c.DeployService(ctx, slug, []byte("{}"), Target{}); return err },
			"remove":  func() error { _, err := c.RemoveService(ctx, slug, Target{}); return err },
			"compose": func() error { _, err := c.GetCompose(ctx, slug, ""); return err },
		}
		for name, call := range calls {
			if err := call(); !errors.Is(err, ErrInvalidSlug) {
				t.Errorf("%s(%q): err = %v, want ErrInvalidSlug", name, slug, err)
			}
		}
	}
	if sent {
		t.Error("a request reached CashPilot for an invalid slug")
	}
	for _, ok := range []string{"honeygain", "sentinel-dvpn", "proxybase-xyz", "earnfm", "a.b_c"} {
		if err := ValidateSlug(ok); err != nil {
			t.Errorf("ValidateSlug(%q) = %v, want nil", ok, err)
		}
	}
}

func TestTarget_sends_worker_id_and_instance(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `{}`)
	c := NewClient(srv.URL, "")

	if _, err := c.StopService(context.Background(), "honeygain", Target{WorkerID: 3, Instance: 2}); err != nil {
		t.Fatal(err)
	}
	q := getReq().URL.Query()
	if q.Get("worker_id") != "3" || q.Get("instance") != "2" {
		t.Errorf("query = %v, want worker_id=3 instance=2", q)
	}

	if _, err := c.RemoveService(context.Background(), "honeygain", Target{WorkerID: 4, Instance: 9}); err != nil {
		t.Fatal(err)
	}
	q = getReq().URL.Query()
	if q.Get("worker_id") != "4" || q.Has("instance") || q.Has("delete_volumes") {
		t.Errorf("remove query = %v, want worker_id=4 only", q)
	}

	if _, err := c.StartService(context.Background(), "honeygain", Target{}); err != nil {
		t.Fatal(err)
	}
	if raw := getReq().URL.RawQuery; raw != "" {
		t.Errorf("zero Target sent %q, want no query", raw)
	}
}

func TestGetCompose_and_health_scores_send_optional_params(t *testing.T) {
	srv, getReq := newCapturingServer(t, 200, `[]`)
	c := NewClient(srv.URL, "")

	if _, err := c.GetCompose(context.Background(), "honeygain", "arm64"); err != nil {
		t.Fatal(err)
	}
	if got := getReq().URL.Query().Get("arch"); got != "arm64" {
		t.Errorf("arch = %q, want arm64", got)
	}
	if _, err := c.GetHealthScores(context.Background(), 30); err != nil {
		t.Fatal(err)
	}
	if got := getReq().URL.Query().Get("days"); got != "30" {
		t.Errorf("days = %q, want 30", got)
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
	_, err := c.RemoveService(context.Background(), "test", Target{})
	if err == nil {
		t.Fatal("expected error for invalid URL in RemoveService")
	}
}
