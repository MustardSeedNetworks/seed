# Contributing to NetScope

Thank you for your interest in contributing to NetScope! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and professional in all interactions.

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 22+
- libpcap-dev (Linux) or libpcap (macOS)
- Git

### Development Setup

```bash
# Clone the repository
git clone https://github.com/krisarmstrong/netscope.git
cd netscope

# Install Go dependencies
go mod download

# Install frontend dependencies
cd web && npm install && cd ..

# Run tests
make test

# Run linting
make lint
```

## Development Workflow

### Branch Naming

Use descriptive branch names with prefixes:

- `feat/` - New features (e.g., `feat/wifi-card`)
- `fix/` - Bug fixes (e.g., `fix/dhcp-timeout`)
- `docs/` - Documentation (e.g., `docs/api-reference`)
- `chore/` - Maintenance (e.g., `chore/update-deps`)
- `refactor/` - Code refactoring (e.g., `refactor/websocket-handler`)

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/). All commits must follow this format:

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `style` - Code style changes (formatting)
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `test` - Adding or updating tests
- `chore` - Maintenance tasks
- `ci` - CI/CD changes
- `build` - Build system changes

**Examples:**
```
feat(dhcp): add phase timing breakdown
fix(websocket): resolve connection drop on idle
docs: update installation instructions
chore(deps): upgrade gopacket to v1.2.0
```

### Pull Request Process

1. **Create an issue first** - Discuss the change before implementing
2. **Fork and branch** - Create a feature branch from `main`
3. **Write tests** - Ensure adequate test coverage
4. **Update docs** - Update relevant documentation
5. **Run checks** - Ensure all tests and linting pass
6. **Submit PR** - Reference the related issue

### PR Title Format

Use the same format as commit messages:
```
feat(dhcp): add phase timing breakdown (#123)
```

## Code Standards

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write table-driven tests
- Document exported functions

### TypeScript/React

- Follow the existing code style
- Use TypeScript strict mode
- Write unit tests with Vitest
- Use functional components with hooks

### General

- Keep functions small and focused
- Write self-documenting code
- Add comments for complex logic
- Handle errors explicitly
- No hardcoded secrets or credentials

## Testing

### Running Tests

```bash
# All tests
make test

# Go tests with coverage
go test -coverprofile=coverage.out ./...

# Frontend tests
cd web && npm test

# E2E tests (requires running server)
make test-e2e
```

### Test Requirements

- Unit tests for business logic
- Integration tests for API endpoints
- Aim for >80% coverage on new code

## Issue Guidelines

### Bug Reports

Include:
- NetScope version
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs

### Feature Requests

Include:
- Clear description of the feature
- Use case / problem it solves
- Proposed implementation (if any)

## Questions?

- Check existing issues and documentation
- Open a discussion for general questions
- File an issue for bugs or features

Thank you for contributing!
