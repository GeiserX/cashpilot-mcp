package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewGetCompose builds the tool definition and handler.
func NewGetCompose(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("get_compose",
		mcp.WithDescription("Get the Docker Compose definition for a service"),
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

		resp, err := c.GetCompose(ctx, slug)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Compose for %s: %s", slug, string(resp)),
		), nil
	}

	return tool, handler
}
