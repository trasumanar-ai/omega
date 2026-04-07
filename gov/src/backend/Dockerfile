FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o serve ./cmd/serve/

FROM alpine:latest
RUN mkdir /data
COPY --from=builder /app/serve /app/serve
EXPOSE 8090
ENTRYPOINT ["/app/serve", "-port", "8090", "-db", "/data/omega.db"]
