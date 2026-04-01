package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewRestartService builds the tool definition and handler.
func NewRestartService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("restart_service",
		mcp.WithDescription("Restart a deployed service"),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("Service slug identifier"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := req.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := c.RestartService(ctx, slug)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Service %s restarted. Response: %s", slug, string(resp)),
		), nil
	}

	return tool, handler
}
