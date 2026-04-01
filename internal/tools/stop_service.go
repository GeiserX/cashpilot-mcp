package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewStopService builds the tool definition and handler.
func NewStopService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("stop_service",
		mcp.WithDescription("Stop a deployed service"),
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

		resp, err := c.StopService(slug)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Service %s stopped. Response: %s", slug, string(resp)),
		), nil
	}

	return tool, handler
}
