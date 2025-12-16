# Installation on macOS

## System Requirements

- macOS 13.0 (Ventura) or later
- Apple Silicon (M1/M2/M3) or Intel processor
- 4GB RAM minimum (8GB recommended)
- 500MB disk space

## Installation Methods

### Method 1: Download Binary (Recommended)

1. **Download** the latest release for macOS from
   [Releases](https://github.com/krisarmstrong/seed/releases/latest)

2. **Extract** the archive:

   ```bash
   tar -xzf seed-darwin-arm64.tar.gz  # Apple Silicon
   # OR
   tar -xzf seed-darwin-amd64.tar.gz  # Intel
   ```

3. **Move** to your PATH:

   ```bash
   sudo mv seed /usr/local/bin/
   sudo chmod +x /usr/local/bin/seed
   ```

4. **Verify** installation:
   ```bash
   seed --version
   ```

### Method 2: Build from Source

See the main [README](https://github.com/krisarmstrong/seed/blob/main/README.md) for build
instructions.

## First Run

1. **Launch** The Seed:

   ```bash
   seed
   ```

2. **macOS Permission Prompts:**
   - Network packet capture requires admin privileges
   - You may be prompted for your password

3. **Setup Wizard** will guide you through:
   - Network interface selection
   - Admin account creation
   - Default configuration

4. **Open Web UI:**
   - Browser should open automatically to http://localhost:8080
   - If not, manually navigate to http://localhost:8080

## Network Permissions

The Seed requires elevated privileges for packet capture.

**Run with sudo:**

```bash
sudo seed
```

## Uninstallation

```bash
# Stop The Seed if running
pkill seed

# Remove binary
sudo rm /usr/local/bin/seed

# Remove config files (optional)
rm -rf ~/.config/seed
```

## Next Steps

- [Quick Start Guide](Quick-Start-Guide.md)
- [Network Discovery](Network-Discovery.md)
- [Hardware Compatibility](Home.md)

## Troubleshooting

**Issue:** "seed: command not found"

- **Solution:** Ensure `/usr/local/bin` is in your PATH

**Issue:** "Operation not permitted" when capturing packets

- **Solution:** Run with `sudo seed`

**Issue:** Web UI doesn't load

- **Solution:** Check if port 8080 is in use: `lsof -i :8080`
