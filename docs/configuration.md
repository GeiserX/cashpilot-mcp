# Configuration

| Variable              | Default                 | Description |
|-----------------------|-------------------------|-------------|
| `CASHPILOT_URL`       | `http://localhost:8080` | CashPilot instance URL (without trailing /) |
| `CASHPILOT_API_KEY`   | _(required)_            | CashPilot API key. The admin key (`CASHPILOT_ADMIN_API_KEY`) allows every tool. The read-only key (`CASHPILOT_READONLY_API_KEY`) allows the reads except `get_service_logs`; pair it with `CASHPILOT_READ_ONLY=true`. Never the fleet key. |
| `CASHPILOT_READ_ONLY` | `false`                 | `true` (or `1`) registers only the tools that change nothing: no start, stop, restart, deploy, remove or collection. |
| `TRANSPORT`           | _(empty = HTTP)_        | `stdio` for stdio. The npm package always uses stdio. |
| `LISTEN_ADDR`         | `127.0.0.1:8081`        | HTTP listen address. The Docker image sets `0.0.0.0:8081` so a published port reaches it. |
| `MCP_AUTH_TOKEN`      | _(empty)_               | Bearer token for the HTTP endpoint, sent as `Authorization: Bearer <token>`. **Required** when `LISTEN_ADDR` is not loopback. |
| `MCP_ALLOWED_ORIGINS` | _(empty)_               | Comma-separated browser origins (`https://app.example.com`) allowed to call the HTTP endpoint cross-site. Requests without an `Origin` header, which is what MCP clients send, are always allowed. |

In HTTP mode a `.env` file (from `.env.example`) in the working directory is read too; stdio mode reads only the environment.

## Protocol

The server speaks MCP 2026-07-28, 2025-11-25, 2025-06-18, 2025-03-26 and 2024-11-05. Clients that open with `initialize` negotiate up to 2025-11-25; 2026-07-28 clients call `server/discover` and send stateless requests. The HTTP endpoint is stateless and never issues `Mcp-Session-Id`.

HTTP limits: request bodies over 1 MiB get 413, a cross-site `Origin` gets 403, headers must arrive within 10 seconds, and idle connections close after 2 minutes.

For legacy clients, an authenticated `GET /mcp` without an `MCP-Protocol-Version` header opens a server-sent event stream that stays open until the client closes it, and `DELETE /mcp` answers 200 with nothing to delete. With `MCP-Protocol-Version: 2026-07-28`, `GET` answers 405.

## MCP client configuration

The README has ready snippets for Claude Code, Claude Desktop, Cursor, VS Code and Codex. For the HTTP server (Docker or `go run`), point a client that accepts remote servers at `/mcp` with the bearer token:

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

Claude Code: `claude mcp add --transport http cashpilot http://127.0.0.1:8081/mcp --header "Authorization: Bearer <your MCP_AUTH_TOKEN>"`.
