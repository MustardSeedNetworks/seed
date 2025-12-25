#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

# Go linting and tests
golangci-lint run ./...
go test -race -coverprofile=coverage.out ./...

# Frontend checks
cd web
npm run lint
npm run build
