# CGO build strategy — libpcap blast radius & build speed

**Status:** Findings + plan — 2026-06-01
**Scope:** Why seed links libpcap (CGO), where that cost lands, and how to keep
builds fast. Records the four findings from the 2026-06-01 build audit and the
plan to shrink the CGO blast radius behind a `Capture` port (executes with the
Phase 6 `discovery`/`sap` extraction — see `PHASE3_RECONCILE_PROPOSAL.md` §5).

---

## 1. Why CGO at all

seed has **no `#cgo` directives of its own**. CGO is pulled in transitively by
`github.com/gopacket/gopacket/pcap`, whose `pcap_unix.go` / `pcap_darwin.go`
carry `#cgo LDFLAGS: -lpcap`. Live packet capture (DHCP rogue detection, CDP/EDP/
LLDP discovery, VLAN traffic sampling) needs libpcap, so `CGO_ENABLED=1` is
required for those features. The Windows build is already `CGO_ENABLED=0`
(`mk/build.mk`) — capture is a no-op there.

> Contrast: **stem** ships `CGO_ENABLED=0` (pure-Go gopacket, no libpcap). seed
> could in principle do the same for everything *except* live capture — see §4.

## 2. The four findings (2026-06-01 audit)

| # | Finding | Disposition |
|---|---|---|
| 1 | **Local incremental builds are already fast** — once the build cache is warm, a rebuild after editing a CGO-tainted package is **<0.1s**. The "57s" seen during the audit was a *one-time cold-cache* CGO compile (libpcap cgo stubs + first external link), caused by thrashing the cache with killed concurrent builds. | No action — don't `go clean -cache`; don't run many concurrent full builds. |
| 2 | **CI recompiles from cold on every run** — the `setup-go` composite had `cache: false` and there was **no `actions/cache`** step, while CI overrides `GOCACHE`/`GOMODCACHE` to workspace paths that were never persisted. Every job paid the full CGO+libpcap compile/link each run. | **Fixed** — `setup-go` now persists the build + module caches (path-discriminated, go.sum + per-commit keyed). |
| 3 | **CGO blast radius is large** — `gopacket/pcap` is imported by `dhcp`, `services/discovery`, and `diagnostics/vlan`, and fans up to **11 packages** (incl. `api`, `diagnostics`, `pipeline`, `security`, `cmd/seed`). Every CGO-tainted *test binary* re-links libpcap, so a cold `go test ./...` does many slow external links. | **Planned** — `Capture` port (§3 below). Executes with Phase 6 because `discovery` (the dominant importer) is a HOT zone; until it moves, the taint keeps flowing up regardless of `dhcp`/`vlan`. |
| 4 | **`ld: warning: ignoring duplicate libraries: '-lpcap'`** — gopacket declares `#cgo darwin LDFLAGS: -lpcap` in **both** `pcap_darwin.go` and `pcap_unix.go`, so macOS links it twice. | Upstream, darwin-only, cosmetic (the linker ignores the dup). Not fixable without forking gopacket — **ignore**. |

## 3. Plan: live capture behind a `Capture` port

The capability-first way to shrink the blast radius (finding #3): make libpcap an
*adapter*, not a deep dependency. Today every package that does discovery/DHCP/
VLAN capture imports `gopacket/pcap` directly, so CGO taints them and everything
above them. Instead:

```
internal/<feature>/ports.go      Capture port (interface): OpenLive, ReadPacket, Close, ...
internal/capture/pcap/        the ONLY package importing gopacket/pcap (CGO lives here)
internal/capture/nullcapture/ CGO-free stub (returns "capture unavailable") for CGO_ENABLED=0
internal/app/                      wires the real pcap adapter into the features
```

Result: only `internal/capture/pcap` and the final `cmd/seed` binary need
CGO. The domain packages (`dhcp`, `discovery`, `vlan`, and everything that
imports them — `api`, `diagnostics`, `pipeline`, `security`) compile and **test** with
`CGO_ENABLED=0`, so `go test ./...` stops re-linking libpcap across ~11 packages.

### Sequencing (why not now)

- The dominant importer is `services/discovery` (the 24.5K-line monolith), a
  **HOT zone** and explicitly **Phase 6** per `PHASE3_RECONCILE_PROPOSAL.md` §5 and
  the CLAUDE.md "do not touch" list. Touching it now risks colliding with the
  active discovery/NMS workstream.
- Extracting only the cold importers (`dhcp`, `vlan`) would establish the
  `Capture` port + a dedicated `internal/capture` package but yield **little build-speed
  payoff**, because `discovery` keeps libpcap flowing up into `api`/`services`/
  `cmd`. The test-speed win only materializes once *all three* importers sit
  behind the port.
- Therefore: design the `Capture` port as a standalone `internal/capture` package
  (no feature migration required to establish the precedent), and complete the pcap move as part of the **Phase 6 discovery split**, where
  `dhcp`/`vlan`/`discovery` all migrate together.

### Acceptance criteria (when executed)

- [ ] Exactly one package imports `gopacket/pcap` (`internal/capture/pcap`).
- [ ] `go build ./... ` with `CGO_ENABLED=0` succeeds (uses the null adapter);
      `CGO_ENABLED=1` builds keep live capture.
- [ ] `go vet`/`go test` for `dhcp`/`discovery`/`vlan` run `CGO_ENABLED=0`.
- [ ] Golden HTTP suite unchanged; capture features behave identically with the
      real adapter.
- [ ] `depguard`: `gopacket/pcap` denied outside `internal/capture/pcap`.

## 4. CI caching (finding #2 — shipped)

`.github/actions/setup-go/action.yml` now:

1. resolves the active `GOCACHE`/`GOMODCACHE` (honoring per-job overrides),
2. persists both via `actions/cache@v4.3.0` using the official Go pattern —
   key `${os}-go-${pathtag}-${hash(go.sum)}` with a `${os}-go-${pathtag}-`
   `restore-keys` prefix fallback,
3. still runs `go mod download`.

The expensive, code-independent CGO objects (libpcap cgo stubs + stdlib) depend
on toolchain + `go.sum`, not app code, so a `go.sum`-keyed cache keeps them warm
across runs — only changed packages recompile (cheap pure-Go work). A `go.sum`
key (rather than per-commit) avoids churning hundreds-of-MB cache entries that
would accelerate LRU eviction. The `pathtag` prevents the workspace-`GOCACHE`
jobs from colliding with default-path jobs on a shared key (actions/cache
restores to the saved path, so a mismatch would leave a job cold).

## 5. Operational notes

- **Don't `go clean -cache`** and **don't run several `go build ./...`
  concurrently** — both force the one-time cold CGO cost (finding #1).
- Cross-compilation already wires the C cross-toolchain (`CC=x86_64-linux-gnu-gcc`,
  `aarch64-linux-gnu-gcc`) in `mk/build.mk`; the canonical release path is
  goreleaser-cross.
- libpcap is linked **dynamically** (default) — keep it that way; static linking
  adds build time for no benefit here.
