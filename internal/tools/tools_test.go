package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// mockAPI returns an httptest server that routes requests to a handler map.
// Each entry maps "METHOD /path" to a status+body pair.
func mockAPI(t *testing.T, routes map[string]struct {
	status int
	body   string
}) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if route, ok := routes[key]; ok {
			w.WriteHeader(route.status)
			_, _ = w.Write([]byte(route.body))
			return
		}
		// Also try matching with prefix for paths that have query strings
		for k, route := range routes {
			parts := strings.SplitN(k, " ", 2)
			if len(parts) == 2 && r.Method == parts[0] && r.URL.Path == parts[1] {
				w.WriteHeader(route.status)
				_, _ = w.Write([]byte(route.body))
				return
			}
		}
		w.WriteHeader(404)
		_, _ = w.Write([]byte("route not found: " + key))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func makeCallToolRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

// ---------------------------------------------------------------------------
// get_earnings_daily
// ---------------------------------------------------------------------------

func TestGetEarningsDaily_returns_daily_earnings_with_default_days(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/daily": {200, `[{"day":"2025-01-01","amount":1.5}]`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsDaily(c)

	req := makeCallToolRequest("get_earnings_daily", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result is error: %+v", result)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "7 days") {
		t.Errorf("expected default 7 days in response, got: %s", text)
	}
	if !strings.Contains(text, "amount") {
		t.Errorf("expected response body in text, got: %s", text)
	}
}

func TestGetEarningsDaily_respects_custom_days_parameter(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/daily": {200, `[]`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsDaily(c)

	req := makeCallToolRequest("get_earnings_daily", map[string]any{"days": float64(30)})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "30 days") {
		t.Errorf("expected 30 days in response, got: %s", text)
	}
}

func TestGetEarningsDaily_clamps_days_to_365_maximum(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/daily": {200, `[]`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsDaily(c)

	req := makeCallToolRequest("get_earnings_daily", map[string]any{"days": float64(999)})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "365 days") {
		t.Errorf("expected clamped to 365 days, got: %s", text)
	}
}

func TestGetEarningsDaily_ignores_negative_days_uses_default(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/daily": {200, `[]`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsDaily(c)

	req := makeCallToolRequest("get_earnings_daily", map[string]any{"days": float64(-5)})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "7 days") {
		t.Errorf("expected default 7 days for negative input, got: %s", text)
	}
}

func TestGetEarningsDaily_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/daily": {500, "server down"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsDaily(c)

	req := makeCallToolRequest("get_earnings_daily", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// get_earnings_history
// ---------------------------------------------------------------------------

func TestGetEarningsHistory_returns_history_for_given_period(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/history": {200, `{"total":99.5}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsHistory(c)

	req := makeCallToolRequest("get_earnings_history", map[string]any{"period": "month"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "month") {
		t.Errorf("expected period in response, got: %s", text)
	}
	if !strings.Contains(text, "99.5") {
		t.Errorf("expected response body in text, got: %s", text)
	}
}

func TestGetEarningsHistory_returns_tool_error_when_period_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/history": {200, `{}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsHistory(c)

	req := makeCallToolRequest("get_earnings_history", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when period is missing")
	}
}

func TestGetEarningsHistory_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/earnings/history": {502, "bad gateway"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetEarningsHistory(c)

	req := makeCallToolRequest("get_earnings_history", map[string]any{"period": "week"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// get_service_logs
// ---------------------------------------------------------------------------

func TestGetServiceLogs_returns_logs_with_default_lines(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/services/honeygain/logs": {200, `"log line 1\nlog line 2"`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetServiceLogs(c)

	req := makeCallToolRequest("get_service_logs", map[string]any{"slug": "honeygain"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "50 lines") {
		t.Errorf("expected default 50 lines, got: %s", text)
	}
	if !strings.Contains(text, "honeygain") {
		t.Errorf("expected slug in response, got: %s", text)
	}
}

func TestGetServiceLogs_respects_custom_lines_parameter(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/services/earnapp/logs": {200, `""`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetServiceLogs(c)

	req := makeCallToolRequest("get_service_logs", map[string]any{
		"slug":  "earnapp",
		"lines": float64(200),
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "200 lines") {
		t.Errorf("expected 200 lines, got: %s", text)
	}
}

func TestGetServiceLogs_clamps_lines_to_1000_maximum(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/services/test/logs": {200, `""`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetServiceLogs(c)

	req := makeCallToolRequest("get_service_logs", map[string]any{
		"slug":  "test",
		"lines": float64(5000),
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "1000 lines") {
		t.Errorf("expected clamped to 1000, got: %s", text)
	}
}

func TestGetServiceLogs_returns_tool_error_when_slug_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetServiceLogs(c)

	req := makeCallToolRequest("get_service_logs", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when slug missing")
	}
}

func TestGetServiceLogs_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/services/broken/logs": {500, "error"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetServiceLogs(c)

	req := makeCallToolRequest("get_service_logs", map[string]any{"slug": "broken"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// restart_service
// ---------------------------------------------------------------------------

func TestRestartService_returns_success_response(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/services/earnapp/restart": {200, `{"status":"restarted"}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewRestartService(c)

	req := makeCallToolRequest("restart_service", map[string]any{"slug": "earnapp"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "earnapp restarted") {
		t.Errorf("expected restart confirmation, got: %s", text)
	}
}

func TestRestartService_returns_tool_error_when_slug_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(srv.URL, "")
	_, handler := NewRestartService(c)

	req := makeCallToolRequest("restart_service", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when slug missing")
	}
}

func TestRestartService_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/services/broken/restart": {500, "failed"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewRestartService(c)

	req := makeCallToolRequest("restart_service", map[string]any{"slug": "broken"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// stop_service
// ---------------------------------------------------------------------------

func TestStopService_returns_success_response(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/services/peer2profit/stop": {200, `{"status":"stopped"}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewStopService(c)

	req := makeCallToolRequest("stop_service", map[string]any{"slug": "peer2profit"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "peer2profit stopped") {
		t.Errorf("expected stop confirmation, got: %s", text)
	}
}

func TestStopService_returns_tool_error_when_slug_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(srv.URL, "")
	_, handler := NewStopService(c)

	req := makeCallToolRequest("stop_service", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when slug missing")
	}
}

func TestStopService_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/services/broken/stop": {503, "unavailable"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewStopService(c)

	req := makeCallToolRequest("stop_service", map[string]any{"slug": "broken"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// start_service
// ---------------------------------------------------------------------------

func TestStartService_returns_success_response(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/services/repocket/start": {200, `{"status":"started"}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewStartService(c)

	req := makeCallToolRequest("start_service", map[string]any{"slug": "repocket"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "repocket started") {
		t.Errorf("expected start confirmation, got: %s", text)
	}
}

func TestStartService_returns_tool_error_when_slug_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(srv.URL, "")
	_, handler := NewStartService(c)

	req := makeCallToolRequest("start_service", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when slug missing")
	}
}

func TestStartService_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/services/broken/start": {500, "error"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewStartService(c)

	req := makeCallToolRequest("start_service", map[string]any{"slug": "broken"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// remove_service
// ---------------------------------------------------------------------------

func TestRemoveService_returns_success_response(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"DELETE /api/services/proxylite": {200, `{"removed":true}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewRemoveService(c)

	req := makeCallToolRequest("remove_service", map[string]any{"slug": "proxylite"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "proxylite removed") {
		t.Errorf("expected remove confirmation, got: %s", text)
	}
}

func TestRemoveService_returns_tool_error_when_slug_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(srv.URL, "")
	_, handler := NewRemoveService(c)

	req := makeCallToolRequest("remove_service", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when slug missing")
	}
}

func TestRemoveService_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"DELETE /api/services/broken": {500, "error"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewRemoveService(c)

	req := makeCallToolRequest("remove_service", map[string]any{"slug": "broken"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// trigger_collection
// ---------------------------------------------------------------------------

func TestTriggerCollection_returns_success_response(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/collect": {200, `{"triggered":true}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewTriggerCollection(c)

	req := makeCallToolRequest("trigger_collection", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "Collection triggered") {
		t.Errorf("expected trigger confirmation, got: %s", text)
	}
}

func TestTriggerCollection_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/collect": {500, "error"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewTriggerCollection(c)

	req := makeCallToolRequest("trigger_collection", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// get_compose
// ---------------------------------------------------------------------------

func TestGetCompose_returns_compose_for_slug(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/compose/honeygain": {200, `version: "3"\nservices:\n  honeygain:\n    image: honeygain`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetCompose(c)

	req := makeCallToolRequest("get_compose", map[string]any{"slug": "honeygain"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "Compose for honeygain") {
		t.Errorf("expected compose header, got: %s", text)
	}
}

func TestGetCompose_returns_tool_error_when_slug_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetCompose(c)

	req := makeCallToolRequest("get_compose", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when slug missing")
	}
}

func TestGetCompose_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"GET /api/compose/broken": {404, "not found"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewGetCompose(c)

	req := makeCallToolRequest("get_compose", map[string]any{"slug": "broken"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// deploy_service
// ---------------------------------------------------------------------------

func TestDeployService_deploys_without_env(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/deploy/honeygain": {200, `{"deployed":true}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewDeployService(c)

	req := makeCallToolRequest("deploy_service", map[string]any{"slug": "honeygain"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "honeygain deployed") {
		t.Errorf("expected deploy confirmation, got: %s", text)
	}
}

func TestDeployService_deploys_with_valid_env_json(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/deploy/earnapp": {200, `{"deployed":true}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewDeployService(c)

	req := makeCallToolRequest("deploy_service", map[string]any{
		"slug": "earnapp",
		"env":  `{"TOKEN":"abc123"}`,
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %+v", result)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "earnapp deployed") {
		t.Errorf("expected deploy confirmation, got: %s", text)
	}
}

func TestDeployService_returns_tool_error_for_invalid_env_json(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/deploy/test": {200, `{}`},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewDeployService(c)

	req := makeCallToolRequest("deploy_service", map[string]any{
		"slug": "test",
		"env":  `{invalid json`,
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for invalid env JSON")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "invalid env JSON") {
		t.Errorf("expected error message about invalid JSON, got: %s", text)
	}
}

func TestDeployService_returns_tool_error_when_slug_missing(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(srv.URL, "")
	_, handler := NewDeployService(c)

	req := makeCallToolRequest("deploy_service", nil)
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when slug missing")
	}
}

func TestDeployService_returns_tool_error_on_api_failure(t *testing.T) {
	srv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{
		"POST /api/deploy/broken": {500, "error"},
	})
	c := client.NewClient(srv.URL, "")
	_, handler := NewDeployService(c)

	req := makeCallToolRequest("deploy_service", map[string]any{"slug": "broken"})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true on API failure")
	}
}

// ---------------------------------------------------------------------------
// Tool definition checks
// ---------------------------------------------------------------------------

func TestNewGetEarningsDaily_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewGetEarningsDaily(c)
	if tool.Name != "get_earnings_daily" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewGetEarningsHistory_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewGetEarningsHistory(c)
	if tool.Name != "get_earnings_history" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewGetServiceLogs_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewGetServiceLogs(c)
	if tool.Name != "get_service_logs" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewRestartService_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewRestartService(c)
	if tool.Name != "restart_service" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewStopService_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewStopService(c)
	if tool.Name != "stop_service" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewStartService_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewStartService(c)
	if tool.Name != "start_service" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewRemoveService_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewRemoveService(c)
	if tool.Name != "remove_service" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewTriggerCollection_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewTriggerCollection(c)
	if tool.Name != "trigger_collection" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewGetCompose_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewGetCompose(c)
	if tool.Name != "get_compose" {
		t.Errorf("tool name = %q", tool.Name)
	}
}

func TestNewDeployService_returns_correct_tool_name(t *testing.T) {
	c := client.NewClient("http://localhost", "")
	tool, _ := NewDeployService(c)
	if tool.Name != "deploy_service" {
		t.Errorf("tool name = %q", tool.Name)
	}
}
