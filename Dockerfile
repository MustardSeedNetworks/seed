# =============================================================================
# Dockerfile - The Seed
# =============================================================================
# Multi-stage build for The Seed network diagnostic tool.
# Produces a minimal Ubuntu-based runtime image with security hardening.
#
# Build: docker build -t seed .
# Run:   docker run -d -p 443:443 -p 8443:8443 --cap-add NET_RAW seed
# =============================================================================

# Stage 1: Build frontend
FROM node:25.2.1-bookworm AS builder-frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: Build Go backend with embedded frontend
FROM golang:1.25.5-bookworm AS builder-backend
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential libpcap-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built frontend for embedding
COPY --from=builder-frontend /app/web/dist ./web/dist
# Build with embedded frontend
ARG VERSION=dev
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X github.com/krisarmstrong/seed/internal/version.Version=${VERSION}" \
    -o /seed ./cmd/seed

# Stage 3: Final runtime image (Ubuntu-based, slim)
FROM ubuntu:24.04

# Labels for container metadata
LABEL org.opencontainers.image.title="The Seed"
LABEL org.opencontainers.image.description="Network Diagnostic Tool by Mustard Seed Networks"
LABEL org.opencontainers.image.vendor="Mustard Seed Networks"
LABEL org.opencontainers.image.source="https://github.com/krisarmstrong/seed"
LABEL org.opencontainers.image.licenses="BSL-1.1"

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpcap0.8 iperf3 ca-certificates libcap2-bin && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN groupadd --system seed && \
    useradd --system --gid seed --home-dir /app --shell /usr/sbin/nologin seed

# Create directories
RUN mkdir -p /app/configs /app/logs /app/data && \
    chown -R seed:seed /app

# Copy binary and set capabilities
COPY --from=builder-backend /seed /usr/local/bin/seed
RUN setcap cap_net_raw=+ep /usr/local/bin/seed

# Copy default config (empty password triggers setup wizard)
COPY configs/seed.yaml /app/configs/seed.yaml
RUN chown seed:seed /app/configs/seed.yaml

# Switch to non-root user
USER seed
WORKDIR /app

# Expose ports (443 preferred, 8443 fallback)
EXPOSE 443 8443

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -sf -k https://localhost:8443/api/health || curl -sf -k https://localhost:443/api/health || exit 1

# Data volume for persistence
VOLUME ["/app/data", "/app/configs", "/app/logs"]

ENTRYPOINT ["/usr/local/bin/seed"]
CMD ["--config", "/app/configs/seed.yaml"]
