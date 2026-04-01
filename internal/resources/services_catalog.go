package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterServicesCatalog wires cashpilot://services/catalog into the server.
func RegisterServicesCatalog(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://services/catalog",
		"Service catalog",
		mcp.WithResourceDescription("Full catalog of available bandwidth-sharing services"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetServicesCatalog()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://services/catalog",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
