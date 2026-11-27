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
	if !strings.Contains(err.Error(), "forbidden-detail") || !strings.Contains(err.Error(), "This API key's role does not allow this call.") {
		t.Errorf("error = %q, want the detail and the role hint", err.Error())
	}
}

// The hint after a 403 or 404 names what is wrong, from CashPilot's detail
// (app/deps.py for the 403s, app/main.py "Worker not found" for the 404).
func TestDo_hint_matches_the_refusal(t *testing.T) {
	cases := []struct {
		name, body    string
		status        int
		want, notWant string
	}{
		{"read-only key", `{"detail":"Read-only token"}`, 403,
			"The read-only key allows only get_earnings_summary, get_earnings_breakdown, get_fleet_summary, get_health_scores and list_deployed_services",
			"cannot change services or read logs"},
		{"fleet key", `{"detail":"The fleet key only enrolls workers"}`, 403,
			"Use the admin key or the read-only key, never the fleet key.", "read-only key allows only"},
		{"owner only", `{"detail":"Owner access required"}`, 403,
			"This API key's role does not allow this call.", "read-only key"},
		{"unknown worker", `{"detail":"Worker not found"}`, 404,
			"Check worker_id with list_workers.", "Check the slug"},
		{"unknown worker by id", `{"detail":"Unknown worker 99"}`, 404,
			"Check worker_id with list_workers.", "Check the slug"},
		{"unknown service", `{"detail":"Service 'nope' not found"}`, 404,
			"Check the slug with list_deployed_services or list_available_services.", "list_workers"},
		{"service named worker", `{"detail":"Service 'worker' not found"}`, 404,
			"Check the slug with list_deployed_services or list_available_services.", "list_workers"},
		{"worker that cannot run containers", `{"detail":"No such worker, or it cannot run containers."}`, 404,
			"Check worker_id with list_workers.", "Check the slug"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newCapturingServer(t, tc.status, tc.body)
			_, err := NewClient(srv.URL, "").StopService(context.Background(), "honeygain", Target{WorkerID: 99})
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), tc.notWant) {
				t.Errorf("error = %q, want %q and not %q", err.Error(), tc.want, tc.notWant)
			}
		})
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

// The 409 the web dashboard answers when a deploy would replace a running
// container (CashPilot 3.0, app/main.py _ask_deploy_confirmation).
const confirmDeployTargetsBody = `{"detail":{"error":"confirm_deploy_targets",` +
	`"message":"Replace these containers for honeygain on box-1?\n\nInstance 1 (0123456789ab)\nProxied copy 3 (abcdef012345)",` +
	`"expected_containers":{"1":"0123456789abcdef","2":"none","p3":"abcdef0123456789","../x":"0123456789ab","4":"NOT-AN-ID"}}}`

func TestDo_reads_message_and_code_of_a_confirm_deploy_targets_detail(t *testing.T) {
	srv, _ := newCapturingServer(t, 409, confirmDeployTargetsBody)
	c := NewClient(srv.URL, "")

	_, err := c.DeployService(context.Background(), "honeygain", []byte(`{}`), Target{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want an *APIError", err)
	}
	if apiErr.Code != CodeConfirmDeployTargets {
		t.Errorf("Code = %q, want %q", apiErr.Code, CodeConfirmDeployTargets)
	}
	wantDetail := "Replace these containers for honeygain on box-1? Instance 1 (0123456789ab) Proxied copy 3 (abcdef012345)"
	if apiErr.Detail != wantDetail {
		t.Errorf("Detail = %q, want %q", apiErr.Detail, wantDetail)
	}
	want := map[string]string{"1": "0123456789abcdef", "2": "none", "p3": "abcdef0123456789"}
	if len(apiErr.Containers) != len(want) {
		t.Fatalf("Containers = %v, want %v", apiErr.Containers, want)
	}
	for k, v := range want {
		if apiErr.Containers[k] != v {
			t.Errorf("Containers[%q] = %q, want %q", k, apiErr.Containers[k], v)
		}
	}
	msg := err.Error()
	if !strings.Contains(msg, wantDetail+" (confirm_deploy_targets)") {
		t.Errorf("error = %q, want the message and the code", msg)
	}
	if strings.Contains(msg, "rejected as invalid") || strings.Contains(msg, "{") {
		t.Errorf("error = %q, still reads the dict detail as noise", msg)
	}
}

func TestDo_reads_message_and_code_of_a_deploy_target_changed_detail(t *testing.T) {
	// The worker's refusal as the web dashboard forwards it (_safe_worker_detail).
	body := `{"detail":{"error":"deploy_target_changed",` +
		`"message":"The container changed or needs approval. Review it and use Redeploy.",` +
		`"current_container":"0123456789abcdef"}}`
	srv, _ := newCapturingServer(t, 409, body)
	c := NewClient(srv.URL, "")

	_, err := c.DeployService(context.Background(), "honeygain", []byte(`{}`), Target{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want an *APIError", err)
	}
	if apiErr.Code != CodeDeployTargetChanged || apiErr.Containers != nil || apiErr.Current != "0123456789abcdef" {
		t.Errorf("Code = %q, Containers = %v, Current = %q", apiErr.Code, apiErr.Containers, apiErr.Current)
	}
	want := "CashPilot answered 409: The container changed or needs approval. Review it and use Redeploy. (deploy_target_changed)"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestDo_drops_a_current_container_that_is_not_an_id(t *testing.T) {
	for _, current := range []string{`"../etc"`, `"0123ABCDEF"`, `123`, `null`} {
		body := `{"detail":{"error":"deploy_target_changed","message":"m","current_container":` + current + `}}`
		srv, _ := newCapturingServer(t, 409, body)
		c := NewClient(srv.URL, "")
		_, err := c.DeployService(context.Background(), "honeygain", []byte(`{}`), Target{})
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err = %v, want an *APIError", err)
		}
		if apiErr.Current != "" {
			t.Errorf("current_container %s: Current = %q, want empty", current, apiErr.Current)
		}
	}
	// current_container belongs to deploy_target_changed only.
	body := `{"detail":{"error":"confirm_deploy_targets","message":"m","current_container":"0123456789abcdef"}}`
	srv, _ := newCapturingServer(t, 409, body)
	_, err := NewClient(srv.URL, "").DeployService(context.Background(), "honeygain", []byte(`{}`), Target{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Current != "" {
		t.Errorf("err = %v, want an *APIError with no Current", err)
	}
}

func TestDo_dict_details_without_a_usable_message(t *testing.T) {
	cases := []struct {
		name, body, wantDetail, wantCode string
	}{
		{"code only", `{"detail":{"error":"copy_not_found"}}`, "copy_not_found", "copy_not_found"},
		{"empty message falls back to the code", `{"detail":{"error":"route_locked","message":""}}`, "route_locked", "route_locked"},
		{"non-string fields", `{"detail":{"error":["x"],"message":{"a":1}}}`, "the request was rejected as invalid", ""},
		{"code that is not a code", `{"detail":{"error":"<b>evil</b>","message":"refused"}}`, "refused", ""},
		{"validation array", `{"detail":[{"loc":["body","env"],"msg":"bad"}]}`, "the request was rejected as invalid", ""},
		{"plain string", `{"detail":"Missing required fields: EMAIL"}`, "Missing required fields: EMAIL", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newCapturingServer(t, 400, tc.body)
			c := NewClient(srv.URL, "")
			_, err := c.GetEarningsSummary(context.Background())
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want an *APIError", err)
			}
			if apiErr.Detail != tc.wantDetail || apiErr.Code != tc.wantCode {
				t.Errorf("Detail, Code = %q, %q; want %q, %q", apiErr.Detail, apiErr.Code, tc.wantDetail, tc.wantCode)
			}
		})
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
