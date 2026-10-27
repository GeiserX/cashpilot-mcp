package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type lifecycleTool struct {
	name, title, description, verb string
	destructive                    bool
	call                           func(*client.Client, context.Context, string, client.Target) ([]byte, error)
}

func (l lifecycleTool) build(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription(l.description),
		slugParam("Service to " + l.verb + "."),
		workerParam(),
		instanceParam(),
	}, hints(l.title, false, l.destructive, true, false)...)
	tool := mcp.NewTool(l.name, opts...)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := req.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		t, err := target(req, true)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		resp, err := l.call(c, ctx, slug, t)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Service %s: %s requested. CashPilot answered: %s", slug, l.verb, resp)), nil
	}
	return tool, handler
}

// NewStartService starts a stopped container.
func NewStartService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return lifecycleTool{
		name: "start_service", title: "Start service", verb: "start",
		description: "Start a deployed service's stopped container on one worker. Starting a running service changes nothing.",
		call:        (*client.Client).StartService,
	}.build(c)
}

// NewRestartService restarts a container.
func NewRestartService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return lifecycleTool{
		name: "restart_service", title: "Restart service", verb: "restart",
		description: "Restart a deployed service's container on one worker, keeping its configuration and data. " +
			"It stops earning for the few seconds the container needs to come back.",
		call: (*client.Client).RestartService,
	}.build(c)
}

// NewStopService stops a container; it keeps its configuration.
func NewStopService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	return lifecycleTool{
		name: "stop_service", title: "Stop service", verb: "stop",
		description: "Stop a deployed service's container on one worker. The service earns nothing until start_service runs; " +
			"configuration and data are kept.",
		destructive: true,
		call:        (*client.Client).StopService,
	}.build(c)
}
