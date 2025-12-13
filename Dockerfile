# Stage 1: Build frontend
FROM node:20-noble AS builder-frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: Build Go backend with embedded frontend
FROM golang:1.25-noble AS builder-backend
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
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /luminetiq ./cmd/netscope

# Stage 3: Final runtime image (Ubuntu-based, slim)
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpcap0.8 iperf3 ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder-backend /luminetiq /usr/local/bin/luminetiq
COPY configs /etc/luminetiq
EXPOSE 8443
ENTRYPOINT ["/usr/local/bin/luminetiq"]
CMD ["--config", "/etc/luminetiq/luminetiq.yaml"]
