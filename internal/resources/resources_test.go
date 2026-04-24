package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// mockAPI creates a test HTTP server that maps "METHOD /path" to responses.
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
		w.WriteHeader(404)
		_, _ = w.Write([]byte("not found: " + key))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// readResourceMsg builds a JSON-RPC resources/read message for the given URI.
func readResourceMsg(id int, uri string) []byte {
	return []byte(fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": %d,
		"method": "resources/read",
		"params": {"uri": %q}
	}`, id, uri))
}

// extractTextFromResponse parses a JSONRPCResponse and returns the text from
// the first TextResourceContents in the result.
func extractTextFromResponse(t *testing.T, resp mcp.JSONRPCMessage) string {
	t.Helper()
	jsonResp, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T: %+v", resp, resp)
	}
	b, err := json.Marshal(jsonResp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var result struct {
		Contents []struct {
			URI      string `json:"uri"`
			MIMEType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Contents) == 0 {
		t.Fatal("no contents in response")
	}
	return result.Contents[0].Text
}

// assertIsError checks that the response is a JSON-RPC error.
func assertIsError(t *testing.T, resp mcp.JSONRPCMessage) {
	t.Helper()
	if _, ok := resp.(mcp.JSONRPCError); !ok {
		t.Fatalf("expected JSONRPCError, got %T: %+v", resp, resp)
	}
}

// resourceTestCase describes a single resource registration test.
type resourceTestCase struct {
	name        string
	registerFn  func(*server.MCPServer, *client.Client)
	apiRoute    string
	resourceURI string
	apiBody     string
}

var resourceTestCases = []resourceTestCase{
	{
		name:        "EarningsSummary",
		registerFn:  RegisterEarningsSummary,
		apiRoute:    "GET /api/earnings/summary",
		resourceURI: "cashpilot://earnings/summary",
		apiBody:     `{"total":42.5,"today":3.2}`,
	},
	{
		name:        "EarningsBreakdown",
		registerFn:  RegisterEarningsBreakdown,
		apiRoute:    "GET /api/earnings/breakdown",
		resourceURI: "cashpilot://earnings/breakdown",
		apiBody:     `[{"platform":"honeygain","amount":10}]`,
	},
	{
		name:        "ServicesDeployed",
		registerFn:  RegisterServicesDeployed,
		apiRoute:    "GET /api/services/deployed",
		resourceURI: "cashpilot://services/deployed",
		apiBody:     `[{"slug":"hg","status":"running"}]`,
	},
	{
		name:        "ServicesCatalog",
		registerFn:  RegisterServicesCatalog,
		apiRoute:    "GET /api/services",
		resourceURI: "cashpilot://services/catalog",
		apiBody:     `[{"slug":"earnapp","name":"EarnApp"}]`,
	},
	{
		name:        "FleetSummary",
		registerFn:  RegisterFleetSummary,
		apiRoute:    "GET /api/fleet/summary",
		resourceURI: "cashpilot://fleet/summary",
		apiBody:     `{"workers":3,"services":10}`,
	},
	{
		name:        "Workers",
		registerFn:  RegisterWorkers,
		apiRoute:    "GET /api/workers",
		resourceURI: "cashpilot://workers",
		apiBody:     `[{"name":"watchtower","online":true}]`,
	},
	{
		name:        "HealthScores",
		registerFn:  RegisterHealthScores,
		apiRoute:    "GET /api/health/scores",
		resourceURI: "cashpilot://health/scores",
		apiBody:     `{"overall":95}`,
	},
	{
		name:        "CollectorAlerts",
		registerFn:  RegisterCollectorAlerts,
		apiRoute:    "GET /api/collector-alerts",
		resourceURI: "cashpilot://collector-alerts",
		apiBody:     `[{"type":"error","message":"auth failed"}]`,
	},
}

func TestResource_handler_returns_correct_content_on_success(t *testing.T) {
	for _, tc := range resourceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			apiSrv := mockAPI(t, map[string]struct {
				status int
				body   string
			}{
				tc.apiRoute: {200, tc.apiBody},
			})
			c := client.NewClient(apiSrv.URL, "")
			s := server.NewMCPServer("test", "0.0.0")
			tc.registerFn(s, c)

			msg := readResourceMsg(1, tc.resourceURI)
			resp := s.HandleMessage(context.Background(), msg)

			text := extractTextFromResponse(t, resp)
			if text != tc.apiBody {
				t.Errorf("text = %q, want %q", text, tc.apiBody)
			}
		})
	}
}

func TestResource_handler_returns_error_on_api_failure(t *testing.T) {
	for _, tc := range resourceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			apiSrv := mockAPI(t, map[string]struct {
				status int
				body   string
			}{
				tc.apiRoute: {500, "internal server error"},
			})
			c := client.NewClient(apiSrv.URL, "")
			s := server.NewMCPServer("test", "0.0.0")
			tc.registerFn(s, c)

			msg := readResourceMsg(1, tc.resourceURI)
			resp := s.HandleMessage(context.Background(), msg)

			assertIsError(t, resp)
		})
	}
}

func TestResource_response_contains_correct_uri_and_mimetype(t *testing.T) {
	for _, tc := range resourceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			apiSrv := mockAPI(t, map[string]struct {
				status int
				body   string
			}{
				tc.apiRoute: {200, `{}`},
			})
			c := client.NewClient(apiSrv.URL, "")
			s := server.NewMCPServer("test", "0.0.0")
			tc.registerFn(s, c)

			msg := readResourceMsg(1, tc.resourceURI)
			resp := s.HandleMessage(context.Background(), msg)

			jsonResp, ok := resp.(mcp.JSONRPCResponse)
			if !ok {
				t.Fatalf("expected JSONRPCResponse, got %T", resp)
			}
			b, _ := json.Marshal(jsonResp.Result)
			resultStr := string(b)

			if !strings.Contains(resultStr, tc.resourceURI) {
				t.Errorf("response does not contain URI %q: %s", tc.resourceURI, resultStr)
			}
			if !strings.Contains(resultStr, "application/json") {
				t.Errorf("response does not contain MIME type: %s", resultStr)
			}
		})
	}
}

func TestResource_registration_does_not_panic(t *testing.T) {
	apiSrv := mockAPI(t, map[string]struct {
		status int
		body   string
	}{})
	c := client.NewClient(apiSrv.URL, "")

	for _, tc := range resourceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			s := server.NewMCPServer("test", "0.0.0")
			tc.registerFn(s, c) // Must not panic
		})
	}
}
