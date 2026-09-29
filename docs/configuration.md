# Configuration

| Variable           | Default                    | Description                                      |
|--------------------|----------------------------|--------------------------------------------------|
| `CASHPILOT_URL`    | `http://localhost:8080`    | CashPilot instance URL (without trailing /)      |
| `CASHPILOT_API_KEY`| _(required)_               | Admin API key (`CASHPILOT_ADMIN_API_KEY` from your CashPilot instance — NOT the fleet key) |
| `LISTEN_ADDR`      | `127.0.0.1:8081`           | HTTP listen address (Docker sets `127.0.0.1:8081`) |
| `MCP_AUTH_TOKEN`   | _(empty)_                  | Bearer token for HTTP transport auth. **Required** when `LISTEN_ADDR` is not loopback |
| `TRANSPORT`        | _(empty = HTTP)_           | Set to `stdio` for stdio transport               |

Put them in a `.env` file (from `.env.example`) or set them in the environment.

## MCP client configuration

The npm package always runs over stdio. Add this to your client's `mcpServers` (Claude Desktop, Claude Code, Cursor):

```json
{
  "mcpServers": {
    "cashpilot": {
      "command": "npx",
      "args": ["-y", "cashpilot-mcp"],
      "env": {
        "CASHPILOT_URL": "http://localhost:8080",
        "CASHPILOT_API_KEY": "<your CASHPILOT_ADMIN_API_KEY>"
      }
    }
  }
}
```

For the HTTP server (Docker or `go run`), point a client that accepts remote servers at `/mcp`, with the bearer token when `MCP_AUTH_TOKEN` is set:

```json
{
  "mcpServers": {
    "cashpilot": {
      "type": "http",
      "url": "http://127.0.0.1:8081/mcp",
      "headers": { "Authorization": "Bearer <your MCP_AUTH_TOKEN>" }
    }
  }
}
```
