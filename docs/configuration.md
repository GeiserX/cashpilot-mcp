# Configuration

| Variable           | Default                    | Description                                      |
|--------------------|----------------------------|--------------------------------------------------|
| `CASHPILOT_URL`    | `http://localhost:8080`    | CashPilot instance URL (without trailing /)      |
| `CASHPILOT_API_KEY`| _(required)_               | Admin API key (`CASHPILOT_ADMIN_API_KEY` from your CashPilot instance — NOT the fleet key) |
| `LISTEN_ADDR`      | `127.0.0.1:8081`           | HTTP listen address (Docker sets `127.0.0.1:8081`) |
| `MCP_AUTH_TOKEN`   | _(empty)_                  | Bearer token for HTTP transport auth. **Required** when `LISTEN_ADDR` is not loopback |
| `TRANSPORT`        | _(empty = HTTP)_           | Set to `stdio` for stdio transport               |

Put them in a `.env` file (from `.env.example`) or set them in the environment.

## Example configuration for client LLMs

```json
{
  "schema_version": "v1",
  "name_for_human": "CashPilot-MCP",
  "name_for_model": "cashpilot_mcp",
  "description_for_human": "Monitor passive income earnings, manage bandwidth-sharing services, and control fleet workers via CashPilot.",
  "description_for_model": "Interact with a CashPilot instance that manages passive income services. First call initialize, then reuse the returned session id in header \"Mcp-Session-Id\" for every other call. Use readResource to fetch URIs that begin with cashpilot://. Use listTools to discover available actions and callTool to execute them.",
  "auth": {
    "type": "bearer",
    "token": "<your-MCP_AUTH_TOKEN>"
  },
  "api": {
    "type": "jsonrpc-mcp",
    "url":  "http://localhost:8081/mcp",
    "init_method": "initialize",
    "session_header": "Mcp-Session-Id"
  },
  "contact_email": "acsdesk@protonmail.com",
  "legal_info_url": "https://github.com/GeiserX/cashpilot-mcp/blob/main/LICENSE"
}
```
