package app

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/server"
)

// stdioPeer drives the real stdio transport in-process: requests go in as
// newline-delimited JSON, exactly as a client process would write them.
type stdioPeer struct {
	t     *testing.T
	in    *io.PipeWriter
	lines chan []byte
}

func newStdioPeer(t *testing.T, s *server.MCPServer) *stdioPeer {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.NewStdioServer(s).Listen(ctx, inR, outW)
		_ = outW.Close()
	}()
	t.Cleanup(func() {
		cancel()
		_ = inW.Close()
		_ = outR.Close()
		<-done
	})
	lines := make(chan []byte, 16)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(outR)
		sc.Buffer(make([]byte, 0, 1<<20), 8<<20)
		for sc.Scan() {
			lines <- append([]byte(nil), sc.Bytes()...)
		}
	}()
	return &stdioPeer{t: t, in: inW, lines: lines}
}

type rpcReply struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// call sends one request and returns the reply carrying the same id,
// skipping any notification the server writes in between.
func (p *stdioPeer) call(id int, method string, params any) rpcReply {
	p.t.Helper()
	p.notify(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	want, _ := json.Marshal(id)
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			p.t.Fatalf("%s: no reply within 10s", method)
		case line, ok := <-p.lines:
			if !ok {
				p.t.Fatalf("%s: server closed stdout", method)
			}
			var r rpcReply
			if err := json.Unmarshal(line, &r); err != nil {
				p.t.Fatalf("%s: reply is not JSON: %s", method, line)
			}
			if string(r.ID) == string(want) {
				return r
			}
		}
	}
}

func (p *stdioPeer) notify(msg map[string]any) {
	p.t.Helper()
	b, err := json.Marshal(msg)
	if err != nil {
		p.t.Fatal(err)
	}
	if _, err := p.in.Write(append(b, '\n')); err != nil {
		p.t.Fatalf("write %v: %v", msg["method"], err)
	}
}

// modernMeta is the per-request _meta a 2026-07-28 client sends in place of
// the initialize handshake.
func modernMeta() map[string]any {
	return map[string]any{"_meta": map[string]any{
		"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "smoke", "version": "0"},
	}}
}

func testServer() *server.MCPServer {
	return NewServer(client.NewClient("http://127.0.0.1:1", ""), Options{})
}

func TestProtocol_modern_discover_lists_2026_07_28(t *testing.T) {
	p := newStdioPeer(t, testServer())
	r := p.call(1, "server/discover", modernMeta())
	if r.Error != nil {
		t.Fatalf("server/discover failed: %d %s", r.Error.Code, r.Error.Message)
	}
	var res struct {
		SupportedVersions []string `json:"supportedVersions"`
	}
	if err := json.Unmarshal(r.Result, &res); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"2026-07-28", "2025-11-25", "2024-11-05"} {
		if !slices.Contains(res.SupportedVersions, v) {
			t.Errorf("supportedVersions %v lacks %s", res.SupportedVersions, v)
		}
	}
}

func TestProtocol_modern_tools_list_is_complete_and_cacheable(t *testing.T) {
	p := newStdioPeer(t, testServer())
	r := p.call(1, "tools/list", modernMeta())
	if r.Error != nil {
		t.Fatalf("tools/list failed: %d %s", r.Error.Code, r.Error.Message)
	}
	var res map[string]json.RawMessage
	if err := json.Unmarshal(r.Result, &res); err != nil {
		t.Fatal(err)
	}
	if string(res["resultType"]) != `"complete"` {
		t.Errorf("resultType = %s, want \"complete\"", res["resultType"])
	}
	for _, k := range []string{"ttlMs", "cacheScope", "tools"} {
		if _, ok := res[k]; !ok {
			t.Errorf("modern tools/list result lacks %q: %s", k, r.Result)
		}
	}
}

func TestProtocol_modern_ping_is_removed(t *testing.T) {
	p := newStdioPeer(t, testServer())
	r := p.call(1, "ping", modernMeta())
	if r.Error == nil || r.Error.Code != -32601 {
		t.Fatalf("modern ping should fail with -32601, got %+v / %s", r.Error, r.Result)
	}
}

func TestProtocol_legacy_initialize_then_tools_list(t *testing.T) {
	p := newStdioPeer(t, testServer())
	r := p.call(1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "legacy", "version": "0"},
	})
	if r.Error != nil {
		t.Fatalf("initialize failed: %d %s", r.Error.Code, r.Error.Message)
	}
	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(r.Result, &init); err != nil {
		t.Fatal(err)
	}
	if init.ProtocolVersion != "2025-11-25" {
		t.Errorf("initialize negotiated %q, want 2025-11-25", init.ProtocolVersion)
	}
	p.notify(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})

	r = p.call(2, "tools/list", map[string]any{})
	if r.Error != nil {
		t.Fatalf("legacy tools/list failed: %d %s", r.Error.Code, r.Error.Message)
	}
	var list struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(r.Result, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) == 0 {
		t.Fatalf("legacy tools/list returned no tools: %s", r.Result)
	}

	// A legacy ping still answers: the removal applies to 2026-07-28 only.
	if r := p.call(3, "ping", map[string]any{}); r.Error != nil {
		t.Errorf("legacy ping failed: %d %s", r.Error.Code, r.Error.Message)
	}
}
