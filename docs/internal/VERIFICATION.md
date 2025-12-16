# Artifact Verification Guide

The Seed releases include Software Bills of Materials (SBOMs) and cryptographic signatures for all
binary artifacts to ensure supply chain security and integrity verification.

## What's Included

Each release contains the following files for each platform (linux-amd64, linux-arm64, darwin-amd64,
darwin-arm64):

- `seed-{platform}` - The binary executable
- `seed-{platform}.sbom.json` - Software Bill of Materials (CycloneDX format)
- `seed-{platform}.cosign.bundle` - Signature bundle for the binary
- `seed-{platform}.sbom.cosign.bundle` - Signature bundle for the SBOM

## Prerequisites

Install the following tools to verify signatures and inspect SBOMs:

```bash
# Install cosign for signature verification
# macOS
brew install cosign

# Linux (download from https://github.com/sigstore/cosign/releases)
wget https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64
chmod +x cosign-linux-amd64
sudo mv cosign-linux-amd64 /usr/local/bin/cosign

# Optional: Install syft to inspect SBOMs
# macOS
brew install syft

# Linux
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
```

## Verifying Binary Signatures

All binaries are signed using [Sigstore](https://www.sigstore.dev/) with keyless OIDC signing during
the GitHub Actions build process.

### Step 1: Download Release Assets

```bash
# Set the version you want to verify
VERSION=v0.14.0  # Change to your desired version
PLATFORM=linux-amd64  # Change to your platform

# Download the binary, SBOM, and signature bundles
gh release download $VERSION \
  -p "seed-${PLATFORM}*" \
  -R krisarmstrong/seed
```

Or download manually from the [GitHub Releases](https://github.com/krisarmstrong/seed/releases)
page.

### Step 2: Verify the Binary Signature

```bash
# Verify binary with cosign
cosign verify-blob \
  --bundle seed-${PLATFORM}.cosign.bundle \
  --certificate-identity-regexp='https://github.com/krisarmstrong/seed' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  seed-${PLATFORM}
```

Expected output:

```
Verified OK
```

### Step 3: Verify the SBOM Signature

```bash
# Verify SBOM signature
cosign verify-blob \
  --bundle seed-${PLATFORM}.sbom.cosign.bundle \
  --certificate-identity-regexp='https://github.com/krisarmstrong/seed' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  seed-${PLATFORM}.sbom.json
```

Expected output:

```
Verified OK
```

## Understanding the Signatures

### Keyless Signing

The Seed uses **keyless signing** via Sigstore, which means:

- No long-lived private keys to manage or leak
- Signatures are tied to the GitHub Actions OIDC identity
- Signatures include a timestamp from Rekor (transparency log)
- Full transparency and auditability via public log

### What's Being Verified

When you verify a signature, cosign checks:

1. **Certificate Identity**: Confirms the binary was built by the `krisarmstrong/seed` repository
2. **OIDC Issuer**: Confirms the build happened in GitHub Actions (not on a developer's laptop)
3. **Signature**: Cryptographically proves the file hasn't been tampered with since signing
4. **Transparency**: The signature is logged in Rekor, a public transparency log

## Inspecting the SBOM

The SBOM (Software Bill of Materials) lists all dependencies included in the binary.

### View SBOM Summary

```bash
# Pretty-print the SBOM
cat seed-${PLATFORM}.sbom.json | jq '.'

# List all components
cat seed-${PLATFORM}.sbom.json | jq '.components[] | {name, version}'
```

### Using Syft

If you have `syft` installed:

```bash
# Analyze the SBOM for vulnerabilities (requires Grype)
grype sbom:./seed-${PLATFORM}.sbom.json

# Convert SBOM to other formats
syft convert seed-${PLATFORM}.sbom.json -o spdx-json
```

## Verification in CI/CD

To automate verification in your CI/CD pipeline:

```yaml
- name: Download and verify The Seed
  run: |
    VERSION=v0.14.0
    PLATFORM=linux-amd64

    # Download artifacts
    gh release download $VERSION \
      -p "seed-${PLATFORM}*" \
      -R krisarmstrong/seed

    # Verify signature
    cosign verify-blob \
      --bundle seed-${PLATFORM}.cosign.bundle \
      --certificate-identity-regexp='https://github.com/krisarmstrong/seed' \
      --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
      seed-${PLATFORM}

    # Make executable
    chmod +x seed-${PLATFORM}
    mv seed-${PLATFORM} /usr/local/bin/seed
```

## Troubleshooting

### "certificate identity does not match"

This error means the binary wasn't signed by the official repository. **Do not use the binary** - it
may be tampered with.

### "signature verification failed"

This means the file has been modified after signing. **Do not use the binary** - download a fresh
copy from GitHub Releases.

### "certificate has expired"

Sigstore certificates are short-lived (hours). However, the signature includes a timestamp from
Rekor proving when it was signed. This is normal and doesn't affect verification.

### COSIGN_EXPERIMENTAL=1

For keyless verification, you may need to set:

```bash
export COSIGN_EXPERIMENTAL=1
```

This enables experimental keyless verification mode in older cosign versions. Newer versions (>=
2.0) don't require this.

## Security Policy

If you discover a security vulnerability in The Seed, please report it privately to the maintainers.
See [SECURITY.md](../SECURITY.md) for details.

## Additional Resources

- [Sigstore Documentation](https://docs.sigstore.dev/)
- [Cosign GitHub Repository](https://github.com/sigstore/cosign)
- [CycloneDX SBOM Specification](https://cyclonedx.org/)
- [Syft SBOM Tool](https://github.com/anchore/syft)
- [Grype Vulnerability Scanner](https://github.com/anchore/grype)

---

**Last Updated**: 2025-12-14 **Applies to**: The Seed v0.14.0 and later
