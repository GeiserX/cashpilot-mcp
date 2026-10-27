package app

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const modernToolsList = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{` +
	`"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
	`"io.modelcontextprotocol/clientCapabilities":{},` +
	`"io.modelcontextprotocol/clientInfo":{"name":"t","version":"0"}}}}`

func newHTTP(t *testing.T, opts HTTPOptions) *httptest.Server {
	t.Helper()
	h, err := HTTPHandler(testServer(), opts)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, url string, body []byte, hdr map[string]string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/mcp", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2026-07-28")
	req.Header.Set("Mcp-Method", "tools/list")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestHTTP_modern_tools_list_is_served_statelessly(t *testing.T) {
	srv := newHTTP(t, HTTPOptions{})
	code, body := post(t, srv.URL, []byte(modernToolsList), nil)
	if code != http.StatusOK {
		t.Fatalf("status %d: %s", code, body)
	}
	if !strings.Contains(body, `"resultType":"complete"`) {
		t.Errorf("modern result lacks resultType complete: %s", body)
	}
}

func TestHTTP_legacy_initialize_still_works(t *testing.T) {
	srv := newHTTP(t, HTTPOptions{})
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"l","version":"0"}}}`
	code, body := post(t, srv.URL, []byte(init), map[string]string{"MCP-Protocol-Version": "", "Mcp-Method": ""})
	if code != http.StatusOK || !strings.Contains(body, `"protocolVersion":"2025-11-25"`) {
		t.Fatalf("legacy initialize: status %d: %s", code, body)
	}
}

func TestHTTP_cross_site_origin_is_rejected(t *testing.T) {
	srv := newHTTP(t, HTTPOptions{TrustedOrigins: []string{"https://ok.example"}})
	if code, body := post(t, srv.URL, []byte(modernToolsList), map[string]string{"Origin": "https://evil.example"}); code != http.StatusForbidden {
		t.Errorf("evil Origin: status %d, want 403: %s", code, body)
	}
	if code, body := post(t, srv.URL, []byte(modernToolsList), map[string]string{"Origin": "https://ok.example"}); code != http.StatusOK {
		t.Errorf("trusted Origin: status %d, want 200: %s", code, body)
	}
}

func TestHTTP_rejects_bad_trusted_origin(t *testing.T) {
	if _, err := HTTPHandler(testServer(), HTTPOptions{TrustedOrigins: []string{"not a url"}}); err == nil {
		t.Fatal("expected an error for a malformed origin")
	}
}

func TestHTTP_oversized_body_is_refused(t *testing.T) {
	srv := newHTTP(t, HTTPOptions{})
	big := bytes.Repeat([]byte(" "), MaxRequestBody+1)
	if code, _ := post(t, srv.URL, big, nil); code != http.StatusRequestEntityTooLarge {
		t.Errorf("status %d, want 413", code)
	}
}

func TestHTTP_bearer_token_is_required_with_scheme(t *testing.T) {
	srv := newHTTP(t, HTTPOptions{Token: "s3cret"})
	cases := map[string]int{
		"":              http.StatusUnauthorized,
		"s3cret":        http.StatusUnauthorized, // no "Bearer " scheme
		"Bearer wrong":  http.StatusUnauthorized,
		"Bearer s3cret": http.StatusOK,
	}
	for auth, want := range cases {
		code, body := post(t, srv.URL, []byte(modernToolsList), map[string]string{"Authorization": auth})
		if code != want {
			t.Errorf("Authorization %q: status %d, want %d: %s", auth, code, want, body)
		}
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:8081": true,
		"localhost:8081": true,
		"[::1]:8081":     true,
		":8081":          false, // empty host binds every interface
		"0.0.0.0:8081":   false,
		"10.0.0.5:8081":  false,
		"garbage":        false,
	}
	for addr, want := range cases {
		if got := IsLoopbackAddr(addr); got != want {
			t.Errorf("IsLoopbackAddr(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestNewHTTPServer_sets_timeouts(t *testing.T) {
	s := NewHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if s.ReadHeaderTimeout == 0 || s.ReadTimeout == 0 || s.IdleTimeout == 0 || s.MaxHeaderBytes == 0 {
		t.Fatalf("missing timeout or header cap: %+v", s)
	}
}
