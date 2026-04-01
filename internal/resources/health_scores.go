package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterHealthScores wires cashpilot://health/scores into the server.
func RegisterHealthScores(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://health/scores",
		"Health scores",
		mcp.WithResourceDescription("Per-service health scores across the fleet"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetHealthScores()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://health/scores",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
