package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterFleetSummary wires cashpilot://fleet/summary into the server.
func RegisterFleetSummary(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://fleet/summary",
		"Fleet summary",
		mcp.WithResourceDescription("Aggregated fleet statistics across all workers"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetFleetSummary()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://fleet/summary",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
