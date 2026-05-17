# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.188.0](https://github.com/krisarmstrong/seed/compare/v0.187.1...v0.188.0) (2026-05-17)


### Features

* **security:** guest network isolation audit ([#397](https://github.com/krisarmstrong/seed/issues/397)) ([#1003](https://github.com/krisarmstrong/seed/issues/1003)) ([81be6a8](https://github.com/krisarmstrong/seed/commit/81be6a8e9342994d4e8756b900b564d2f7102465))


### Bug Fixes

* **auth:** clear stale state, gate setup completion, match SSO contract ([#996](https://github.com/krisarmstrong/seed/issues/996)) ([e6280cf](https://github.com/krisarmstrong/seed/commit/e6280cf4edf9d4084fde127f5e0a76fd8ddc26a8))
* **setup:** enforce password complexity rules with live checklist ([#997](https://github.com/krisarmstrong/seed/issues/997)) ([073eb35](https://github.com/krisarmstrong/seed/commit/073eb35563e30e4f19f8fefc044e1d36347f9614))
* **survey:** client-side validation for ids, coords, floorplan size ([#999](https://github.com/krisarmstrong/seed/issues/999)) ([83ad1e9](https://github.com/krisarmstrong/seed/commit/83ad1e99a8d942c73d776f42f258a57fbf9d1ed7))
* **survey:** persist AirMapper-imported placements + criteria ([#727](https://github.com/krisarmstrong/seed/issues/727)) ([#1000](https://github.com/krisarmstrong/seed/issues/1000)) ([acffbd7](https://github.com/krisarmstrong/seed/commit/acffbd7a0e5adf3fa235a3d6a2b35ab72a8d5010))

## [0.187.1](https://github.com/krisarmstrong/seed/compare/v0.187.0...v0.187.1) (2026-05-16)


### Bug Fixes

* **ui:** gate Cable Test card on link absence ([#740](https://github.com/krisarmstrong/seed/issues/740)) ([fa5e028](https://github.com/krisarmstrong/seed/commit/fa5e0280b2643724bb7a5b1755137495ec517e54))

## [0.186.0](https://github.com/krisarmstrong/seed/compare/v0.185.13...v0.186.0) (2026-05-16)


### Features

* **ci:** restore Windows ARM64 in release matrix ([#944](https://github.com/krisarmstrong/seed/issues/944)) ([de8c595](https://github.com/krisarmstrong/seed/commit/de8c5957160214aba3d1ff2bf143e357ef49044a))
* implement Universal Build Contract for seed ([#946](https://github.com/krisarmstrong/seed/issues/946)) ([0c6870f](https://github.com/krisarmstrong/seed/commit/0c6870f7313e0981ce393194a0dd930c261c0653))


### Bug Fixes

* **ci:** pre-commit hook masks failing tests ([#947](https://github.com/krisarmstrong/seed/issues/947)) ([e8840f8](https://github.com/krisarmstrong/seed/commit/e8840f8db66f13bd07fb24feb4f6680b29689ebd))

## [0.185.13](https://github.com/krisarmstrong/seed/compare/v0.185.12...v0.185.13) (2026-05-15)


### Bug Fixes

* **ci:** stabilize seed release artifact matrix ([cd9b368](https://github.com/krisarmstrong/seed/commit/cd9b368df37ab223921748a435871fb97184a641))

## [0.185.12](https://github.com/krisarmstrong/seed/compare/v0.185.11...v0.185.12) (2026-05-14)


### Bug Fixes

* **ci:** skip seed docker publish without dockerfile ([8e1a075](https://github.com/krisarmstrong/seed/commit/8e1a075ba946c7fd1ea0b2618272e35a19194b56))

## [0.185.11](https://github.com/krisarmstrong/seed/compare/v0.185.10...v0.185.11) (2026-05-14)


### Bug Fixes

* **ci:** align seed setup e2e with current UI ([2505626](https://github.com/krisarmstrong/seed/commit/25056260412df6c420dba4ac4102d7ab3a31ff5b))
* **ci:** align seed validation steps ([34c03bb](https://github.com/krisarmstrong/seed/commit/34c03bb5fe5ccbc61989bcf1ee0e516d59e623a7))
* **ci:** allow MPL npm dependencies ([07f5e24](https://github.com/krisarmstrong/seed/commit/07f5e241da445e10400a30125621de2896e5deca))
* **ci:** build seed amd64 before arm64 deps ([774536b](https://github.com/krisarmstrong/seed/commit/774536b223205131a2b976b57f4623c6f15067ba))
* **ci:** exclude private npm packages from license scan ([ec78b14](https://github.com/krisarmstrong/seed/commit/ec78b14607daf21050ac8751962abcf147e8a46d))
* **ci:** fetch full history for security scans ([f2d00e4](https://github.com/krisarmstrong/seed/commit/f2d00e492814e6f2492e08aad6ca16e77e26fd21))
* **ci:** format tracked go sources only ([bbb36f0](https://github.com/krisarmstrong/seed/commit/bbb36f0ef63ba98539d6037a7c1470d89b64c8ba))
* **ci:** install arm64 kernel headers for seed builds ([e9a72a9](https://github.com/krisarmstrong/seed/commit/e9a72a9a43fefc1df71b08b0f8d22ebc705f9296))
* **ci:** keep seed lighthouse gate focused ([976b507](https://github.com/krisarmstrong/seed/commit/976b507ff1113c2573bed96a70bb423e6cda85ef))
* **ci:** keep seed setup e2e focused ([fdecc42](https://github.com/krisarmstrong/seed/commit/fdecc42b3ec5b48ba7e5f66c583d2371eacde3d6))
* **ci:** prepare assets before backend validation ([42fa3fd](https://github.com/krisarmstrong/seed/commit/42fa3fd57a6016473fe3747a24bfdcc18edc2454))
* **ci:** prepare seed data dir for browser jobs ([aab9b37](https://github.com/krisarmstrong/seed/commit/aab9b378c6586c86e0b0660ac7cd274473cbb777))
* **ci:** repair buildpacks project metadata ([863b7c7](https://github.com/krisarmstrong/seed/commit/863b7c7b4ee52411b49cf2eef79bad7c8a2116b6))
* **ci:** repair label sync workflow ([8711e8a](https://github.com/krisarmstrong/seed/commit/8711e8ab07960cdc6ada9951777c078973fcff61))
* **ci:** report seed gosec findings ([ce9b018](https://github.com/krisarmstrong/seed/commit/ce9b0186cb287e67236b1a42d71d0d1edf87f61a))
* **ci:** resolve seed validation blockers ([d34a4cf](https://github.com/krisarmstrong/seed/commit/d34a4cf96d76d584aefb432f68087e0fee2319f4))
* **ci:** scope seed browser smoke tests ([a7043f2](https://github.com/krisarmstrong/seed/commit/a7043f2207d104699813ac0a68ef90c949e8ab11))
* **ci:** scope seed license checks ([fbb9c7b](https://github.com/krisarmstrong/seed/commit/fbb9c7b34577882231f20ffadf41c667be4c5845))
* **ci:** skip seed docker publish without dockerfile ([fbd0962](https://github.com/krisarmstrong/seed/commit/fbd096287786d75e35c92d97e2da721d014e7989))
* **ci:** stabilize automated validation ([c822698](https://github.com/krisarmstrong/seed/commit/c8226987bce86539e8ffdc9647b0f418db860ece))
* **ci:** stabilize seed backend suite ([c92d728](https://github.com/krisarmstrong/seed/commit/c92d728558a652fea4c3f0294a1116b22b1fdf02))
* **ci:** stabilize seed backend tests ([d4cb236](https://github.com/krisarmstrong/seed/commit/d4cb236bd46eea39d0e2b0b8686101e1f9fa69e8))
* **ci:** stabilize seed reporting gates ([21edd25](https://github.com/krisarmstrong/seed/commit/21edd2572f4ed8548a0892325959c500d595f668))
* **ci:** use compatible labeler action ([92fed97](https://github.com/krisarmstrong/seed/commit/92fed972599e8cba169c5e1f284c2158488bbd04))
* **ci:** use labeler yaml format ([4629c5f](https://github.com/krisarmstrong/seed/commit/4629c5f7b2f36a799622fa4119ffbf59d776d6da))
* **ci:** use target dependencies for seed arm build ([1bf940f](https://github.com/krisarmstrong/seed/commit/1bf940f6cc63e8a758c9a38a03f462bd2693251b))
* **ci:** use writable seed config for browser jobs ([7a7a40b](https://github.com/krisarmstrong/seed/commit/7a7a40b8382d767e9b5fee3ba51f2229aed348be))
* **services:** reject dhcp tests for missing interfaces ([d205b88](https://github.com/krisarmstrong/seed/commit/d205b88f199ac8afb5848b7dfc095d8736d9b24f))

## [0.12.1](https://github.com/krisarmstrong/seed/compare/v0.12.0...v0.12.1) (2025-12-09)

### Bug Fixes

- **ci:** move libpcap-dev install to backend job for golangci-lint
  ([298d305](https://github.com/krisarmstrong/seed/commit/298d30511d4faaf900e0caf43fb3511eb75a20e6))
- **ci:** remove 'shadow' linter from .golangci.yml
  ([24ed597](https://github.com/krisarmstrong/seed/commit/24ed597ca9cb01d5d266f3408437243635eaa060))
- **ci:** remove accidental automerge.yml
  ([33a2b3f](https://github.com/krisarmstrong/seed/commit/33a2b3f76eb6c0e7d77b04da4c469bd5bc62b89b))
- **ci:** update golangci-lint version and format code
  ([5e58e96](https://github.com/krisarmstrong/seed/commit/5e58e964055fc884a1064cec71e051f060214d4c))
- **ci:** upgrade golangci-lint-action to v6
  ([2496c06](https://github.com/krisarmstrong/seed/commit/2496c060114d726daba19579034ec335159e6007))
- **ci:** use goinstall for golangci-lint to resolve go version incompatibility
  ([1ecd63f](https://github.com/krisarmstrong/seed/commit/1ecd63f988c010966931598c6f7ac55c6e82da70))
- **frontend:** debug eslint tsconfig path
  ([c86ab94](https://github.com/krisarmstrong/seed/commit/c86ab9493bec0d525affb96a05340147d6327a65))
- **frontend:** remove parserOptions.project from eslint config
  ([5a4d710](https://github.com/krisarmstrong/seed/commit/5a4d710f6c34fcc8343ff9838b52345e3d19bfd6))
- make DNS tester thread-safe for race tests
  ([31d74bf](https://github.com/krisarmstrong/seed/commit/31d74bfec7793b26d74d9bc02af616a9afa7980d))
- **release:** remove deprecated inputs from release-please config
  ([a602821](https://github.com/krisarmstrong/seed/commit/a6028217a9036068516b4f34ca468665a66957e8))

## [0.12.0](https://github.com/krisarmstrong/seed/compare/v0.11.9...v0.12.0) (2025-12-08)

### Features

- **release:** add debian packaging and systemd service
  ([bd1ed1a](https://github.com/krisarmstrong/seed/commit/bd1ed1a38430ee3344bc390850c90468791d7ba3))
- **release:** add docker containerization
  ([0b865ce](https://github.com/krisarmstrong/seed/commit/0b865cee356d3cc491247517567dcf918d0f9e5e))
- **release:** add fedora rpm packaging
  ([9353217](https://github.com/krisarmstrong/seed/commit/93532173f2fbf112cfbb0e5cf1dbacdc48d7383f))
- **web:** upgrade react to v19
  ([dec0cb9](https://github.com/krisarmstrong/seed/commit/dec0cb9deaa215cbc8b332b5760e9f5bf9198951))

### Bug Fixes

- **ci:** explicitly pass GITHUB_TOKEN to release-please
  ([f1f183e](https://github.com/krisarmstrong/seed/commit/f1f183e15108495e1cf15f93817ba1c5ae2075ef))
- **ci:** update golangci-lint to a compatible version
  ([8f97797](https://github.com/krisarmstrong/seed/commit/8f977974f47c6dc084177dd17c6b5e3c52c03c5c))
- **ci:** use PAT for release-please
  ([c0da65e](https://github.com/krisarmstrong/seed/commit/c0da65eeba1543de7a6bb58e0e2c8bf8a8943856))
- **frontend:** correct eslint tsconfig path
  ([31fe551](https://github.com/krisarmstrong/seed/commit/31fe55141d4932ae0284ace5cb169eebe60e547f))

## [Unreleased]

## [0.1.0] - 2025-12-03

### Added

#### Backend (Go)

- HTTP/HTTPS server with auto-generated self-signed TLS certificates
- WebSocket server for real-time card updates with heartbeat/ping-pong
- JWT authentication with bcrypt password hashing
- Network interface detection and management
- Configuration loading from YAML with sensible defaults
- Graceful shutdown handling

#### Frontend (React + TypeScript)

- WebSocket hook with auto-reconnect and connection status
- Authentication hook with login/logout flow
- Card component system with status indicators (green/yellow/red)
- 8 diagnostic cards: Link, Cable, VLAN, Switch, Wi-Fi, DHCP, DNS, Gateway
- Login form with default credentials hint
- Connection status indicator in header
- Responsive grid layout (mobile-friendly)
- WiFi Vigilante color scheme (dark mode default)

#### Infrastructure

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

For detailed commit history, see: https://github.com/krisarmstrong/seed/commits/main
