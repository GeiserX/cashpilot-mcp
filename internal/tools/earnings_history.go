package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewGetEarningsHistory builds the tool definition and handler.
func NewGetEarningsHistory(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("get_earnings_history",
		mcp.WithDescription("Get earnings history for a given period"),
		mcp.WithString("period",
			mcp.Required(),
			mcp.Description("Time period: week, month, year, or all"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		period, err := req.RequireString("period")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := c.GetEarningsHistory(period)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Earnings history (%s): %s", period, string(resp)),
		), nil
	}

	return tool, handler
}
