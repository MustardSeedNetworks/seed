# Development Guide

## Prerequisites

- Go 1.27.0+
- Node.js 26.8.1+
- npm 12.0.2+

## Setup

```bash
# Clone and setup
git clone <repo-url>
cd seed

# Backend
make build

# Frontend
cd ui
npm install
```

## Development

```bash
# Backend
make dev          # Run backend with hot reload
make test         # Run tests
make lint         # Run linters

# Frontend (cd ui/)
npm run dev       # Start dev server (port 5173)
npm run test      # Run unit tests
npm run test:e2e  # Run E2E tests
npm run test:storybook # Build and test tagged Storybook interactions and accessibility
npm run lint      # Run Biome linter
```

## Project Structure

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Common Tasks

| Task | Command |
| ---- | ------- |
| Build all | `make build` |
| Run tests | `make test` |
| Lint | `make lint` |
| Clean | `make clean` |
| Dev server | `cd ui && npm run dev` |
