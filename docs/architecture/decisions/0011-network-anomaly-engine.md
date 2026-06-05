# ADR-0011: Network-wide anomaly engine (one typed stream, data-driven catalog)

**Status:** Accepted — 2026-06-05 · implementation pending (Wi-Fi feature W4)

## Context

The owner wants proactive problem/anomaly detection — starting with Wi-Fi (rogue/evil-twin
APs, vendor mismatch within an SSID, security mismatch, Pineapple/KARMA) but explicitly
spanning much more: a NetAlly-style catalog that covers Wi-Fi **and** wired/SNMP network
health (CPU/mem/disk, interface errors/FCS/discards, half-duplex, spanning-tree change,
duplicate IP, bad mask, SNMPv3-answering-v1/v2) **and** connectivity-test outcomes. The
detections must be *guided*: each carries a precise description (cite IEEE/802.11 where it
applies), a concrete fix recommendation, and — where one observation is ambiguous — a way to
**narrow the diagnosis**, either by auto-running a deeper test or prompting the user.

Two failure modes to avoid:
1. **Three sibling buckets** ("Network / Wi-Fi / Security Anomalies"). They overlap so heavily
   (a Pineapple is all three) that users hunt across lists for one problem, and detections get
   split. The owner agreed this is "a pain in the ass."
2. **Hard-coding detections per subsystem.** That fragments the catalog, blocks cross-source
   correlation (e.g. a captured Wi-Fi BSSID whose MAC also appears in the wired ARP/switch
   table = a rogue AP bridged onto the LAN), and makes the copy/thresholds un-tunable.

## Decision

One **general `internal/anomaly` engine**, network-wide — NOT a Wi-Fi-only or per-subsystem
detector. "Network Anomalies" is the umbrella; Wi-Fi is its first rule *source*.

- **One typed instance.** `Anomaly{ id, defKey, category, severity, subjectRef (SSID/BSSID/
  client/device/interface), title, detail, evidence (measured values), firstSeen, lastSeen,
  count }`. `category` encodes the *domain* (e.g. `security`, `rf`, `roaming`, `capacity`,
  `nethealth`, `authorization`) — there is exactly one stream, filterable by category + severity.
- **Data-driven catalog (`AnomalyDef`).** Each anomaly *type* is a catalog entry, separate from
  its detections: `{ id, category, defaultSeverity, standards []string (cite IEEE/802.11, e.g.
  "IEEE 802.11w-2009"), title, description, impact, recommendation, followUps []FollowUp }`.
  Copy is **authored originally** (the competitor's analysis prose is copyrighted and must not be
  pasted into this source-available repo); the external list is used only as a checklist of what
  to detect. The catalog lives as data so copy/thresholds are tunable without code changes.
- **Capability-gated follow-ups (ADR-0002).** A `FollowUp{kind:"auto"|"prompt", label, action}`
  runs as an automatic deeper test where the platform/adapter registers the capability (Linux
  active diagnostics — nl80211 station dump, association/auth probe, RTS/CTS hidden-node test,
  PMF/deauth-response check, throughput retest) and **degrades to a guided user prompt** where it
  does not. One catalog, two execution modes, chosen per registered capability.
- **Pluggable rule sources via the event bus (ADR-0004).** Sources (Wi-Fi capture first; wired
  discovery / SNMP / AutoTest later) evaluate their own domain data and emit detections; the
  engine dedups, correlates (incl. cross-source, e.g. Wi-Fi↔wired MAC), escalates severity,
  applies TTL/clear, and persists. The engine has **no dependency on any one source** — it is
  unit-tested against synthetic detections.
- **Feeds existing surfaces.** Anomalies map onto the existing severity vocabulary + Alerts model
  and publish on the event bus for live UI. Wire DTOs are camelCase (ADR-0010), code-first schema
  (ADR-0008). Where a source is Pro (Wi-Fi management capture / association forensics), its
  detections are gated by `requireFeature`; the engine itself is tier-neutral.

## Consequences

- One stream scales to a large, growing catalog with no UI sprawl; the "Network Anomalies"
  umbrella can absorb wired + Wi-Fi + security + test sources over time.
- Cross-source correlation becomes possible (rogue-AP-on-LAN, IP/MAC conflicts) because all
  detections land in one engine.
- Reuses Alerts/severity/events rather than inventing a parallel notification path.
- The catalog being *data* lets us tune descriptions, IEEE citations, thresholds, and severities
  without redeploying logic, and makes the catalog reviewable in one place.
- New responsibilities to design carefully: detection **dedup + TTL + clear**, severity
  escalation on recurrence, and the follow-up **capability registration** for active diagnostics.
- The guided-remediation promise depends on the capability registry being wired for the active
  Linux diagnostics; until then those follow-ups present as prompts everywhere (still useful).
