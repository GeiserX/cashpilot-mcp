package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewTriggerCollection builds the tool definition and handler.
func NewTriggerCollection(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("trigger_collection",
		append([]mcp.ToolOption{
			mcp.WithDescription("Start an earnings collection now: CashPilot asks every provider for the current balances. " +
				"It runs in the background and returns at once; read get_earnings_summary or get_collector_alerts a minute later."),
		}, hints("Collect earnings now", false, false, true, true)...)...,
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := c.TriggerCollection(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Collection started in the background. CashPilot answered: %s", resp)), nil
	}

	return tool, handler
}
