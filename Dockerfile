# Base images are pinned by digest; Dependabot's docker updates move tag and
# digest together.
FROM golang:1.27.4@sha256:08bcb99a38f85a265f9a93d1d3323e2440eec333380f3ea26ee0dec9264bb41c AS builder
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X github.com/geiserx/cashpilot-mcp/version.Version=${VERSION} -X github.com/geiserx/cashpilot-mcp/version.Commit=${COMMIT} -X github.com/geiserx/cashpilot-mcp/version.Date=${DATE}" \
      -o /out/cashpilot-mcp ./cmd/server

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
# The base image can lag the CA bundle; take the current one, which drops
# the compromised eMudhra (emSign) intermediates.
RUN apk upgrade --no-cache ca-certificates-bundle
LABEL io.modelcontextprotocol.server.name="io.github.GeiserX/cashpilot-mcp"
COPY --from=builder /out/cashpilot-mcp /usr/local/bin/cashpilot-mcp
EXPOSE 8081
# Inside a container 127.0.0.1 is unreachable through a published port, so the
# HTTP mode listens on every interface, and then refuses to start without
# MCP_AUTH_TOKEN. TRANSPORT=stdio (the registry's OCI entry) opens no port.
ENV LISTEN_ADDR=0.0.0.0:8081
ENV CASHPILOT_URL=http://cashpilot-ui:8080
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/cashpilot-mcp"]
