# Installation on Linux

## System Requirements

- Linux kernel 4.15+ (Ubuntu 18.04+, Debian 10+, RHEL 8+)
- x86_64 or ARM64 architecture
- 4GB RAM minimum (8GB recommended)
- 500MB disk space
- `libpcap` installed (for packet capture)

## Supported Distributions

| Distribution     | Version              | Status              |
| ---------------- | -------------------- | ------------------- |
| Ubuntu           | 22.04 LTS, 24.04 LTS | Recommended         |
| Debian           | 11, 12               | Supported           |
| RHEL / AlmaLinux | 8, 9                 | Supported           |
| Fedora           | 38, 39               | Supported           |
| Arch Linux       | Latest               | Community supported |

## Installation Methods

### Method 1: Download Binary (Recommended)

1. **Install dependencies:**

   **Ubuntu/Debian:**

   ```bash
   sudo apt update
   sudo apt install libpcap0.8
   ```

   **RHEL/AlmaLinux/Fedora:**

   ```bash
   sudo dnf install libpcap
   ```

2. **Download** the latest release:

   ```bash
   wget https://github.com/krisarmstrong/seed/releases/latest/download/seed-linux-amd64.tar.gz
   # OR for ARM64
   wget https://github.com/krisarmstrong/seed/releases/latest/download/seed-linux-arm64.tar.gz
   ```

3. **Extract** the archive:

   ```bash
   tar -xzf seed-linux-amd64.tar.gz
   ```

4. **Move** to your PATH:

   ```bash
   sudo mv seed /usr/local/bin/
   sudo chmod +x /usr/local/bin/seed
   ```

5. **Grant network capture capabilities:**

   ```bash
   sudo setcap cap_net_raw=+ep /usr/local/bin/seed
   ```

6. **Verify** installation:

   ```bash
   seed --version
   ```

### Method 2: Build from Source

See [Building from Source](Building-from-Source) guide.

## Running as a Service (systemd)

**Recommended for servers and production deployments.**

1. **Create systemd service file:**

   ```bash
   sudo nano /etc/systemd/system/seed.service
   ```

2. **Add configuration:**

   ```ini
   [Unit]
   Description=The Seed - AI-Powered Network Diagnostics
   After=network.target

   [Service]
   Type=simple
   User=seed
   Group=seed
   ExecStart=/usr/local/bin/seed
   Restart=on-failure
   RestartSec=10

   # Security hardening
   NoNewPrivileges=true
   PrivateTmp=true

   [Install]
   WantedBy=multi-user.target
   ```

3. **Create dedicated user:**

   ```bash
   sudo useradd -r -s /bin/false seed
   ```

4. **Grant capabilities:**

   ```bash
   sudo setcap cap_net_raw=+ep /usr/local/bin/seed
   ```

5. **Enable and start:**

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable seed
   sudo systemctl start seed
   ```

6. **Check status:**

   ```bash
   sudo systemctl status seed
   ```

7. **View logs:**

   ```bash
   sudo journalctl -u seed -f
   ```

## Firewall Configuration

**Allow Web UI access (port 8080):**

**UFW (Ubuntu):**

```bash
sudo ufw allow 8080/tcp
```

**firewalld (RHEL/Fedora):**

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

## Uninstallation

```bash
# Stop service
sudo systemctl stop seed
sudo systemctl disable seed

# Remove binary
sudo rm /usr/local/bin/seed

# Remove service file
sudo rm /etc/systemd/system/seed.service
sudo systemctl daemon-reload

# Remove config files (optional)
sudo rm -rf /etc/seed
sudo rm -rf ~/.config/seed
```

## Next Steps

- [First-Time Setup](First-Time-Setup)
- [Quick Start Guide](Quick-Start-Guide)
- [Running as a Service (systemd)](https://github.com/krisarmstrong/seed/tree/main/deploy/systemd)

## Troubleshooting

**Issue:** "Operation not permitted" when running

- **Solution:** Grant network capture capability: `sudo setcap cap_net_raw=+ep /usr/local/bin/seed`

**Issue:** "Cannot bind to port 8080"

- **Solution:** Port already in use. Stop conflicting service or change port in config.

**Issue:** Service fails to start

- **Solution:** Check logs: `sudo journalctl -u seed -n 50`

[More Troubleshooting](Troubleshooting)
