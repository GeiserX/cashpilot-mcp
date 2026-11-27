package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestGetEarningsDaily_rejects_days_above_365(t *testing.T) {
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
	if !result.IsError || !strings.Contains(text, "between 1 and 365") {
		t.Errorf("expected an out-of-range error, got: %s", text)
	}
}

func TestGetEarningsDaily_rejects_negative_days(t *testing.T) {
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
	if !result.IsError || !strings.Contains(text, "between 1 and 365") {
		t.Errorf("expected an out-of-range error, got: %s", text)
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

func TestGetServiceLogs_rejects_lines_above_1000(t *testing.T) {
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
	if !result.IsError || !strings.Contains(text, "between 1 and 1000") {
		t.Errorf("expected an out-of-range error, got: %s", text)
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
	if !strings.Contains(text, "earnapp: restart requested") {
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
	if !strings.Contains(text, "peer2profit: stop requested") {
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
	if !strings.Contains(text, "repocket: start requested") {
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
	if !strings.Contains(text, "Collection started") {
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
	if !strings.Contains(text, "docker-compose.yml for honeygain") {
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
		"env":  map[string]any{"TOKEN": "abc123"},
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
		"env":  `{"TOKEN":"abc123"}`, // a JSON string, not an object
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for env passed as a string")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "env must be an object") {
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

// The web dashboard's 409 for a redeploy (CashPilot app/main.py _ask_deploy_confirmation).
const confirmDeployBody = `{"detail":{"error":"confirm_deploy_targets",` +
	`"message":"Replace these containers for honeygain on box-1?\n\nInstance 1 (0123456789ab)",` +
	`"expected_containers":{"p3":"abcdef0123456789","2":"0123456789abcdef","1":"fedcba9876543210","4":"none"}}}`

// recordingAPI answers POST /api/deploy/honeygain with each status and body in
// turn and keeps every request body.
func recordingAPI(t *testing.T, answers ...[2]string) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		bodies = append(bodies, b)
		if r.Method+" "+r.URL.Path != "POST /api/deploy/honeygain" || len(bodies) > len(answers) {
			w.WriteHeader(500)
			return
		}
		a := answers[len(bodies)-1]
		if a[0] == "409" {
			w.WriteHeader(409)
		}
		_, _ = w.Write([]byte(a[1]))
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}

func TestDeployService_runs_the_3_0_replace_confirmation(t *testing.T) {
	srv, bodies := recordingAPI(t,
		[2]string{"409", confirmDeployBody},
		[2]string{"200", `{"status":"deployed"}`},
	)
	_, handler := NewDeployService(client.NewClient(srv.URL, ""))

	// First call: CashPilot lists what runs and replaces nothing.
	first, err := handler(context.Background(), makeCallToolRequest("deploy_service",
		map[string]any{"slug": "honeygain", "instances": 4}))
	if err != nil {
		t.Fatal(err)
	}
	if !first.IsError {
		t.Fatal("a refused deploy must be a tool error")
	}
	text := first.Content[0].(mcp.TextContent).Text
	for _, want := range []string{
		"honeygain already runs on this worker: instance 1 (container fedcba987654), instance 2 (container 0123456789ab), proxied copy 3 (container abcdef012345).",
		"CashPilot has not replaced them.",
		"Ask the owner",
		`replace_containers set to {"1":"fedcba9876543210","2":"0123456789abcdef","4":"none","p3":"abcdef0123456789"}`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "cannot") || strings.Contains(text, "container none") {
		t.Errorf("text = %s", text)
	}
	b := (*bodies)[0]
	if b["replace_existing"] != true || b["expected_containers"] != nil {
		t.Errorf("first call must ask CashPilot for the targets: %v", b)
	}

	// Second call, after the owner agreed: the map goes back unchanged.
	second, err := handler(context.Background(), makeCallToolRequest("deploy_service", map[string]any{
		"slug": "honeygain", "instances": 4,
		"replace_containers": map[string]any{"1": "fedcba9876543210", "2": "0123456789abcdef", "4": "none", "p3": "abcdef0123456789"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if second.IsError {
		t.Fatalf("confirmed deploy failed: %v", second.Content)
	}
	b = (*bodies)[1]
	want := map[string]any{"1": "fedcba9876543210", "2": "0123456789abcdef", "4": "none", "p3": "abcdef0123456789"}
	if got, _ := b["expected_containers"].(map[string]any); !reflect.DeepEqual(got, want) || b["replace_existing"] != true || b["instances"] != float64(4) {
		t.Errorf("confirmed body = %v", b)
	}
}

func TestDeployService_old_worker_warning_survives_many_copies(t *testing.T) {
	// The message _ask_deploy_confirmation writes for an old worker running six
	// copies: the warning comes last, past the usual quote cap.
	targets := map[string]string{}
	lines := []string{}
	for n := 1; n <= 6; n++ {
		id := fmt.Sprintf("%d123456789abcdef", n)
		targets[fmt.Sprint(n)] = id
		lines = append(lines, fmt.Sprintf("Instance %d (%s)", n, id[:12]))
	}
	msg := "Replace these containers for honeygain on home-server-in-the-garage?\n\n" + strings.Join(lines, "\n") +
		"\n\nThese are the container IDs this worker last reported. It could replace a different container." +
		"\n\nUpdate this worker so it can verify the container ID itself."
	detail, _ := json.Marshal(map[string]any{"detail": map[string]any{
		"error": "confirm_deploy_targets", "message": msg, "expected_containers": targets,
	}})
	srv, _ := recordingAPI(t, [2]string{"409", string(detail)})
	_, handler := NewDeployService(client.NewClient(srv.URL, ""))

	result, err := handler(context.Background(), makeCallToolRequest("deploy_service",
		map[string]any{"slug": "honeygain", "instances": 6}))
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	for _, want := range []string{
		"This worker is old: the ids are the ones it last reported, so it could replace a different container. Tell the owner this before asking.",
		"It could replace a different container. Update this worker so it can verify the container ID itself.",
	} {
		if !result.IsError || !strings.Contains(text, want) {
			t.Errorf("IsError = %v, text lacks %q:\n%s", result.IsError, want, text)
		}
	}
}

func TestDeployService_target_changed_asks_for_a_fresh_look(t *testing.T) {
	changed := `{"detail":{"error":"deploy_target_changed",` +
		`"message":"The container changed or needs approval. Review it and use Redeploy.","current_container":"0123456789abcdef"}}`
	srv, _ := recordingAPI(t, [2]string{"409", changed})
	_, handler := NewDeployService(client.NewClient(srv.URL, ""))

	result, err := handler(context.Background(), makeCallToolRequest("deploy_service", map[string]any{
		"slug": "honeygain", "replace_containers": map[string]any{"1": "fedcba9876543210"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	for _, want := range []string{
		"now container 0123456789ab",
		"CashPilot stopped at that container. Copies it reached before it may already run the new settings.",
		"Call deploy_service again without replace_containers",
	} {
		if !result.IsError || !strings.Contains(text, want) {
			t.Errorf("IsError = %v, text lacks %q:\n%s", result.IsError, want, text)
		}
	}
}

func TestDeployService_refuses_a_bad_replace_containers(t *testing.T) {
	for name, rc := range map[string]any{
		"not an object":  "1=abc",
		"empty":          map[string]any{},
		"non-string id":  map[string]any{"1": 7},
		"short id":       map[string]any{"1": "abc"},
		"bad copy key":   map[string]any{"x1": "fedcba9876543210"},
		"upper-case hex": map[string]any{"1": "FEDCBA9876543210"},
	} {
		t.Run(name, func(t *testing.T) {
			srv, bodies := recordingAPI(t)
			_, handler := NewDeployService(client.NewClient(srv.URL, ""))
			result, err := handler(context.Background(), makeCallToolRequest("deploy_service",
				map[string]any{"slug": "honeygain", "replace_containers": rc}))
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError || len(*bodies) != 0 {
				t.Errorf("IsError = %v, requests = %d; want a tool error and no request", result.IsError, len(*bodies))
			}
		})
	}
}

func TestDeployService_passes_other_conflicts_through(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"409 with a plain detail", 409,
			`{"detail":"A container already exists. Update the worker so CashPilot can identify it before replacing it."}`,
			"CashPilot answered 409: A container already exists. Update the worker so CashPilot can identify it before replacing it."},
		// Only a 409 carries the confirmation; the same code on another status passes through.
		{"400 with a refusal code", 400,
			`{"detail":{"error":"deploy_target_changed","message":"refused","current_container":"0123456789abcdef"}}`,
			"CashPilot answered 400: refused (deploy_target_changed)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := mockAPI(t, map[string]struct {
				status int
				body   string
			}{
				"POST /api/deploy/honeygain": {tc.status, tc.body},
			})
			_, handler := NewDeployService(client.NewClient(srv.URL, ""))

			result, err := handler(context.Background(), makeCallToolRequest("deploy_service", map[string]any{"slug": "honeygain"}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			text := result.Content[0].(mcp.TextContent).Text
			if !result.IsError || text != tc.want {
				t.Errorf("IsError = %v, text = %q; want %q", result.IsError, text, tc.want)
			}
		})
	}
}

func TestDeployService_description_states_the_3_0_confirmation(t *testing.T) {
	tool, _ := NewDeployService(client.NewClient("http://localhost", ""))
	for _, bad := range []string{"redeploys it with the new settings;", "cannot confirm"} {
		if strings.Contains(tool.Description, bad) {
			t.Errorf("description says %q: %s", bad, tool.Description)
		}
	}
	for _, want := range []string{"CashPilot 3.0", "only after the owner agrees", "replace_containers"} {
		if !strings.Contains(tool.Description, want) {
			t.Errorf("description lacks %q: %s", want, tool.Description)
		}
	}
	if _, ok := tool.InputSchema.Properties["replace_containers"]; !ok {
		t.Error("deploy_service has no replace_containers parameter")
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

func TestListSchema_is_valid_json_whatever_the_field(t *testing.T) {
	for _, field := range []string{"earnings", `we"ird`, `x":{},"y`} {
		var tool mcp.Tool
		listSchema(field, "array", `a "quoted" description`)(&tool)
		var s struct {
			Type       string                    `json:"type"`
			Properties map[string]map[string]any `json:"properties"`
			Required   []string                  `json:"required"`
		}
		if err := json.Unmarshal(tool.RawOutputSchema, &s); err != nil {
			t.Fatalf("field %q: outputSchema is not JSON: %v\n%s", field, err, tool.RawOutputSchema)
		}
		if s.Type != "object" || len(s.Properties) != 1 || s.Properties[field]["type"] != "array" ||
			len(s.Required) != 1 || s.Required[0] != field {
			t.Errorf("field %q: outputSchema = %s", field, tool.RawOutputSchema)
		}
	}
}
