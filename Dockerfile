# Multi-stage build: stage 1 compiles the binary
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o /truenas-mcp ./cmd/truenas-mcp

# Stage 2: minimal runtime image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata docker-cli
COPY --from=builder /truenas-mcp /usr/local/bin/truenas-mcp
EXPOSE 8080
ENTRYPOINT ["truenas-mcp"]
CMD ["--transport=http", "--http-addr=:8080"]