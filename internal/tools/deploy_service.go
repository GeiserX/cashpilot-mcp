package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewDeployService builds the tool definition and handler.
func NewDeployService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("deploy_service",
		mcp.WithDescription("Deploy a service from the catalog"),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("Service slug to deploy"),
		),
		mcp.WithString("env",
			mcp.Description("Optional JSON string of environment variable key-value pairs"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := req.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Build JSON body: if env is provided, parse it and wrap it.
		body := []byte("{}")
		if envStr, ok := req.GetArguments()["env"].(string); ok && envStr != "" {
			var envMap map[string]string
			if err := json.Unmarshal([]byte(envStr), &envMap); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid env JSON: %v", err)), nil
			}
			body, err = json.Marshal(map[string]any{"env": envMap})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("marshal deploy body: %v", err)), nil
			}
		}

		resp, err := c.DeployService(ctx, slug, body)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Service %s deployed. Response: %s", slug, string(resp)),
		), nil
	}

	return tool, handler
}
