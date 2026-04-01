package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterWorkers wires cashpilot://workers into the server.
func RegisterWorkers(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://workers",
		"Workers",
		mcp.WithResourceDescription("All registered workers (servers) in the fleet"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetWorkers(ctx)
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://workers",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
