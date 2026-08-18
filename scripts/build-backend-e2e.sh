#!/usr/bin/env bash
#
# Build the seed binary the E2E jobs run.
#
# Why this exists rather than `make build-backend-quiet`: the E2E jobs run
# inside mcr.microsoft.com/playwright:v1.62.1-noble, and that image ships no
# `make` — and no gcc, and no libpcap. It also cannot apt-install them: the
# image's sources point at azure.archive.ubuntu.com, which is unreachable from
# the container network (every index fails with a connection timeout). So the
# E2E binary is built CGO_ENABLED=0.
#
# What that costs: live packet capture is unavailable, and internal/api's
# `!cgo` build tag selects the null capture adapter, whose OpenLive returns
# ErrUnavailable rather than panicking. Nothing in the E2E suite exercises
# capture — no spec references pcap — and run-e2e.sh's generated config already
# disables networkDiscovery and every passive protocol. The CGO_ENABLED=1 build
# stays covered by the backend, race and build jobs, which run on the host
# runner where libpcap is available.
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

echo "building e2e backend: version=${VERSION} commit=${COMMIT} uiBuildHash=${UI_BUILD_HASH}"

CGO_ENABLED=0 go build -trimpath -buildvcs=false \
  -ldflags="-s -w \
    -X ${VERSION_PKG}.Version=${VERSION} \
    -X ${VERSION_PKG}.Commit=${COMMIT} \
    -X ${VERSION_PKG}.BuildTime=${BUILD_TIME} \
    -X ${VERSION_PKG}.UIBuildHash=${UI_BUILD_HASH}" \
  -o seed ./cmd/seed
