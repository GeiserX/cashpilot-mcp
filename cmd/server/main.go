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
	"github.com/geiserx/cashpilot-mcp/internal/resources"
	"github.com/geiserx/cashpilot-mcp/internal/tools"
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

	// Create MCP server
	s := server.NewMCPServer(
		"CashPilot MCP Bridge",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	// -----------------------------------------------------------------
	// RESOURCES
	// -----------------------------------------------------------------

	// cashpilot://earnings/summary
	resources.RegisterEarningsSummary(s, cp)

	// cashpilot://earnings/breakdown
	resources.RegisterEarningsBreakdown(s, cp)

	// cashpilot://services/deployed
	resources.RegisterServicesDeployed(s, cp)

	// cashpilot://services/catalog
	resources.RegisterServicesCatalog(s, cp)

	// cashpilot://fleet/summary
	resources.RegisterFleetSummary(s, cp)

	// cashpilot://workers
	resources.RegisterWorkers(s, cp)

	// cashpilot://health/scores
	resources.RegisterHealthScores(s, cp)

	// cashpilot://collector-alerts
	resources.RegisterCollectorAlerts(s, cp)

	// -----------------------------------------------------------------
	// TOOLS
	// -----------------------------------------------------------------

	// get_earnings_daily
	tool, handler := tools.NewGetEarningsDaily(cp)
	s.AddTool(tool, handler)

	// get_earnings_history
	tool, handler = tools.NewGetEarningsHistory(cp)
	s.AddTool(tool, handler)

	// get_service_logs
	tool, handler = tools.NewGetServiceLogs(cp)
	s.AddTool(tool, handler)

	// restart_service
	tool, handler = tools.NewRestartService(cp)
	s.AddTool(tool, handler)

	// stop_service
	tool, handler = tools.NewStopService(cp)
	s.AddTool(tool, handler)

	// start_service
	tool, handler = tools.NewStartService(cp)
	s.AddTool(tool, handler)

	// deploy_service
	tool, handler = tools.NewDeployService(cp)
	s.AddTool(tool, handler)

	// remove_service
	tool, handler = tools.NewRemoveService(cp)
	s.AddTool(tool, handler)

	// trigger_collection
	tool, handler = tools.NewTriggerCollection(cp)
	s.AddTool(tool, handler)

	// get_compose
	tool, handler = tools.NewGetCompose(cp)
	s.AddTool(tool, handler)

	transport := strings.ToLower(os.Getenv("TRANSPORT"))
	if transport == "stdio" {
		stdioSrv := server.NewStdioServer(s)
		log.Println("CashPilot MCP bridge running on stdio")
		if err := stdioSrv.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
	} else {
		httpSrv := server.NewStreamableHTTPServer(s)
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
