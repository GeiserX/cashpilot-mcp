package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewDeployService builds the tool definition and handler.
func NewDeployService(c *client.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("deploy_service",
		append([]mcp.ToolOption{
			mcp.WithDescription("Deploy a catalog service on one worker: CashPilot pulls the image and starts its container(s). " +
				"CashPilot 3.0 and later refuse to replace a container that already exists there (a redeploy, or a new number of instances for a running service) until the owner confirms it in the CashPilot dashboard. " +
				"This tool cannot confirm: it reports which containers run and sends the owner to the dashboard. " +
				"Older CashPilot redeploys with the new settings at once. Keys left out of env keep their saved values. " +
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
			if msg, ok := needsApproval(slug, err); ok {
				return mcp.NewToolResultError(msg), nil
			}
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Service %s deployed. CashPilot answered: %s", slug, resp)), nil
	}

	return tool, handler
}

// needsApproval turns CashPilot 3.0's refusal to replace an existing
// container into what the agent can act on: what runs, that CashPilot wants
// the owner's confirmation, and where the owner gives it.
func needsApproval(slug string, err error) (string, bool) {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict {
		return "", false
	}
	if apiErr.Code != client.CodeConfirmDeployTargets && apiErr.Code != client.CodeDeployTargetChanged {
		return "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s already has a container on this worker, so this deploy would replace it", slug)
	if copies := describeCopies(apiErr.Containers); copies != "" {
		b.WriteString(": " + copies)
	} else if id := apiErr.Current; id != "" && id != "none" {
		b.WriteString(": container " + shortID(id))
	}
	b.WriteString(". CashPilot asks the owner to confirm before it replaces a running container, so it did not replace it. " +
		"This tool cannot give that confirmation. Tell the owner to open the CashPilot dashboard and redeploy " + slug +
		" there: the dashboard shows the containers it will replace and asks them to confirm. " +
		"The settings sent in this call may not be saved, so the owner should enter them again in the dashboard (env keys, hostname, instances). ")
	b.WriteString("CashPilot answered: " + apiErr.Detail)
	return b.String(), true
}

// describeCopies lists the containers a confirmation is about, in copy order:
// instances first, then proxied copies.
func describeCopies(containers map[string]string) string {
	keys := make([]string, 0, len(containers))
	for k, id := range containers {
		if id != "none" {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, pj := strings.HasPrefix(keys[i], "p"), strings.HasPrefix(keys[j], "p")
		if pi != pj {
			return pj
		}
		ni, _ := strconv.Atoi(strings.TrimPrefix(keys[i], "p"))
		nj, _ := strconv.Atoi(strings.TrimPrefix(keys[j], "p"))
		return ni < nj
	})
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		id := shortID(containers[k])
		if n, ok := strings.CutPrefix(k, "p"); ok {
			parts = append(parts, fmt.Sprintf("proxied copy %s (container %s)", n, id))
		} else {
			parts = append(parts, fmt.Sprintf("instance %s (container %s)", k, id))
		}
	}
	return strings.Join(parts, ", ")
}

// shortID is the 12-character form of a container id the dashboard shows.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
