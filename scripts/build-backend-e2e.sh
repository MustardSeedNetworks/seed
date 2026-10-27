#!/usr/bin/env bash
#
# Build the seed binary the E2E jobs run.
#
# CI runs the daemon inside mcr.microsoft.com/playwright:v1.64.1-noble, which
# ships no libpcap and cannot apt-install one (its sources point at
# azure.archive.ubuntu.com, which the container network cannot reach). So the
# e2e caller's daemon job builds it on a host runner, with cgo against the static
# libpcap from scripts/build-static-libpcap.sh, passed in through CGO_CFLAGS and
# CGO_LDFLAGS. The binary then needs libc alone and captures in the image, so
# the packet-capture spec runs its capture path in CI (#2986). Locally the
# system libpcap serves: libpcap-dev on Linux, the SDK's on a Mac.
#
# The build is cgo unless the caller sets CGO_ENABLED=0. Only the phone-width
# job does, because the fleet's reusable workflow has no libpcap; that daemon
# selects the null capture adapter, whose OpenLive returns ErrUnavailable. A
# CGO_ENABLED=0 darwin build does not compile: CoreWLAN, the Wi-Fi helper's
# peer check and the scoped DNS and DHCP readers are cgo-only there (#2518).
#
# The empty-UI guard below is why this is a script and not
# `make build-backend-quiet`.
#
# The ldflags MUST mirror the Makefile's GO_LDFLAGS. Per the universal build
# contract, a binary built without them reports "unknown" from /__version,
# which is a silent violation — raw `go build` in CI is called out by name.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
UI_BUILD_HASH=$(./scripts/ui-build-hash.sh)
VERSION_PKG="github.com/MustardSeedNetworks/seed/internal/version"

# ui-build-hash.sh prints "unknown" for an empty internal/api/ui. That means the
# frontend-dist artifact did not land, and the resulting binary would serve no
# UI at all — every spec would fail at first paint with a blank page. Fail here
# instead, where the cause is obvious.
if [ "$UI_BUILD_HASH" = "unknown" ]; then
  echo "::error::internal/api/ui is empty — the frontend-dist artifact did not land; refusing to build a UI-less binary" >&2
  exit 1
fi

CGO_ENABLED=${CGO_ENABLED:-1}
export CGO_ENABLED

echo "building e2e backend: version=${VERSION} commit=${COMMIT} uiBuildHash=${UI_BUILD_HASH} cgo=${CGO_ENABLED}"

go build -trimpath -buildvcs=false \
  -ldflags="-s -w \
    -X ${VERSION_PKG}.Version=${VERSION} \
    -X ${VERSION_PKG}.Commit=${COMMIT} \
    -X ${VERSION_PKG}.BuildTime=${BUILD_TIME} \
    -X ${VERSION_PKG}.UIBuildHash=${UI_BUILD_HASH}" \
  -o seed ./cmd/seed
