# Getting started

## Docker Compose

```yaml
services:
  cashpilot-mcp:
    image: drumsergio/cashpilot-mcp:v0.2.0
    ports:
      - "127.0.0.1:8081:8081"
    environment:
      - CASHPILOT_URL=http://cashpilot:8080
      - CASHPILOT_API_KEY=<your-CASHPILOT_ADMIN_API_KEY>
```

> **Security note:** The HTTP transport listens on `127.0.0.1:8081` by default. If you need to expose it on a network, place it behind a reverse proxy with authentication.

## npm (stdio transport)

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
