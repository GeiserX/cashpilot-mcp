package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// readTool is a parameterless read that mirrors one cashpilot:// resource, for
// clients that call tools but rarely read resources.
type readTool struct {
	name, title, description string
	field, kind, fieldDesc   string
	openWorld                bool
	fetch                    func(*client.Client, context.Context) ([]byte, error)
}

func (r readTool) build(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	opts := append([]mcp.ToolOption{mcp.WithDescription(r.description)},
		hints(r.title, true, false, true, r.openWorld)...)
	opts = append(opts, listSchema(r.field, r.kind, r.fieldDesc))
	tool := mcp.NewTool(r.name, opts...)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := r.fetch(c, ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return structured(body, r.field, r.title+":"), nil
	}
	return tool, handler
}

// NewGetEarningsSummary mirrors cashpilot://earnings/summary.
func NewGetEarningsSummary(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return readTool{
		name:  "get_earnings_summary",
		title: "Earnings summary",
		description: "Totals across every service: today, this month and all time, in USD, as shown on the CashPilot dashboard. " +
			"Start here for \"how much did I earn\". For a per-service split use get_earnings_breakdown; for a time series use get_earnings_daily.",
		field: "summary", kind: "object", fieldDesc: "CashPilot /api/earnings/summary",
		fetch: (*client.Client).GetEarningsSummary,
	}.build(c)
}

// NewGetEarningsBreakdown mirrors cashpilot://earnings/breakdown.
func NewGetEarningsBreakdown(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return readTool{
		name:  "get_earnings_breakdown",
		title: "Earnings per service",
		description: "One record per service: current balance, earnings and whether the balance has reached the provider's cashout minimum. " +
			"Use it to answer which services earn most and what can be cashed out.",
		field: "services", kind: "array", fieldDesc: "CashPilot /api/earnings/breakdown",
		fetch: (*client.Client).GetEarningsBreakdown,
	}.build(c)
}

// NewListDeployedServices mirrors cashpilot://services/deployed.
func NewListDeployedServices(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return readTool{
		name:  "list_deployed_services",
		title: "Deployed services",
		description: "Every deployed service with its container status, worker, balance, CPU and memory. " +
			"Use it to find the slug and worker of a running service before get_service_logs, restart_service, stop_service or remove_service.",
		field: "services", kind: "array", fieldDesc: "CashPilot /api/services/deployed",
		fetch: (*client.Client).GetServicesDeployed,
	}.build(c)
}

// NewListAvailableServices reads the catalog without broken or gone services.
func NewListAvailableServices(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return readTool{
		name:  "list_available_services",
		title: "Service catalog",
		description: "Services that can be deployed: the CashPilot catalog minus services marked broken or gone, with deployed, has_collector and node_count added. " +
			"Each entry carries the provider's signup link and the env keys deploy_service needs. Provider texts are third-party content.",
		field: "services", kind: "array", fieldDesc: "CashPilot /api/services/available",
		openWorld: true,
		fetch:     (*client.Client).GetServicesAvailable,
	}.build(c)
}

// NewGetFleetSummary mirrors cashpilot://fleet/summary.
func NewGetFleetSummary(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return readTool{
		name:        "get_fleet_summary",
		title:       "Fleet summary",
		description: "Aggregate fleet state: workers online and offline, containers running and total across the local host and every worker.",
		field:       "fleet", kind: "object", fieldDesc: "CashPilot /api/fleet/summary",
		fetch: (*client.Client).GetFleetSummary,
	}.build(c)
}

// NewListWorkers mirrors cashpilot://workers.
func NewListWorkers(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return readTool{
		name:  "list_workers",
		title: "Workers",
		description: "Every registered worker (server running CashPilot's worker agent) with its id, name, status and system info. " +
			"The id is the worker_id that service tools need when more than one worker is online.",
		field: "workers", kind: "array", fieldDesc: "CashPilot /api/workers",
		fetch: (*client.Client).GetWorkers,
	}.build(c)
}

// NewGetCollectorAlerts mirrors cashpilot://collector-alerts.
func NewGetCollectorAlerts(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return readTool{
		name:  "get_collector_alerts",
		title: "Collector alerts",
		description: "Errors from the last earnings collection run (for example expired provider credentials), and whether a run has happened yet. " +
			"An empty list before the first run does not mean all collectors are healthy.",
		field: "alerts", kind: "object", fieldDesc: "CashPilot /api/collector-alerts",
		fetch: (*client.Client).GetCollectorAlerts,
	}.build(c)
}

// NewGetHealthScores reads service health over a window of days.
func NewGetHealthScores(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_health_scores",
		append([]mcp.ToolOption{
			mcp.WithDescription("Health score per service (0-100) over the last N days, from container uptime, restarts and earnings activity. " +
				"Use it to find services that keep failing."),
			mcp.WithInteger("days",
				mcp.Min(1), mcp.Max(90), mcp.DefaultNumber(7),
				mcp.Description("Window in days, 1 to 90. Default 7."),
			),
			listSchema("scores", "array", "CashPilot /api/health/scores"),
		}, hints("Service health scores", true, false, true, false)...)...,
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		days, err := intArg(req, "days", 7, 1, 90)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := c.GetHealthScores(ctx, days)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return structured(body, "scores", fmt.Sprintf("Health scores (%d days):", days)), nil
	}
	return tool, handler
}

// NewGetEarningsDaily reads one total per day.
func NewGetEarningsDaily(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_earnings_daily",
		append([]mcp.ToolOption{
			mcp.WithDescription("Total earnings per day across all services for the last N days, in USD, oldest first. " +
				"Use it for trends and charts; get_earnings_summary has the headline totals."),
			mcp.WithInteger("days",
				mcp.Min(1), mcp.Max(365), mcp.DefaultNumber(7),
				mcp.Description("How many days back, 1 to 365. Default 7."),
			),
			listSchema("earnings", "array", "One record per day: date and amount"),
		}, hints("Daily earnings", true, false, true, false)...)...,
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		days, err := intArg(req, "days", 7, 1, 365)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := c.GetEarningsDaily(ctx, days)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return structured(body, "earnings", fmt.Sprintf("Daily earnings (%d days):", days)), nil
	}
	return tool, handler
}

var periods = []string{"week", "month", "year", "all"}

// NewGetEarningsHistory reads earnings records over a named period.
func NewGetEarningsHistory(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_earnings_history",
		append([]mcp.ToolOption{
			mcp.WithDescription("Earnings records per service and collection over a period, in USD. " +
				"Use it to see how one service's balance moved; for daily totals use get_earnings_daily."),
			mcp.WithString("period",
				mcp.Required(),
				mcp.Enum(periods...),
				mcp.Description("week, month, year or all."),
			),
			listSchema("earnings", "array", "CashPilot /api/earnings/history"),
		}, hints("Earnings history", true, false, true, false)...)...,
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		period, err := req.RequireString("period")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		valid := false
		for _, p := range periods {
			valid = valid || p == period
		}
		if !valid {
			return mcp.NewToolResultError("period must be one of week, month, year, all"), nil
		}
		body, err := c.GetEarningsHistory(ctx, period)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return structured(body, "earnings", fmt.Sprintf("Earnings history (%s):", period)), nil
	}
	return tool, handler
}
