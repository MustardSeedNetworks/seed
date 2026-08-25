# DHCP OFFER fixtures

Complete Ethernet/IPv4/UDP/DHCP frames, replayed through `rogue.go`'s real
parse path by `rogue_fixture_test.go`. No network, no privileges, no pcap
handle.

| fixture | what it is |
| --- | --- |
| `dhcp_offer_authorized.bin` | OFFER from `192.0.2.1`, option 54 present |
| `dhcp_offer_rogue.bin` | OFFER from `192.0.2.66`, option 54 present |
| `dhcp_offer_no_serverid.bin` | OFFER from `192.0.2.77` with **no** option 54, so the detector must fall back to the source IP |
| `dhcp_offer_truncated.bin` | first 20 bytes only — no DHCP layer decodes |

All addresses are from `192.0.2.0/24` (RFC 5737 TEST-NET-1) and all MACs are
locally administered, so nothing here can collide with a real network.

## Regenerating

The frames are **synthesised**, not captured. #498 proposed standing up a
dnsmasq lab once to capture them; building them instead makes the bytes a
function of code in the repo, so a reviewer can see exactly what each fixture
asserts and regenerate them on any machine:

```bash
go run ./internal/dhcp/testdata/generate.go
```

The file carries `//go:build ignore`, so it is not part of the package build.

## Adding a scenario

Add an entry to the fixture table in `generate.go`, re-run it, and add the
assertion to `rogue_fixture_test.go`. Keep the addresses inside TEST-NET-1.
