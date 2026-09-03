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
| macOS | routing socket (`route.FetchRIB`) | **native** for IPv4; IPv6 still shells `ndp`, see below |
| Windows | `GetIpNetTable2` | **not bound** in `golang.org/x/sys/windows` |

**macOS.** The routing socket does carry the neighbour cache, and the read has
two halves that both have to be right. The family must be `AF_INET`, and the
`NET_RT_FLAGS` argument must be `RTF_LLINFO` — that sysctl filters on the flag
it is handed, so asking for `0` asks for routes with no flags and the kernel
returns zero bytes. Measured on Darwin 27.0.0, 2026-09-03, parsed with
`route.ParseRIB`:

```text
AF_INET   NET_RT_FLAGS arg=0            bytes=0
AF_INET   NET_RT_FLAGS arg=RTF_LLINFO   bytes=1292  9 entries, byte-identical to `arp -an`
AF_INET6  NET_RT_FLAGS arg=RTF_LLINFO   bytes=3032  18 entries, 10 with a resolved link address
```

That is why `arp_darwin.go` is native as of #2272.

**The earlier measurements in this ADR were wrong, and the reason matters.**
They were taken from a process the `go` tool had spawned, and macOS answers such
a process with a filtered neighbour cache — zero bytes for IPv4, and every IPv6
link-layer address replaced by the placeholder `02:00:00:00:00:00`. Those
placeholders were read here as a broken parser. The same binary, run from a
shell rather than by `go run` or `go test`, reads the table:

```text
go run .        AF_INET NET_RT_FLAGS RTF_LLINFO -> 0 bytes
./probe         AF_INET NET_RT_FLAGS RTF_LLINFO -> 1292 bytes
sh -c ./probe   AF_INET NET_RT_FLAGS RTF_LLINFO -> 1292 bytes
env -i ./probe  AF_INET NET_RT_FLAGS RTF_LLINFO -> 1292 bytes
```

So a measurement taken under `go test` is not evidence about what the shipped
daemon can see.

**macOS IPv6.** `ndp_darwin.go` still shells out, and the measurement above says
it probably need not. Reconciling the RIB's 10 resolved addresses against
`ndp -an`'s 17 rows is #2336, not a claim this ADR should make either way.

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

Why `/usr/sbin/arp -an` returns zero bytes when spawned from a Go process is
still not fully explained — the gate is per-process and the discriminator is the
parent, but no TCC denial was logged. It is no longer load-bearing: nothing in
the tree shells `arp`.

## Consequences

- New platform code calls the API. A shell-out needs the measurement in the
  comment, the way the two above do.
- The gap list is now explicit, so "we shell out here" is a recorded decision
  rather than an accident, and #749's runtime capability report has something
  concrete to describe.
- Two candidates for native replacement, neither free: Windows `GetIpNetTable2`
  behind a hand-rolled binding, and Linux DHCP renewal over D-Bus.
