package main

import (
	"context"
	"crypto/subtle"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/geiserx/cashpilot-mcp/config"
	"github.com/geiserx/cashpilot-mcp/internal/app"
	"github.com/geiserx/cashpilot-mcp/version"
	"github.com/mark3labs/mcp-go/server"
)

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func bearerAuth(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Printf("CashPilot MCP %s starting…", version.String())

	// Load config & initialise CashPilot client
	cfg := config.LoadCashPilotConfig()
	cp := client.NewClient(cfg.BaseURL, cfg.APIKey)

	s := app.NewServer(cp)

	transport := strings.ToLower(os.Getenv("TRANSPORT"))
	if transport == "stdio" {
		stdioSrv := server.NewStdioServer(s)
		log.Println("CashPilot MCP bridge running on stdio")
		if err := stdioSrv.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
	} else {
		// Stateless: the 2026-07-28 protocol has no sessions, and a session-minting
		// server would advertise only the legacy versions in server/discover.
		httpSrv := server.NewStreamableHTTPServer(s, server.WithStateLess(true))
		addr := os.Getenv("LISTEN_ADDR")
		if addr == "" {
			addr = "127.0.0.1:8081"
		}
		authToken := os.Getenv("MCP_AUTH_TOKEN")
		if authToken == "" && !isLoopbackAddr(addr) {
			log.Fatal("MCP_AUTH_TOKEN is required when LISTEN_ADDR is not loopback")
		}
		if authToken != "" {
			mux := http.NewServeMux()
			mux.Handle("/mcp", bearerAuth(httpSrv, authToken))
			log.Printf("CashPilot MCP bridge listening on %s (auth enabled)", addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Fatalf("server error: %v", err)
			}
		} else {
			log.Printf("CashPilot MCP bridge listening on %s", addr)
			if err := httpSrv.Start(addr); err != nil {
				log.Fatalf("server error: %v", err)
			}
		}
	}
}
