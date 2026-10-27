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
		append([]mcp.ToolOption{
			mcp.WithDescription("Remove a deployed service from one worker: its containers are deleted and it stops earning. " +
				"Volumes are kept, so node identities and wallets survive and a later deploy_service picks them up. " +
				"Confirm with the user before calling."),
			slugParam("Service to remove."),
			workerParam(),
		}, hints("Remove service", false, true, true, false)...)...,
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := req.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		t, err := target(req, false)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		resp, err := c.RemoveService(ctx, slug, t)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Service %s removed (volumes kept). CashPilot answered: %s", slug, resp)), nil
	}

	return tool, handler
}
