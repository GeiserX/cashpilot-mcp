package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewGetServiceLogs builds the tool definition and handler.
func NewGetServiceLogs(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("get_service_logs",
		mcp.WithDescription("Get recent logs for a deployed service"),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("Service slug identifier"),
		),
		mcp.WithNumber("lines",
			mcp.Description("Number of log lines to retrieve (default 50)"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := req.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		lines := 50
		if v, ok := req.GetArguments()["lines"].(float64); ok && v > 0 {
			lines = int(v)
		}
		if lines > 1000 {
			lines = 1000
		}

		resp, err := c.GetServiceLogs(ctx, slug, lines)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Logs for %s (%d lines): %s", slug, lines, string(resp)),
		), nil
	}

	return tool, handler
}
