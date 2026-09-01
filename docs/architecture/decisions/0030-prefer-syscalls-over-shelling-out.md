# 0030 — Prefer syscalls and OS APIs over shelling out

Status: Accepted
Date: 2026-09-01

## Context

Seed reads and changes host network state on three platforms. There are two
ways to do that: call the OS API, or run the command-line tool that wraps it.

Shelling out is worse in ways that have already cost us:

- **It breaks when the tool is removed.** #2031: macOS Wi-Fi scanning shelled
  `airport`, Apple deleted the binary, and the feature died silently.
- **It breaks when the tool is absent.** `dhcp_linux.go` shells `dhclient` to
  probe a lease. Ubuntu 26.04 ships neither `dhclient` nor NetworkManager, so
  that path cannot work on a current Ubuntu.
- **Output is a UI, not an interface.** It is localised, reformatted between
  releases, and unpadded in ways a parser has to know about (`ndp` prints
  `3a:f0:5a:8b:c5:4`, which `net.ParseMAC` rejects).
- **It costs a process per call** and inherits whatever environment the daemon
  happens to have.

So: **use the OS API unless it demonstrably does not work.** A shell-out is a
last resort that must be justified in the code, with evidence, at the call site.

## Decision

Native first. Where a shell-out remains, the comment at the call site says what
was measured and why the API path was rejected.

The measurements below were taken on 2026-09-01 and are the justification for
every shell-out currently in the tree.

### Neighbour caches (ARP / NDP)

| Platform | API path | Verdict |
| --- | --- | --- |
| Linux | netlink via `vishvananda/netlink` | **native** — already used, no shell |
| macOS | routing socket (`route.FetchRIB`) | **does not work**, see below |
| Windows | `GetIpNetTable2` | **not bound** in `golang.org/x/sys/windows` |

**macOS.** The routing socket does not carry the neighbour cache on Darwin 27.
Fetched every way the API allows and parsed with `route.ParseRIB` — the typed
parser, not the hand-rolled byte offsets that previously mangled the addresses:

```text
AF_UNSPEC RIBTypeRoute arg=0             bytes=11004  msgs=62   llinfo v4=0 v6=4
AF_INET   RIBTypeRoute arg=0             bytes=1344   msgs=9    llinfo v4=0 v6=0
AF_INET6  RIBTypeRoute arg=0             bytes=9660   msgs=53   llinfo v4=0 v6=4
AF_UNSPEC NET_RT_FLAGS RTF_LLINFO        bytes=2024   msgs=12   llinfo v4=0 v6=4
```

Four entries, every one with a placeholder `02:00:00:00:00:00` link address,
including duplicates — against **12 resolved unicast neighbours** that `ndp -an`
reports on the same machine at the same moment. IPv4 yields nothing at all.

So `ndp -an` is not a shortcut here; it is the only source that has the data.
That is why `ndp_darwin.go` shells out, and the comment there says so.

**Windows.** `x/sys/windows` binds `GetIpForwardTable`, `GetIpInterfaceTable`
and `GetUnicastIpAddressTable`, but not `GetIpNetTable2`. Using it means a
hand-rolled `LazyDLL` binding — which existed once, had no callers, and carried
two `possible misuse of unsafe.Pointer` vet findings before being removed
under #2174. Reintroducing it is defensible, but it is a deliberate piece of work with
a real risk surface, not a cleanup. `Get-NetNeighbor` with `ConvertTo-Csv` is
the current path; it is structured output rather than localised prose, which is
why it was chosen over `netsh`.

### DHCP lease renewal

No platform offers this through a Go-bindable API today:

- **Linux** — systemd-networkd exposes it on D-Bus, which would add a dependency
  and only covers hosts that systemd-networkd manages. The tool is detected per
  host (#170) rather than assumed.
- **macOS** — the SystemConfiguration framework requires cgo, which the
  discovery packages deliberately avoid (see `CGO_BUILD_STRATEGY.md`).
- **Windows** — `DhcpNotifyConfigChange` is not bound in `x/sys/windows`.

### Unresolved

`arp -an` returns **zero bytes when spawned from a Go process** and ten lines
from an interactive shell — same binary, back to back, with sandboxing disabled
and from a compiled binary rather than `go run`. `ndp -an` works from that same
process. Tracked in #2272; until it is understood, macOS has no working path to
the IPv4 neighbour cache at all, native or otherwise.

## Consequences

- New platform code calls the API. A shell-out needs the measurement in the
  comment, the way the two above do.
- The gap list is now explicit, so "we shell out here" is a recorded decision
  rather than an accident, and #749's runtime capability report has something
  concrete to describe.
- Two candidates for native replacement, neither free: Windows `GetIpNetTable2`
  behind a hand-rolled binding, and Linux DHCP renewal over D-Bus.
