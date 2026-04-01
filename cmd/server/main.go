package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/geiserx/cashpilot-mcp/config"
	"github.com/geiserx/cashpilot-mcp/internal/resources"
	"github.com/geiserx/cashpilot-mcp/internal/tools"
	"github.com/geiserx/cashpilot-mcp/version"
	"github.com/mark3labs/mcp-go/server"
)

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
		log.Println("CashPilot MCP bridge listening on :8081")
		if err := httpSrv.Start(":8081"); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}
}
