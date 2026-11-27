# Getting started

## Docker Compose (HTTP)

```yaml
services:
  cashpilot-mcp:
    image: drumsergio/cashpilot-mcp:v0.4.0
    ports:
      - "127.0.0.1:8081:8081"
    environment:
      - LISTEN_ADDR=0.0.0.0:8081
      - MCP_AUTH_TOKEN=<a long random string>
      - CASHPILOT_URL=http://cashpilot-ui:8080
      - CASHPILOT_API_KEY=<your CASHPILOT_ADMIN_API_KEY>
    networks:
      - cashpilot

networks:
  cashpilot:
    name: cashpilot_cashpilot_owned
    external: true
```

The server reaches CashPilot by its container name, `cashpilot-ui`, on the network of CashPilot's compose file. Compose puts the project name in front of that network's name. The project name is the folder that holds CashPilot's `docker-compose.yml`, so `cashpilot_cashpilot_owned` is right when that folder is `cashpilot`. Run `docker network ls` to see the real name and put it in `name:`.

Inside a container the server has to listen on `0.0.0.0` for the published port to reach it, and on any non-loopback address it refuses to start without `MCP_AUTH_TOKEN`. Clients connect to `http://127.0.0.1:8081/mcp` with `Authorization: Bearer <token>`. The image runs as an unprivileged user (65532).

## Docker (stdio)

```sh
docker run -i --rm --network cashpilot_cashpilot_owned -e TRANSPORT=stdio -e CASHPILOT_URL=http://cashpilot-ui:8080 -e CASHPILOT_API_KEY drumsergio/cashpilot-mcp:v0.4.0
```

`--network` takes the same network name as the Compose file above.

## npm (stdio)

```sh
npx -y cashpilot-mcp
```

Or install globally:

```sh
npm install -g cashpilot-mcp
cashpilot-mcp
```

The package downloads the Go binary for your platform from the matching [GitHub release](https://github.com/GeiserX/cashpilot-mcp/releases) and checks its SHA-256 against the `checksums.txt` shipped inside the npm package. It does this at install time, or on first run when the package manager skips install scripts (npm 12, pnpm, Bun).

## Local build

```sh
git clone https://github.com/GeiserX/cashpilot-mcp
cd cashpilot-mcp

# (optional) create .env from the sample
cp .env.example .env && $EDITOR .env

go run ./cmd/server
```
