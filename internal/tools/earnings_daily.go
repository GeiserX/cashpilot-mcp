package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewGetEarningsDaily builds the tool definition and handler.
func NewGetEarningsDaily(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("get_earnings_daily",
		mcp.WithDescription("Get daily earnings for the last N days"),
		mcp.WithNumber("days",
			mcp.Description("Number of days to retrieve (default 7, max 365)"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		days := 7
		if v, ok := req.GetArguments()["days"].(float64); ok && v > 0 {
			days = int(v)
			if days > 365 {
				days = 365
			}
		}

		resp, err := c.GetEarningsDaily(ctx, days)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Daily earnings (%d days): %s", days, string(resp)),
		), nil
	}

	return tool, handler
}
