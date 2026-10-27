package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/geiserx/cashpilot-mcp/config"
	"github.com/geiserx/cashpilot-mcp/internal/app"
	"github.com/geiserx/cashpilot-mcp/version"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Printf("CashPilot MCP %s starting…", version.String())

	cfg := config.LoadCashPilotConfig()
	cp := client.NewClient(cfg.BaseURL, cfg.APIKey)
	s := app.NewServer(cp, app.Options{ReadOnly: cfg.ReadOnly})
	if cfg.ReadOnly {
		log.Println("read-only mode: tools that change the fleet are not registered")
	}

	if strings.ToLower(os.Getenv("TRANSPORT")) == "stdio" {
		log.Println("CashPilot MCP bridge running on stdio")
		if err := server.NewStdioServer(s).Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
		return
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8081"
	}
	token := os.Getenv("MCP_AUTH_TOKEN")
	if token == "" && !app.IsLoopbackAddr(addr) {
		log.Fatal("MCP_AUTH_TOKEN is required when LISTEN_ADDR is not loopback")
	}
	h, err := app.HTTPHandler(s, app.HTTPOptions{
		Token:          token,
		TrustedOrigins: splitList(os.Getenv("MCP_ALLOWED_ORIGINS")),
	})
	if err != nil {
		log.Fatal(err)
	}
	auth := ""
	if token != "" {
		auth = " (auth enabled)"
	}
	log.Printf("CashPilot MCP bridge listening on %s/mcp%s", addr, auth)
	if err := app.NewHTTPServer(addr, h).ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func splitList(v string) []string {
	var out []string
	for _, f := range strings.Split(v, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}
