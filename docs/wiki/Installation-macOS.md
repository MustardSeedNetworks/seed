# Installation on macOS

## System Requirements

- macOS 13.0 (Ventura) or later
- Apple Silicon (M1/M2/M3) or Intel processor
- 4GB RAM minimum (8GB recommended)
- 500MB disk space

## Installation Methods

### Method 1: Homebrew (Recommended)

**Coming Soon:** The Seed will be available via Homebrew tap.

```bash
# Not yet available - planned for launch
brew install mustardseednetworks/tap/seed
```

### Method 2: Download Binary

1. **Download** the latest release for macOS:
   - [Download for Apple Silicon (M1/M2/M3)](https://github.com/krisarmstrong/seed/releases/latest)
   - [Download for Intel](https://github.com/krisarmstrong/seed/releases/latest)

2. **Extract** the archive:

   ```bash
   tar -xzf seed-darwin-arm64.tar.gz  # Apple Silicon
   # OR
   tar -xzf seed-darwin-amd64.tar.gz  # Intel
   ```

3. **Move** to your PATH:

   ```bash
   sudo mv seed /usr/local/bin/
   ```

4. **Verify** installation:

   ```bash
   seed --version
   ```

### Method 3: Build from Source

See [Building from Source](Building-from-Source) guide.

## First Run

1. **Launch** The Seed:

   ```bash
   seed
   ```

2. **macOS Permission Prompts:**
   - You'll see "seed wants to access files" - Click **OK**
   - Network packet capture requires admin privileges

3. **Setup Wizard** will guide you through:
   - Network interface selection
   - Admin password (for first-time user creation)
   - Default configuration

4. **Open Web UI:**
   - Browser should open automatically to `http://localhost:8080`
   - If not, manually navigate to `http://localhost:8080`

## Network Permissions

The Seed requires elevated privileges for packet capture.

**macOS Ventura 13+ (Recommended):**

```bash
# Grant network access without full sudo
sudo chmod +x /usr/local/bin/seed
```

**Alternative:** Run with sudo:

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

- [First-Time Setup](First-Time-Setup)
- [Quick Start Guide](Quick-Start-Guide)
- [Configuration](Configuration)

## Troubleshooting

**Issue:** "seed: command not found"

- **Solution:** Add `/usr/local/bin` to your PATH or move `seed` to `/usr/bin`

**Issue:** "Operation not permitted" when capturing packets

- **Solution:** Run with `sudo` or grant network access permissions

**Issue:** Web UI doesn't load

- **Solution:** Check if port 8080 is already in use: `lsof -i :8080`

[More Troubleshooting](Troubleshooting)
