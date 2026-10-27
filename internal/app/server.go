// Package app assembles the CashPilot MCP server: it registers the resources
// and tools on an MCPServer so main and the tests build the exact same thing.
package app

import (
	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/geiserx/cashpilot-mcp/internal/resources"
	"github.com/geiserx/cashpilot-mcp/internal/tools"
	"github.com/geiserx/cashpilot-mcp/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer returns the MCP server with every resource and tool registered.
func NewServer(cp *client.Client) *server.MCPServer {
	s := server.NewMCPServer(
		"CashPilot MCP Bridge",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	resources.RegisterEarningsSummary(s, cp)
	resources.RegisterEarningsBreakdown(s, cp)
	resources.RegisterServicesDeployed(s, cp)
	resources.RegisterServicesCatalog(s, cp)
	resources.RegisterFleetSummary(s, cp)
	resources.RegisterWorkers(s, cp)
	resources.RegisterHealthScores(s, cp)
	resources.RegisterCollectorAlerts(s, cp)

	for _, build := range []func(*client.Client) (mcp.Tool, server.ToolHandlerFunc){
		tools.NewGetEarningsDaily,
		tools.NewGetEarningsHistory,
		tools.NewGetServiceLogs,
		tools.NewRestartService,
		tools.NewStopService,
		tools.NewStartService,
		tools.NewDeployService,
		tools.NewRemoveService,
		tools.NewTriggerCollection,
		tools.NewGetCompose,
	} {
		tool, handler := build(cp)
		s.AddTool(tool, handler)
	}
	return s
}
