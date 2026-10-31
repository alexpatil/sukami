# ---------------------------------------------------------
# Build Stage
# ---------------------------------------------------------
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache dependency layer first
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Compile high-performance static binary with zero debug bloat
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /src/bin/sukami ./cmd/sukami

# ---------------------------------------------------------
# Production Runtime Stage (Minimal Alpine ~12MB)
# ---------------------------------------------------------
FROM alpine:latest

# Install latest CA certificates for secure TLS outbound LLM calls & curl for healthcheck
RUN apk --no-cache add ca-certificates curl \
    && addgroup -S sukami && adduser -S sukami -G sukami

WORKDIR /app

# Copy binary
COPY --from=builder /src/bin/sukami /app/sukami

# Run as non-root unprivileged user for production security
USER sukami:sukami

# Expose default gateway port
EXPOSE 8080

# Configure healthcheck against native /health endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/sukami"]
