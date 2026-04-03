FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/cashpilot-mcp ./cmd/server

FROM alpine:3.23
LABEL io.modelcontextprotocol.server.name="io.github.GeiserX/cashpilot-mcp"
COPY --from=builder /out/cashpilot-mcp /usr/local/bin/cashpilot-mcp
EXPOSE 8081
ENV LISTEN_ADDR=0.0.0.0:8081
ENV CASHPILOT_URL=http://cashpilot:8080
ENV CASHPILOT_API_KEY=""
ENTRYPOINT ["/usr/local/bin/cashpilot-mcp"]
