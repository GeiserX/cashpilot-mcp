package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterEarningsSummary wires cashpilot://earnings/summary into the server.
func RegisterEarningsSummary(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://earnings/summary",
		"Earnings summary",
		mcp.WithResourceDescription("Total earnings, today, this month, and active service count"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetEarningsSummary()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://earnings/summary",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
