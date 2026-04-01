package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterCollectorAlerts wires cashpilot://collector-alerts into the server.
func RegisterCollectorAlerts(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://collector-alerts",
		"Collector alerts",
		mcp.WithResourceDescription("Recent collection errors and alerts"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetCollectorAlerts()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://collector-alerts",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
