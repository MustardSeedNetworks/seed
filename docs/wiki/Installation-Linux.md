# Installation on Linux

## System Requirements

- Linux kernel 4.15+ (Ubuntu 18.04+, Debian 10+, RHEL 8+)
- x86_64 or ARM64 architecture
- 4GB RAM minimum (8GB recommended)
- 500MB disk space
- `libpcap` installed (for packet capture)

## Installation

### 1. Install Dependencies

**Ubuntu/Debian:**

```bash
sudo apt update
sudo apt install libpcap0.8
```

**RHEL/AlmaLinux/Fedora:**

```bash
sudo dnf install libpcap
```

### 2. Download Binary

```bash
wget https://github.com/krisarmstrong/netscope/releases/latest/download/seed-linux-amd64.tar.gz
# OR for ARM64
wget https://github.com/krisarmstrong/netscope/releases/latest/download/seed-linux-arm64.tar.gz
```

### 3. Install

```bash
tar -xzf seed-linux-amd64.tar.gz
sudo mv seed /usr/local/bin/
sudo chmod +x /usr/local/bin/seed
```

### 4. Grant Network Capabilities

```bash
sudo setcap cap_net_raw=+ep /usr/local/bin/seed
```

### 5. Verify

```bash
seed --version
```

## Running as a Service (systemd)

See [deploy/systemd](https://github.com/krisarmstrong/netscope/tree/main/deploy/systemd) for systemd
service installation.

## Firewall Configuration

Allow Web UI access (port 8080):

**UFW (Ubuntu):**

```bash
sudo ufw allow 8080/tcp
```

**firewalld (RHEL/Fedora):**

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

## Next Steps

- [Quick Start Guide](Quick-Start-Guide.md)
- [Network Discovery](Network-Discovery.md)

## Troubleshooting

**Issue:** "Operation not permitted"

- **Solution:** Grant network capability: `sudo setcap cap_net_raw=+ep /usr/local/bin/seed`

**Issue:** "Cannot bind to port 8080"

- **Solution:** Port in use. Check: `sudo lsof -i :8080`
