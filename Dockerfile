# Stage 1: Build Go backend (Debian-based for CGO with libpcap)
FROM golang:1.25-bookworm AS builder-backend
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential libpcap-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /netscope ./cmd/netscope

# Stage 2: Build frontend
FROM node:20-bookworm-slim AS builder-frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 3: Final runtime image (Debian-based)
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpcap0.8 iperf3 ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder-backend /netscope /usr/local/bin/netscope
COPY --from=builder-frontend /app/web/dist /usr/share/netscope/web
COPY configs /etc/netscope
EXPOSE 8443
ENTRYPOINT ["/usr/local/bin/netscope"]
CMD ["--config", "/etc/netscope/netscope.yaml"]
