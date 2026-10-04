# ADR-0022: Passive-ingress listeners share the engine lifecycle and a sink seam

**Status:** Accepted — 2026-06-10 · as-built (documents `internal/listener`, shipped during the V1.0 NMS expansion)
· amended 2026-10-04 (SNMPv3 traps, P-B4; flow collector, P-C1)

## Context

The active diagnostics path _initiates_ network conversations: the SNMP poller
(ADR-0023) reaches out to targets, discovery probes hosts. But an NMS also has to
receive **unsolicited** signals — syslog messages, SNMP traps, later NetFlow/IPFIX
— that devices push at us over UDP with no request on our side. These arrive
continuously, from many sources, on privileged low ports, and must be persisted
without back-pressuring the receive loop.

These passive endpoints share a shape: bind a socket, parse a wire format, emit a
normalized record, and run for the lifetime of the daemon. The risk was that each
one grows its own ad-hoc goroutine, its own start/stop handling, and its own
database writes — the same "per-subsystem bucket" sprawl the re-architecture is
removing elsewhere. We already have a uniform lifecycle owner: the `engine.Engine`
registry that supervises the probe, retention, and snmp-poller engines (Name +
Start + Stop, with `Status` reporting). Passive listeners fit that shape exactly.

## Decision

A single small contract package, **`internal/listener`**, defines the passive
ingress seam; concrete listeners live in subpackages and persistence lives behind
a port.

- **`Listener` is `engine.Engine`-shaped** — `Name() string`, `Start(ctx) error`,
  `Stop(ctx) error`, both idempotent. Listeners register directly with the engine
  registry; there is **no listener-specific supervisor**. Start binds the socket
  and streams; Stop closes it and drains in-flight handlers within the context
  deadline.

- **Listeners own no persistence — they publish to a `Sink` port.** `Sink.Publish(
  ctx, Event) error` is the only outbound seam. The default implementation
  (`internal/listener/sink`) inserts into the `listener_events` table; tests inject
  a recording fake. A listener never imports the database.

- **`Event` is the normalized record** — `Kind` (which listener), `SourceAddr`,
  `Severity` (listener-native string), `Timestamp`, and an opaque listener-specific
  `Payload json.RawMessage` whose schema each subpackage documents. `ClientID`,
  `TargetKind`, and `TargetID` are **enrichment outputs**, not listener concerns:
  the listener emits the raw `SourceAddr` and a later step resolves it.

- **Concrete listeners (V1.0):** `internal/listener/syslog` (RFC 3164 / RFC 5424
  over UDP) and `internal/listener/snmptrap` (SNMPv2c traps over UDP/162).

- **Binding is opt-in and fail-soft.** The composition root wires listeners in
  `initListeners` (`internal/api/server.go`); a bind failure logs a warning and the
  listener is simply not registered, rather than aborting daemon startup — a busy
  `:514`/`:162` or an unprivileged deployment must not take the server down.

## Consequences

- A new passive source (NetFlow, IPFIX, a TLS syslog variant) is a new subpackage
  implementing `Listener` plus a registry registration — no new supervisor, no new
  persistence path, no change to `internal/listener` itself.
- Because `Sink` is the only write seam, the storage decision (currently one row
  per event in `listener_events`) can change in one place without touching any
  listener, and the alerts pipeline (`internal/alerts/pipeline/listener_pipeline.go`)
  consumes the same persisted stream.
- Publishing is decoupled from receiving: a slow sink must not block the UDP read
  loop, so sink errors are logged rather than propagated back into the hot path.
- Listeners inherit the engine registry's uniform `Status`/lifecycle, so the same
  health surface that covers the poller covers ingress — no bespoke monitoring.

## Implementation status (2026-06-10)

- **`internal/listener`** — `Listener`, `Event`, `Sink` contracts (no I/O).
- **`internal/listener/syslog`** — UDP listener, RFC 3164 + RFC 5424 parse,
  default bind `:514`, remappable for unprivileged hosts. **TLS/TCP syslog
  (RFC 5425) is deferred** — it is a separate `Listener` type, not a flag on this
  one.
- **`internal/listener/snmptrap`** — SNMP traps over UDP/162: v1 and v2c traps
  and informs, and SNMPv3 traps (see the 2026-10-04 amendment below). Event kind
  `snmp-trap` for every version.
- **`internal/listener/sink`** — default `Sink` over `database.ListenerEvents()`
  (`listener_events`, migration `00001_init.sql`).
- **Wiring** — `initListeners` in `internal/api/server.go` constructs the persist
  sink and both listeners and registers them with the engine registry.
- **Enrichment is NOT wired yet (Stage A4).** `ClientID` is `"default"` and
  `TargetKind`/`TargetID` are empty/`"unknown_ip"` on every event until the step
  that resolves `SourceAddr` against `polling_targets` / `discovered_devices`
  lands. Events persist and flow to the alerts pipeline; they are simply not yet
  attributed to a known device.

## Amendment 2026-10-04: SNMPv3 traps (P-B4, seed#1376)

The original cut left SNMPv3 out because gosnmp's trap code was flagged as
unreliable for v3. The parity program puts v3 trap receipt in v1, so we
re-tested gosnmp v1.44.0 against net-snmp 5.9.4's `snmptrap` and `snmpinform`.

**USM decoding is sound and is kept.** Traps at authPriv with SHA/AES-128,
SHA-256/AES-256-C and SHA/DES, and at authNoPriv with SHA-512, all
authenticate and decrypt. Keys are localized against each sender's engine ID,
and a wrong passphrase is rejected.

**`gosnmp.TrapListener` is not a safe v3 receiver, so it is no longer used.**
The re-test found three gaps:

1. It authenticates at whatever level the message claims. A noAuthNoPriv trap
   that names an authPriv user is accepted without authentication. Anyone who
   knows a user name can inject traps under it.
2. It does not check the time window (RFC 3414 3.2.7). An authenticated trap
   captured once can be replayed indefinitely.
3. With more than one user configured, it drops the RFC 3414 §4 discovery
   probe, so a v3 inform sender never learns an engine ID and times out.

`internal/listener/snmptrap` now owns the UDP socket and calls gosnmp only to
decode. It also:

- **Takes users from the credential vault.** The v3 rows of
  `device_credentials` are resolved through the provider discovery uses, with
  the same single-client rule. Seed has no second place to configure SNMPv3
  users. The table is re-read at most every 30 seconds, so a credential added
  after start is honoured without a restart.
- **Enforces the stored security level.** A message is accepted only at a
  level one of its user's stored credentials has. A noAuthNoPriv trap for an
  authPriv user is dropped.
- **Checks the time window.** It keeps the non-authoritative clock cache of
  RFC 3414 3.2.7 b for each sending engine ID. A trap more than 150 seconds
  behind the receiver's running estimate of that engine's clock, or from an
  earlier boot, is dropped. Replays inside the window are still accepted, as
  the RFC allows.
- **Drops v3 informs for now.** For an inform, Seed is the authoritative
  engine. Supporting informs needs a persisted snmpEngineID and
  snmpEngineBoots, and the discovery and notInTimeWindow reports. That is the
  second slice of seed#1376. Informs at v1 and v2c are still acknowledged.

A dropped datagram is logged at debug level with its reason and is never
persisted. The trap listener's event kind is now `snmp-trap` for every
version. Migration `00016` renames stored events and alert rules that used the
old `snmp-trap-v2c`.

## Amendment 2026-10-04: the flow collector (P-C1, seed#3082)

`internal/listener/flow` receives NetFlow v5, NetFlow v9 and IPFIX on one UDP
socket (`SEED_FLOW_BIND`, conventionally `:2055`) and picks the decoder from
each datagram's version field. It is a `Listener` and registers like the
others, at the Pro tier.

**Flows do not go through `Sink`.** `Sink.Publish` takes one `Event` with a
JSON payload and writes one `listener_events` row per call. A datagram holds up
to 30 flows, and an exporter sends thousands of datagrams a minute. A row per
flow with a JSON body would make every later reader parse JSON to sum bytes,
and a write per flow would cap ingest at the database's single-row insert rate.
The flow package therefore defines its own port, `flow.Store`, which takes a
batch of typed `flow.Record`s. `database.FlowRecordsRepository` writes each
batch in one transaction into `flow_records` (migration `00019`), with one
typed column per field. The rest of the decision holds: the listener owns no
SQL, and the composition root binds the port.

**Memory is bounded on both paths an exporter controls.**

- _Records._ The read loop decodes each datagram and offers its records to a
  queue of 8,192. A record that does not fit is dropped and counted. The read
  loop never waits on the database. One writer goroutine drains the queue in
  batches of up to 512, or each second. Stop closes the socket, and the writer
  flushes what is queued before it returns.
- _Templates._ v9 and IPFIX templates are keyed by exporter address, version,
  observation domain and template ID (RFC 7011 §8). The cache holds at most
  4,096 templates of at most 256 fields. When it is full, the least recently
  refreshed template is evicted. A template that has not been refreshed for an
  hour expires.

**Template loss is counted, not buffered.** A data set whose template is not
known is dropped and counted. This happens when the collector starts after
the exporter sent its templates, or after the template expires. Exporters
resend templates on a timer, so flows decode again from the next refresh.
Holding data sets until their template arrives would put an exporter-controlled
buffer back into memory. A redefinition of the same template ID, as after an
exporter restart, replaces the old template. IPFIX withdrawals, of one
template or of all of them, are honoured.

The listener logs one summary a minute while it is losing records: queue-full
drops, data sets without a template, malformed datagrams and failed batches.
The retention engine purges `flow_records` at the tier's raw horizon. Flows
have no hourly or daily tier yet. The conversation aggregates of P-C3 will be
that tier.

Options records carry exporter metadata, not flows. The collector reads one
value from them: `systemInitTimeMilliseconds`. softflowd, nProbe and several
Cisco images send IPFIX flow times relative to exporter uptime and send the
boot time this way. The boot time is kept per exporter and observation domain,
and expires with the templates. A 32-bit millisecond uptime wraps every 49.7
days, so an uptime resolves to the instant nearest the export time that it can
denote. Without it, an uptime-relative IPFIX record is
stamped with its export time.

Stored byte and packet counts are estimates of the traffic observed, not the
sampled counts. v5 counters are scaled by the header's sampling interval. v9
and IPFIX exporters announce their rate in options records, as a 1-in-N
interval (elements 34 and 50) or as RFC 5476's selected interval and skipped
space (305 and 306). The collector keeps each rate by what it covers, most
specific first: a sampler (an options record carrying sampler ID 48 or
selector ID 302, which data records then name), an ingress interface (a v9
Interface scope or an IPFIX `ingressInterface` scope), or the whole domain.
softflowd scopes its v9 rate to an interface and its IPFIX rate to the
metering process; Cisco names a sampler. A rate carried in the data record
itself wins over all of them. Rates share the template cache's bound and
lifetime. A flow decoded before its exporter's options record arrives keeps
its sampled counts, the same trade the template cache makes for data ahead of
its template.

## Amendment 2026-10-04: sFlow on the flow collector (P-C2, seed#3090)

The flow collector also takes sFlow v5. It shares the NetFlow socket rather
than adding a second listener: a NetFlow or IPFIX datagram opens with a
two-byte version of 5, 9 or 10, and an sFlow datagram with a four-byte version,
so its first two bytes are zero. The decoder dispatches on them. An operator
points sFlow agents at the `SEED_FLOW_BIND` port, not at 6343.

An sFlow flow sample is one packet the agent sampled 1-in-N. Each sample of an
IP packet becomes one `flow_records` row standing for N packets: `packets` is
the sampling rate and `bytes` the packet's IP length times the rate. That is
the layer-3 length NetFlow counts, so rows from both protocols sum together.
The IP length comes from the header itself (the IPv4 total length, or the IPv6
payload length plus 40), not from the frame length, so it is right for a
header the agent truncated. The decoder reads the flow key from a
`sampled_header` record holding an Ethernet (with up to four VLAN tags), IPv4
or IPv6 header, or from a `sampled_ipv4` or `sampled_ipv6` record, which wins
when an agent sends both. It walks IPv6 extension headers to the transport
header and reads no ports from a non-first fragment. A sample of a packet that
is not IP is skipped. Counter samples and other enterprises' structures are
skipped too.

sFlow carries no wall clock, only agent uptime, so a row's start and end are
the time the datagram arrived. The exporter is the agent address the datagram
names, falling back to the UDP source when it names none. The observation
domain is the sub-agent ID. An interface that is not a single ifIndex
(discarded, several outputs, or unknown) is stored as 0, the NetFlow
convention.

`flow_records.version` became `format` (migration `00021`): `netflow5`,
`netflow9`, `ipfix` or `sflow5`. sFlow v5 and NetFlow v5 share version 5, so
the number alone could not say which protocol a row came from.
