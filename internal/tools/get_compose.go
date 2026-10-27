package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var arches = []string{"amd64", "arm64", "arm"}

// NewGetCompose builds the tool definition and handler.
func NewGetCompose(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_compose",
		append([]mcp.ToolOption{
			mcp.WithDescription("A docker-compose.yml (YAML text) that runs one catalog service outside CashPilot, with placeholders for its credentials. " +
				"Nothing is deployed; use deploy_service for that."),
			slugParam("Service to export."),
			mcp.WithString("arch",
				mcp.Enum(arches...),
				mcp.Description("CPU architecture of the target machine: amd64, arm64 (Raspberry Pi 4/5, Apple silicon) or arm (32-bit). Default: the catalog's default image, which is amd64."),
			),
		}, hints("Docker Compose export", true, false, true, false)...)...,
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := req.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		arch := req.GetString("arch", "")
		if arch != "" && arch != "amd64" && arch != "arm64" && arch != "arm" {
			return mcp.NewToolResultError("arch must be amd64, arm64 or arm"), nil
		}
		resp, err := c.GetCompose(ctx, slug, arch)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("# docker-compose.yml for %s\n%s", slug, resp)), nil
	}

	return tool, handler
}
