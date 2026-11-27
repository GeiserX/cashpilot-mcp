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
				"Keys left out of env keep their saved values. The env keys a service needs (account email, token, ...) are listed in list_available_services. Can take a few minutes. " +
				"A redeploy, or a new number of instances for a running service, replaces running containers. CashPilot 3.0 and later do that only after the owner agrees: " +
				"the first call replaces nothing and lists the containers it would replace. Show that list to the owner. " +
				"If the owner agrees, call again with the same arguments plus replace_containers, copied from that answer. " +
				"Older CashPilot redeploys at once."),
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
			mcp.WithObject("replace_containers",
				mcp.AdditionalProperties(map[string]any{"type": "string", "pattern": `^(none|[0-9a-f]{12,64})$`}),
				mcp.Description(`Send only after the owner agreed to replace the running containers. Copy it unchanged from the answer of the previous deploy_service call: copy number to container id, e.g. {"1":"<container id>"}. A 3.0 worker replaces exactly these containers, and refuses if any of them changed. An older worker cannot check them and could replace a different container.`),
			),
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
		// replace_existing asks CashPilot 3.0 for the containers a redeploy
		// replaces; with nothing running it deploys at once. The owner's yes is
		// that list sent back as expected_containers (as the dashboard does).
		body["replace_existing"] = true
		if raw, ok := args["replace_containers"]; ok && raw != nil {
			confirmed, err := containersArg(raw)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			body["expected_containers"] = confirmed
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

// containersArg reads replace_containers: an object of container ids in the
// shape CashPilot sent it.
func containersArg(raw any) (map[string]string, error) {
	in, ok := raw.(map[string]any)
	if !ok {
		return nil, client.ErrInvalidContainers
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		id, ok := v.(string)
		if !ok {
			return nil, client.ErrInvalidContainers
		}
		out[k] = id
	}
	if err := client.ValidateContainers(out); err != nil {
		return nil, err
	}
	return out, nil
}

// needsApproval turns CashPilot 3.0's refusal to replace an existing
// container into what the agent can act on: what runs, that the owner must
// agree, and the replace_containers value that carries the owner's yes.
func needsApproval(slug string, err error) (string, bool) {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict {
		return "", false
	}
	var b strings.Builder
	switch apiErr.Code {
	case client.CodeConfirmDeployTargets:
		fmt.Fprintf(&b, "%s already runs on this worker", slug)
		if copies := describeCopies(apiErr.Containers); copies != "" {
			b.WriteString(": " + copies)
		}
		b.WriteString(". This deploy replaces those containers, and CashPilot does that only after the owner agrees. CashPilot has not replaced them. ")
		if strings.Contains(apiErr.Detail, "last reported") {
			b.WriteString("This worker is old: the ids are the ones it last reported, so it could replace a different container. Tell the owner this before asking. ")
		}
		if confirm, err := json.Marshal(apiErr.Containers); err == nil && len(apiErr.Containers) > 0 {
			b.WriteString("Ask the owner whether to replace them. If they agree, call deploy_service again with the same arguments and replace_containers set to " +
				string(confirm) + ". ")
		} else {
			b.WriteString("CashPilot did not say which containers, so ask the owner to redeploy " + slug + " from the CashPilot dashboard. ")
		}
	case client.CodeDeployTargetChanged:
		fmt.Fprintf(&b, "A container of %s on this worker is not the one this deploy expected", slug)
		if id := apiErr.Current; id != "" && id != "none" {
			b.WriteString(" (now container " + shortID(id) + ")")
		}
		b.WriteString(". CashPilot stopped at that container. Copies it reached before it may already run the new settings. Call deploy_service again without replace_containers to see what runs now, then ask the owner again. ")
	default:
		return "", false
	}
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
