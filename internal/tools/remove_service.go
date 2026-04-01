package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewRemoveService builds the tool definition and handler.
func NewRemoveService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("remove_service",
		mcp.WithDescription("Remove a deployed service"),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("Service slug to remove"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := req.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := c.RemoveService(ctx, slug)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Service %s removed. Response: %s", slug, string(resp)),
		), nil
	}

	return tool, handler
}
