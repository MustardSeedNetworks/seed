# Build Commands - The Seed

## Quick Reference

| Command | Purpose |
|---------|---------|
| `make build` | Build all (Go backend) |
| `make test` | Run all Go tests |
| `make lint` | Run golangci-lint |
| `make clean` | Clean build artifacts |

## UI Commands

```bash
cd ui
npm install        # Install dependencies
npm run dev        # Development server (hot reload)
npm run build      # Production build
npm run lint       # Biome linting
npm run lint:fix   # Auto-fix lint issues
npm run test       # Run Vitest tests
```

## Go Commands

```bash
go build ./...     # Build all packages
go test ./...      # Run all tests
go test -v ./internal/discovery/...  # Test specific package
```

## Pre-commit

The project has pre-commit hooks that run:
- Secret detection (gitleaks)
- Go linting
- Sensitive file checks
