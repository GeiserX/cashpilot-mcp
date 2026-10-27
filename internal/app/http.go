package app

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// MaxRequestBody caps one POST to /mcp. A JSON-RPC request to this server is
// a few kilobytes; the SDK would otherwise buffer any size in memory.
const MaxRequestBody = 1 << 20

// HTTPOptions configures the Streamable HTTP endpoint.
type HTTPOptions struct {
	// Token, when set, is required as "Authorization: Bearer <token>".
	Token string
	// TrustedOrigins are browser origins (scheme://host[:port]) allowed to
	// call the endpoint cross-site. Requests without an Origin header, which
	// is what every non-browser MCP client sends, are not affected.
	TrustedOrigins []string
}

// HTTPHandler returns the /mcp handler: Origin check, then body cap, then
// bearer auth, then the stateless MCP transport.
func HTTPHandler(s *server.MCPServer, opts HTTPOptions) (http.Handler, error) {
	// Stateless: the 2026-07-28 protocol has no sessions, and a session-minting
	// server would advertise only the legacy versions in server/discover.
	var h http.Handler = server.NewStreamableHTTPServer(s,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)
	if opts.Token != "" {
		h = bearerAuth(h, opts.Token)
	}
	h = limitBody(h, MaxRequestBody)

	// The spec requires rejecting a cross-site Origin with 403; the SDK only
	// guards the Host header on loopback binds.
	cop := http.NewCrossOriginProtection()
	for _, o := range opts.TrustedOrigins {
		if err := cop.AddTrustedOrigin(o); err != nil {
			return nil, fmt.Errorf("MCP_ALLOWED_ORIGINS: %w", err)
		}
	}
	h = cop.Handler(h)

	mux := http.NewServeMux()
	mux.Handle("/mcp", h)
	return mux, nil
}

// NewHTTPServer wraps h in an http.Server with timeouts, so a client that
// trickles headers or idles cannot hold connections open forever. There is
// no WriteTimeout: a deploy that pulls an image can legitimately take minutes.
func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
}

// IsLoopbackAddr reports whether a listen address only accepts local
// connections.
func IsLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func bearerAuth(next http.Handler, token string) http.Handler {
	want := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="cashpilot-mcp"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func limitBody(next http.Handler, max int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > max {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, max)
		next.ServeHTTP(w, r)
	})
}
