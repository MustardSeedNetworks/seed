# S4-4: Seed discovers the niac hospital pack (2026-09-06)

Fleet rule 8: each product validates the others, and the EtherScope nXG with
Link-Live arbitrates. This is the first run of seed against a niac pack.

## Setup

| Item | Value |
| --- | --- |
| Seed | 0.214.6 on CT313 (Proxmox pvm01), Pro trial |
| Pack | niac hospital, generated through `POST /api/v1/scenario/generate` on niac 0.95.9, 78 devices / 88 links |
| Path | CT313 `eth0.200` = 10.254.200.252/24, routes to 10.51.0.0/16, 203.0.113.0/24 and 8.8.8.0/24 via the pack's edge router |
| Arbiter | Link-Live analysis `6a9d48789e0a52ab615b1d33` (EtherScope nXG, zero findings against the same pack) |

Seed's own discovery scan found one device (the edge router) because it only
sweeps the interface's subnet (seed#2449). The 74 SNMP devices were added as
polling targets through the API, with one credential for the pack community.

## Result

| Measure | Authored | Link-Live | Seed |
| --- | --- | --- | --- |
| Devices named | 78 | 71 | 74 (every SNMP device; the 4 NBSTAT-only nurse stations are not reachable to seed) |
| Extra devices | 0 | 0 | 0 |
| Links | 88 | 88 | 62 (all 26 missing are switch-to-endpoint; seed has no FDB edges, seed#2454) |
| Device type | per profile | Switch / Router / WLC / AP / Server / Host | `unknown` for all 74 (seed#2456 and the leading-dot fix) |
| Interface names on links | both ends | both ends | far end only; near end is `ifIndex-N` (seed#2455) |

## Defects filed on seed

| Issue | What |
| --- | --- |
| #2449 | Discovery cannot sweep off-subnet ranges; routed devices are hand-added polling targets |
| #2450 | Every personal access token is rejected by the JWT middleware after the PAT layer accepts it |
| #2452 | Poller loads targets once at start; API-added targets are never polled until restart |
| #2453 | Collectors fail with SQLITE_BUSY when 74 targets poll at once |
| #2454 | No switch-to-endpoint links: fdb observations are never reconciled into edges |
| #2455 | LLDP edges label the local port `ifIndex-N` |
| #2456 | Device type is a vendor label, never a role |

No defect was found on niac from this run: every device seed reached answered
with the authored sysName, and the LLDP edges seed built all match authored
trunk links.

## Reproduce

Harness and comparator live in the session scratchpad for now
(`s4-4-seed-vs-niac.py`, `s4-4-compare.py`); niac plan P3-2 turns them into
the acceptance suite. The pack YAML is the file `acceptance.sh` generated on
CT304; the Link-Live hosts come from `GET /v1/admin/hosts?query={"analysisId":…}`.
