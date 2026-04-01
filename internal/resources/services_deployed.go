package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterServicesDeployed wires cashpilot://services/deployed into the server.
func RegisterServicesDeployed(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://services/deployed",
		"Deployed services",
		mcp.WithResourceDescription("Running services with health, CPU, and memory usage"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetServicesDeployed(ctx)
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://services/deployed",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
