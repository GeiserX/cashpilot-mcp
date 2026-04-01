package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewStartService builds the tool definition and handler.
func NewStartService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("start_service",
		mcp.WithDescription("Start a stopped service"),
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

		resp, err := c.StartService(ctx, slug)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Service %s started. Response: %s", slug, string(resp)),
		), nil
	}

	return tool, handler
}
