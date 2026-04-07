FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY src/backend/go.mod src/backend/go.sum ./
RUN go mod download
COPY src/backend/ ./
RUN CGO_ENABLED=0 go build -o /omega-id ./cmd/identity/
RUN CGO_ENABLED=0 go build -o /omega-proxy ./cmd/proxy/

FROM ghcr.io/openclaw/openclaw:latest
COPY --from=builder /omega-id /usr/local/bin/omega-id
COPY --from=builder /omega-proxy /usr/local/bin/omega-proxy
COPY agents/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
