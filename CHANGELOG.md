# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.12.1](https://github.com/krisarmstrong/netscope/compare/v0.12.0...v0.12.1) (2025-12-09)


### Bug Fixes

* **ci:** move libpcap-dev install to backend job for golangci-lint ([298d305](https://github.com/krisarmstrong/netscope/commit/298d30511d4faaf900e0caf43fb3511eb75a20e6))
* **ci:** remove 'shadow' linter from .golangci.yml ([24ed597](https://github.com/krisarmstrong/netscope/commit/24ed597ca9cb01d5d266f3408437243635eaa060))
* **ci:** remove accidental automerge.yml ([33a2b3f](https://github.com/krisarmstrong/netscope/commit/33a2b3f76eb6c0e7d77b04da4c469bd5bc62b89b))
* **ci:** update golangci-lint version and format code ([5e58e96](https://github.com/krisarmstrong/netscope/commit/5e58e964055fc884a1064cec71e051f060214d4c))
* **ci:** upgrade golangci-lint-action to v6 ([2496c06](https://github.com/krisarmstrong/netscope/commit/2496c060114d726daba19579034ec335159e6007))
* **ci:** use goinstall for golangci-lint to resolve go version incompatibility ([1ecd63f](https://github.com/krisarmstrong/netscope/commit/1ecd63f988c010966931598c6f7ac55c6e82da70))
* **frontend:** debug eslint tsconfig path ([c86ab94](https://github.com/krisarmstrong/netscope/commit/c86ab9493bec0d525affb96a05340147d6327a65))
* **frontend:** remove parserOptions.project from eslint config ([5a4d710](https://github.com/krisarmstrong/netscope/commit/5a4d710f6c34fcc8343ff9838b52345e3d19bfd6))
* make DNS tester thread-safe for race tests ([31d74bf](https://github.com/krisarmstrong/netscope/commit/31d74bfec7793b26d74d9bc02af616a9afa7980d))
* **release:** remove deprecated inputs from release-please config ([a602821](https://github.com/krisarmstrong/netscope/commit/a6028217a9036068516b4f34ca468665a66957e8))

## [0.12.0](https://github.com/krisarmstrong/netscope/compare/v0.11.9...v0.12.0) (2025-12-08)

### Features

- **release:** add debian packaging and systemd service ([bd1ed1a](https://github.com/krisarmstrong/netscope/commit/bd1ed1a38430ee3344bc390850c90468791d7ba3))
- **release:** add docker containerization ([0b865ce](https://github.com/krisarmstrong/netscope/commit/0b865cee356d3cc491247517567dcf918d0f9e5e))
- **release:** add fedora rpm packaging ([9353217](https://github.com/krisarmstrong/netscope/commit/93532173f2fbf112cfbb0e5cf1dbacdc48d7383f))
- **web:** upgrade react to v19 ([dec0cb9](https://github.com/krisarmstrong/netscope/commit/dec0cb9deaa215cbc8b332b5760e9f5bf9198951))

### Bug Fixes

- **ci:** explicitly pass GITHUB_TOKEN to release-please ([f1f183e](https://github.com/krisarmstrong/netscope/commit/f1f183e15108495e1cf15f93817ba1c5ae2075ef))
- **ci:** update golangci-lint to a compatible version ([8f97797](https://github.com/krisarmstrong/netscope/commit/8f977974f47c6dc084177dd17c6b5e3c52c03c5c))
- **ci:** use PAT for release-please ([c0da65e](https://github.com/krisarmstrong/netscope/commit/c0da65eeba1543de7a6bb58e0e2c8bf8a8943856))
- **frontend:** correct eslint tsconfig path ([31fe551](https://github.com/krisarmstrong/netscope/commit/31fe55141d4932ae0284ace5cb169eebe60e547f))

## [Unreleased]

## [0.1.0] - 2025-12-03

### Added

**Backend (Go)**

- HTTP/HTTPS server with auto-generated self-signed TLS certificates
- WebSocket server for real-time card updates with heartbeat/ping-pong
- JWT authentication with bcrypt password hashing
- Network interface detection and management
- Configuration loading from YAML with sensible defaults
- Graceful shutdown handling

**Frontend (React + TypeScript)**

- WebSocket hook with auto-reconnect and connection status
- Authentication hook with login/logout flow
- Card component system with status indicators (green/yellow/red)
- 8 diagnostic cards: Link, Cable, VLAN, Switch, Wi-Fi, DHCP, DNS, Gateway
- Login form with default credentials hint
- Connection status indicator in header
- Responsive grid layout (mobile-friendly)
- WiFi Vigilante color scheme (dark mode default)

**Infrastructure**

- CI/CD pipeline with GitHub Actions
- Security scanning with CodeQL
- Dependabot for automated dependency updates
- Conventional commits enforcement
- BSL 1.1 license (converts to Apache 2.0 on 2029-12-01)

---

## [0.0.0] - 2025-12-02

### Added

- Initial project structure
- Project plan and architecture documentation

---

For detailed commit history, see: https://github.com/krisarmstrong/netscope/commits/main
