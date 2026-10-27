<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/cashpilot-mcp/main/docs/images/banner.svg" alt="cashpilot-mcp" width="900"/>
</p>

<h1 align="center">cashpilot-mcp</h1>

<p align="center">
  <a href="https://www.npmjs.com/package/cashpilot-mcp"><img src="https://img.shields.io/npm/v/cashpilot-mcp?style=flat-square&logo=npm" alt="npm"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/actions/workflows/ci.yml"><img src="https://github.com/GeiserX/cashpilot-mcp/actions/workflows/ci.yml/badge.svg" alt="CI"/></a>
  <a href="https://hub.docker.com/r/drumsergio/cashpilot-mcp"><img src="https://img.shields.io/docker/pulls/drumsergio/cashpilot-mcp?style=flat-square&logo=docker" alt="Docker Pulls"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/stargazers"><img src="https://img.shields.io/github/stars/GeiserX/cashpilot-mcp?style=flat-square&logo=github" alt="GitHub Stars"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/cashpilot-mcp?style=flat-square" alt="License"/></a>
</p>

<p align="center"><strong>A tiny bridge that exposes any CashPilot instance as an MCP server, enabling LLMs to monitor passive income earnings, manage services, and control fleet workers.</strong></p>

It talks to your own [CashPilot](https://github.com/GeiserX/CashPilot) instance with its API key, over stdio or Streamable HTTP.

Speaks MCP 2026-07-28 and every earlier revision down to 2024-11-05 from the same binary: current clients that still use the `initialize` handshake (Claude Code, Claude Desktop, Cursor, VS Code) and 2026-07-28 clients (`server/discover`, stateless requests) both work.

## Features

- Earnings: totals, per-service balances with cashout eligibility, daily series and history.
- Fleet: deployed services, the deployable catalog, workers, health scores and collector alerts, as tools and as `cashpilot://` resources.
- Service control on any worker: start, stop, restart, deploy, remove, logs, plus `trigger_collection` and a Docker Compose export per architecture.
- Read-only mode: `CASHPILOT_READ_ONLY=true` registers only the tools that change nothing.
- Every tool has a title and read-only/destructive hints, list tools return structured JSON, inputs are validated.
- HTTP mode listens on `127.0.0.1:8081`, checks `Origin`, caps request size and requires `MCP_AUTH_TOKEN` on any non-loopback address.

## Install

Every snippet runs the npm package over stdio. Use the admin key (`CASHPILOT_ADMIN_API_KEY`) for full control, or the read-only key (`CASHPILOT_READONLY_API_KEY`) with `CASHPILOT_READ_ONLY=true`. Never the fleet key.

**Claude Code**

```sh
claude mcp add --transport stdio --env CASHPILOT_URL=http://localhost:8080 --env CASHPILOT_API_KEY=<key> cashpilot -- npx -y cashpilot-mcp
```

**Claude Desktop** (`claude_desktop_config.json`) and **Cursor** (`~/.cursor/mcp.json` or `.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "cashpilot": {
      "command": "npx",
      "args": ["-y", "cashpilot-mcp"],
      "env": {
        "CASHPILOT_URL": "http://localhost:8080",
        "CASHPILOT_API_KEY": "<key>"
      }
    }
  }
}
```

**VS Code** (`.vscode/mcp.json`; the key is asked once and stored by VS Code)

```json
{
  "inputs": [
    { "id": "cashpilot-key", "type": "promptString", "description": "CashPilot API key", "password": true }
  ],
  "servers": {
    "cashpilot": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "cashpilot-mcp"],
      "env": { "CASHPILOT_URL": "http://localhost:8080", "CASHPILOT_API_KEY": "${input:cashpilot-key}" }
    }
  }
}
```

**Codex** (`~/.codex/config.toml`)

```toml
[mcp_servers.cashpilot]
command = "npx"
args = ["-y", "cashpilot-mcp"]
env = { CASHPILOT_URL = "http://localhost:8080", CASHPILOT_API_KEY = "<key>" }
```

Docker, the HTTP server and local builds are in [Getting started](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/getting-started.md); every variable is in [Configuration](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/configuration.md).

## Tools

| Tool | What it does | Changes the fleet |
|------|--------------|-------------------|
| `get_earnings_summary` | Totals: today, this month, all time | no |
| `get_earnings_breakdown` | Balance and cashout eligibility per service | no |
| `get_earnings_daily` | One total per day, 1 to 365 days | no |
| `get_earnings_history` | Records for a week, month, year or all | no |
| `list_deployed_services` | What runs where, with status and resources | no |
| `list_available_services` | Deployable catalog with the env keys each needs | no |
| `get_fleet_summary` | Workers and containers, online and total | no |
| `list_workers` | Worker ids for `worker_id` | no |
| `get_health_scores` | Health per service over 1 to 90 days | no |
| `get_collector_alerts` | Errors from the last earnings collection | no |
| `get_service_logs` | Latest container log lines (untrusted content) | no |
| `get_compose` | docker-compose.yml for amd64, arm64 or arm | no |
| `trigger_collection` | Collect earnings now, in the background | yes |
| `start_service` / `restart_service` | Start or restart a container | yes |
| `stop_service` | Stop a container; it stops earning | yes, destructive |
| `deploy_service` | Deploy or redeploy with env, hostname, instances | yes, destructive |
| `remove_service` | Remove a deployment; volumes are kept | yes, destructive |

With more than one worker online, service tools need `worker_id` (from `list_workers`). Resources and details are in [Usage](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/usage.md).

## Documentation

- [Getting started](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/getting-started.md): Docker Compose, npm, local build
- [Configuration](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/configuration.md): environment variables and MCP client configuration
- [Usage](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/usage.md): resources and tools
- [Development](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/development.md): testing, contributing, credits
- [Related projects](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/related.md): the family and registry listings

## Related projects

[CashPilot](https://github.com/GeiserX/CashPilot), the passive income fleet manager this server talks to.

## License

[GPL-3.0-or-later](https://github.com/GeiserX/cashpilot-mcp/blob/main/LICENSE)
