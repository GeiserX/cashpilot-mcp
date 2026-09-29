<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/cashpilot-mcp/main/docs/images/banner.svg" alt="CashPilot MCP banner" width="900"/>
</p>

<h1 align="center">CashPilot-MCP</h1>

<p align="center">
  <a href="https://www.npmjs.com/package/cashpilot-mcp"><img src="https://img.shields.io/npm/v/cashpilot-mcp?style=flat-square&logo=npm" alt="npm"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/actions/workflows/ci.yml"><img src="https://github.com/GeiserX/cashpilot-mcp/actions/workflows/ci.yml/badge.svg" alt="CI"/></a>
  <a href="https://hub.docker.com/r/drumsergio/cashpilot-mcp"><img src="https://img.shields.io/docker/pulls/drumsergio/cashpilot-mcp?style=flat-square&logo=docker" alt="Docker Pulls"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/stargazers"><img src="https://img.shields.io/github/stars/GeiserX/cashpilot-mcp?style=flat-square&logo=github" alt="GitHub Stars"/></a>
  <a href="https://github.com/GeiserX/cashpilot-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/cashpilot-mcp?style=flat-square" alt="License"/></a>
</p>

<p align="center"><strong>A tiny bridge that exposes any CashPilot instance as an MCP server, enabling LLMs to monitor passive income earnings, manage services, and control fleet workers.</strong></p>

It talks to your own [CashPilot](https://github.com/GeiserX/CashPilot) instance with its admin API key, over HTTP or stdio.

## Features

- Read-only resources for earnings, deployed services, the catalog, fleet status, workers, health scores and collector alerts (`cashpilot://earnings/summary`, `cashpilot://fleet/summary`, ...).
- Tools to query daily and historical earnings, read service logs, and start, stop, restart, deploy or remove services.
- `trigger_collection` runs an earnings collection across all services now; `get_compose` returns a service's Docker Compose definition.
- One JSON-RPC endpoint (`/mcp`) over HTTP, or stdio with `TRANSPORT=stdio`.
- Listens on `127.0.0.1:8081` by default; `MCP_AUTH_TOKEN` adds bearer auth when you expose it.
- Ships as a Docker image, an npm package (`npx cashpilot-mcp`) and multi-arch Go binaries.

## Quick start

```sh
npx cashpilot-mcp
```

Set `CASHPILOT_URL` and `CASHPILOT_API_KEY` (your `CASHPILOT_ADMIN_API_KEY`, not the fleet key) first. Docker Compose and local builds are in [Installation](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/installation.md).

## Documentation

- [Installation](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/installation.md): Docker Compose, npm, local build
- [Configuration](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/configuration.md): environment variables and an example client config
- [Resources and tools](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/usage.md)
- [Development](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/development.md): testing, contributing, credits
- [Related projects and listings](https://github.com/GeiserX/cashpilot-mcp/blob/main/docs/related.md)

Related: [CashPilot](https://github.com/GeiserX/CashPilot), the passive income fleet manager this server talks to.

## License

[GPL-3.0](https://github.com/GeiserX/cashpilot-mcp/blob/main/LICENSE)
