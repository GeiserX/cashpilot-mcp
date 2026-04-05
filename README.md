<p align="center">
  <img src="docs/images/banner.svg" alt="CashPilot MCP banner" width="900"/>
</p>

<h1 align="center">CashPilot-MCP</h1>

<p align="center">
  <a href="https://www.npmjs.com/package/cashpilot-mcp"><img src="https://img.shields.io/npm/v/cashpilot-mcp?style=flat-square&logo=npm" alt="npm"/></a>
  <img src="https://img.shields.io/badge/Go-1.24-blue?style=flat-square&logo=go&logoColor=white" alt="Go"/>
  <a href="https://hub.docker.com/r/drumsergio/cashpilot-mcp"><img src="https://img.shields.io/docker/pulls/drumsergio/cashpilot-mcp?style=flat-square&logo=docker" alt="Docker Pulls"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/stargazers"><img src="https://img.shields.io/github/stars/GeiserX/cashpilot-mcp?style=flat-square&logo=github" alt="GitHub Stars"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/cashpilot-mcp?style=flat-square" alt="License"/></a>
</p>
<p align="center">
  <a href="https://registry.modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-Official%20Registry-E6522C?style=flat-square" alt="Official MCP Registry"/></a>
  <a href="https://glama.ai/mcp/servers/GeiserX/cashpilot-mcp"><img src="https://glama.ai/mcp/servers/GeiserX/cashpilot-mcp/badges/score.svg" alt="Glama MCP Server" /></a>
  <a href="https://mcpservers.org/servers/geiserx/cashpilot-mcp"><img src="https://img.shields.io/badge/MCPServers.org-listed-green?style=flat-square" alt="MCPServers.org"/></a>
  <a href="https://mcp.so/server/cashpilot-mcp"><img src="https://img.shields.io/badge/mcp.so-listed-blue?style=flat-square" alt="mcp.so"/></a>
  <a href="https://github.com/toolsdk-ai/toolsdk-mcp-registry"><img src="https://img.shields.io/badge/ToolSDK-Registry-orange?style=flat-square" alt="ToolSDK Registry"/></a>
</p>

<p align="center"><strong>A tiny bridge that exposes any CashPilot instance as an MCP server, enabling LLMs to monitor passive income earnings, manage services, and control fleet workers.</strong></p>

---

## What you get

| Type          | What for                                                           | MCP URI / Tool id                |
|---------------|--------------------------------------------------------------------|----------------------------------|
| **Resources** | Browse earnings, services, fleet status, and health scores read-only | `cashpilot://earnings/summary`<br>`cashpilot://earnings/breakdown`<br>`cashpilot://services/deployed`<br>`cashpilot://services/catalog`<br>`cashpilot://fleet/summary`<br>`cashpilot://workers`<br>`cashpilot://health/scores`<br>`cashpilot://collector-alerts` |
| **Tools**     | Query earnings, manage services, and trigger collection             | `get_earnings_daily`<br>`get_earnings_history`<br>`get_service_logs`<br>`restart_service`<br>`stop_service`<br>`start_service`<br>`deploy_service`<br>`remove_service`<br>`trigger_collection`<br>`get_compose` |

Everything is exposed over a single JSON-RPC endpoint (`/mcp`).
LLMs / Agents can: `initialize` -> `readResource` -> `listTools` -> `callTool` ... and so on.

---

## Quick-start (Docker Compose)

```yaml
services:
  cashpilot-mcp:
    image: drumsergio/cashpilot-mcp:latest
    ports:
      - "127.0.0.1:8081:8081"
    environment:
      - CASHPILOT_URL=http://cashpilot:8080
      - CASHPILOT_API_KEY=your-admin-api-key
```

> **Security note:** The HTTP transport listens on `127.0.0.1:8081` by default. If you need to expose it on a network, place it behind a reverse proxy with authentication.

## Install via npm (stdio transport)

```sh
npx cashpilot-mcp
```

Or install globally:

```sh
npm install -g cashpilot-mcp
cashpilot-mcp
```

This downloads the pre-built Go binary from GitHub Releases for your platform and runs it with stdio transport. Requires at least one [published release](https://github.com/GeiserX/cashpilot-mcp/releases).

## Local build

```sh
git clone https://github.com/GeiserX/cashpilot-mcp
cd cashpilot-mcp

# (optional) create .env from the sample
cp .env.example .env && $EDITOR .env

go run ./cmd/server
```

## Configuration

| Variable           | Default                    | Description                                      |
|--------------------|----------------------------|--------------------------------------------------|
| `CASHPILOT_URL`    | `http://localhost:8080`    | CashPilot instance URL (without trailing /)      |
| `CASHPILOT_API_KEY`| _(required)_               | Admin API key for authentication                 |
| `LISTEN_ADDR`      | `127.0.0.1:8081`           | HTTP listen address (Docker sets `0.0.0.0:8081`) |
| `TRANSPORT`        | _(empty = HTTP)_           | Set to `stdio` for stdio transport               |

Put them in a `.env` file (from `.env.example`) or set them in the environment.

## Testing

Tested with [Inspector](https://modelcontextprotocol.io/docs/tools/inspector) and it is currently fully working. Before making a PR, make sure this MCP server behaves well via this medium.

## Example configuration for client LLMs

```json
{
  "schema_version": "v1",
  "name_for_human": "CashPilot-MCP",
  "name_for_model": "cashpilot_mcp",
  "description_for_human": "Monitor passive income earnings, manage bandwidth-sharing services, and control fleet workers via CashPilot.",
  "description_for_model": "Interact with a CashPilot instance that manages passive income services. First call initialize, then reuse the returned session id in header \"Mcp-Session-Id\" for every other call. Use readResource to fetch URIs that begin with cashpilot://. Use listTools to discover available actions and callTool to execute them.",
  "auth": { "type": "none" },
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

## Credits

[CashPilot](https://github.com/GeiserX/CashPilot) -- passive income fleet management dashboard

[MCP-GO](https://github.com/mark3labs/mcp-go) -- modern MCP implementation

[GoReleaser](https://goreleaser.com/) -- painless multi-arch releases

## Maintainers

[@GeiserX](https://github.com/GeiserX).

## Contributing

Feel free to dive in! [Open an issue](https://github.com/GeiserX/cashpilot-mcp/issues/new) or submit PRs.

CashPilot-MCP follows the [Contributor Covenant](http://contributor-covenant.org/version/2/1/) Code of Conduct.

## Other MCP Servers by GeiserX

- [duplicacy-mcp](https://github.com/GeiserX/duplicacy-mcp) — Backup health monitoring
- [genieacs-mcp](https://github.com/GeiserX/genieacs-mcp) — TR-069 device management
- [lynxprompt-mcp](https://github.com/GeiserX/lynxprompt-mcp) — AI configuration blueprints
- [pumperly-mcp](https://github.com/GeiserX/pumperly-mcp) — Fuel and EV charging prices
- [telegram-archive-mcp](https://github.com/GeiserX/telegram-archive-mcp) — Telegram message archive
