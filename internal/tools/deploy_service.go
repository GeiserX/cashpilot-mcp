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
		append([]mcp.ToolOption{
			mcp.WithDescription("Deploy a catalog service on one worker: CashPilot pulls the image and starts its container(s). " +
				"Deploying a service that already runs there redeploys it with the new settings; keys left out of env keep their saved values. " +
				"The env keys a service needs (account email, token, ...) are listed in list_available_services. Can take a few minutes."),
			slugParam("Service to deploy."),
			mcp.WithObject("env",
				mcp.AdditionalProperties(map[string]any{"type": "string"}),
				mcp.Description(`Environment variables for the container, as string values, e.g. {"EMAIL":"me@example.com","PASSWORD":"..."}.`),
			),
			mcp.WithString("hostname",
				mcp.MaxLength(63),
				mcp.Description("Device name the provider shows for this node. Default: CashPilot's own."),
			),
			mcp.WithInteger("instances",
				mcp.Min(1), mcp.Max(MaxInstances),
				mcp.Description("Copies to run on this worker, 1 to 10. Many providers allow one per IP; CashPilot refuses more where that is the case. Default: keep the current number, 1 for a new deployment."),
			),
			workerParam(),
		}, hints("Deploy service", false, true, false, true)...)...,
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
		args := req.GetArguments()
		body := map[string]any{}
		if raw, ok := args["env"]; ok && raw != nil {
			envIn, ok := raw.(map[string]any)
			if !ok {
				return mcp.NewToolResultError(`env must be an object of string values, e.g. {"EMAIL":"me@example.com"}`), nil
			}
			env := make(map[string]string, len(envIn))
			for k, v := range envIn {
				s, ok := v.(string)
				if !ok {
					return mcp.NewToolResultError(fmt.Sprintf("env value for %q must be a string", k)), nil
				}
				env[k] = s
			}
			body["env"] = env
		}
		if h := req.GetString("hostname", ""); h != "" {
			body["hostname"] = h
		}
		if _, ok := args["instances"]; ok {
			n, err := intArg(req, "instances", 0, 1, MaxInstances)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			body["instances"] = n
		}
		payload, err := json.Marshal(body)
		if err != nil {
			return mcp.NewToolResultError("could not encode the deploy request"), nil
		}

		resp, err := c.DeployService(ctx, slug, payload, t)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Service %s deployed. CashPilot answered: %s", slug, resp)), nil
	}

	return tool, handler
}
