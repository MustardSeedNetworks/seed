# Stage 1: Build Go backend
FROM golang:1.25-alpine AS builder-backend
WORKDIR /app
RUN apk add --no-cache gcc musl-dev libpcap-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /netscope ./cmd/netscope

# Stage 2: Build frontend
FROM node:20-alpine AS builder-frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 3: Final image
FROM alpine:latest
RUN apk add --no-cache libpcap iperf3
COPY --from=builder-backend /netscope /usr/local/bin/netscope
COPY --from=builder-frontend /app/web/dist /usr/share/netscope/web
COPY configs /etc/netscope
EXPOSE 8443
ENTRYPOINT ["/usr/local/bin/netscope"]
CMD ["--config", "/etc/netscope/netscope.yaml"]
