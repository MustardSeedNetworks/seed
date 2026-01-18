# Tests

This directory contains integration, smoke, and load tests for The Seed.

## Directory Structure

```
tests/
├── integration/    # Full system integration tests
├── smoke/          # Quick sanity checks for deployments
└── load/           # Performance and load testing
```

## Running Tests

```bash
# Run all Go tests (unit + integration)
make test

# Run smoke tests only
make test-smoke

# Run load tests
make test-load

# Run E2E tests (Playwright)
cd ui && npm run e2e
```

## Test Categories

### Integration Tests (`integration/`)
Tests that verify multiple components work together correctly.
- Database integration
- API endpoint testing
- Service-to-service communication

### Smoke Tests (`smoke/`)
Fast sanity checks to verify basic functionality after deployment.
- Health endpoints respond
- Basic CRUD operations work
- Critical user flows function

### Load Tests (`load/`)
Performance testing to verify the system handles expected load.
- Concurrent user simulation
- API throughput benchmarks
- Memory/CPU profiling under load

## Unit Tests

Unit tests are colocated with source code in `*_test.go` files under `internal/`.
