package resources

import (
	"context"

	"github.com/geiserx/cashpilot-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterEarningsBreakdown wires cashpilot://earnings/breakdown into the server.
func RegisterEarningsBreakdown(s *server.MCPServer, c *client.Client) {
	res := mcp.NewResource(
		"cashpilot://earnings/breakdown",
		"Earnings breakdown",
		mcp.WithResourceDescription("Per-platform earnings with cashout eligibility"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := c.GetEarningsBreakdown()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "cashpilot://earnings/breakdown",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
