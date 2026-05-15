# Super AI Gateway — Dockerfile
# Multi-stage build: compile Go binary, minimal runtime image

FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway ./cmd/gateway

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /gateway /usr/local/bin/gateway
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/gateway"]
CMD ["--config", "/etc/gateway/config.yaml"]
