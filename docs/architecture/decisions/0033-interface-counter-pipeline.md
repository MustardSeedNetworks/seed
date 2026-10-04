# 0033 — Interface counter pipeline: rates in `metrics`, not counters in observations

Status: Accepted
Date: 2026-10-04

## Context

SNMP interface counters are cumulative. One reading of `ifHCInOctets` says
nothing about current traffic; the difference between two readings is a rate
only when nothing reset the counter in between. Until P-A2 (#3001, #3006) seed
walked the six counters (in/out octets, errors, discards) on the discovery path
alone, so it never had two readings of the same interface a known interval
apart, and every trending, utilization and baseline feature had nothing to
stand on.

P-A2 put the counters on the `if_table` collector, which already runs on every
poll of every target ([ADR-0023](0023-snmp-polling-orchestrator.md)). This ADR
records the four choices that pipeline makes, because each one bounds what the
rows built on it (P-A3, P-A4, P-A6, P-B6 to P-B8) can claim:

1. how often a counter is sampled;
2. how a delta is taken across a 32-bit wrap, an agent restart and a cleared
   counter;
3. where the result is stored and how the retention tiers treat it;
4. why the stored thing is a rate in the timeseries store rather than the raw
   counter in `snmp_observations`;
5. how utilization is derived from the octet rates, and how the line rate
   bounds them (P-A4).

## Decision

### 1. Cadence is the target's poll interval

Counters are read from the `if_table` walks the collector already does, so
they are sampled at the polling target's `poll_interval_seconds` (default 300).
There is no separate counter schedule. A second scheduler would double the walk
load on every agent to buy a resolution that P-A4's utilization and the P-B
trend reports do not need.

A rate is therefore the **mean over one poll interval**. A burst shorter than
the interval is averaged away. Sub-second burst detection is P-A6's microburst
sampling (100 ms, 10 ms in burst mode), not a shorter SNMP poll.

### 2. Deltas: wrap, restart and clear

`internal/timeseries/ifrate` keeps the previous reading per target and
interface in memory and rates each new reading against it. A pair of readings
yields **no rate**, and the new reading becomes the baseline, when any of these
holds:

- `sysUpTime` went backwards: the agent restarted, and every counter it serves
  may have restarted with it. A missing `sysUpTime` is treated the same way,
  since it cannot be told apart from a restart.
- `ifCounterDiscontinuityTime` changed: the interface's counters were cleared
  without an agent restart.
- the octet source switched between the 64-bit `ifHC*` columns and the 32-bit
  `ifTable` columns. The collector uses the 64-bit pair whenever the agent
  serves both and records which width it used.
- a 64-bit counter went down. A Counter64 at 100 Gb/s takes decades to wrap, so
  a smaller value is a reset.

A smaller 32-bit value with none of the above is **one** wrap through 2^32
(`snmp.Counter32Delta`). If any of an interface's six counters is
discontinuous, all six are dropped for that interval: a reset of one is a reset
of the interface.

Every rule errs towards dropping a sample. A false gap of one interval is
visible and harmless; a false spike of several gigabytes corrupts every hourly
and daily rollup it lands in, and P-B alert thresholds would fire on it.

The known costs of that choice:

- The first poll after a daemon restart only sets a baseline, because the
  Rater's state is in memory.
- An `snmpd` restart that keeps the kernel counters still drops one interval.
- `sysUpTime` is TimeTicks and wraps after about 497 days, which reads as a
  restart and drops one interval.

### 3. Storage and retention

Each interval's six rates are written to `metrics` as six points with
`target_kind = snmp_interface` and `target_id = <polling target ID>/<ifIndex>`.
Octets are in `octets/s`, errors and discards in `packets/s`. An Ethernet
interface adds one point per EtherLike-MIB `dot3StatsTable` counter its agent
serves (P-A3, up to 13, metric types `dot3_*`), in `frames/s`, or `events/s`
for the four that count line events rather than frames. `interface_name`
repeats `target_id`, because the hourly and daily rollups key on it and an
`ifName` is unique only within one device.

`metrics` is already a `RollupSource` of the unified retention engine, so the
rates inherit the locked tier horizons with no new code: raw for 7 days on
every tier, hourly to 30 days on Starter and 90 on Pro, daily to 2 years on Pro.
The hourly rollup keeps average, minimum and maximum. Over rates that means the
hourly maximum is the busiest single poll interval in the hour, which is the
peak a capacity report wants. The average is an unweighted mean of interval
rates, which is the true mean only while a target's intervals are equal; poll
jitter keeps it close, not exact.

Volume is the cost to watch. Each interface adds 6 rows per poll: 1,728 a day
at the default interval. A 48-port switch adds about 83,000 rows a day, and
7 days of raw for 100 such switches is about 58 million rows. That is inside
what the single-writer SQLite design ([ADR-0031](0031-single-writer-sqlite-connection.md))
handles, but it is the reason the default interval is not shortened, and the
reason raw retention stays at 7 days on every tier. The EtherLike counters
raise it to as many as 19 rows per Ethernet port per poll, about three times
those figures for a switch whose agent serves the whole `dot3StatsTable`. That
is the price of a zero rate being a stored fact rather than a missing row.

### 4. Why rates, and why `metrics`

The raw cumulative counters still land in `snmp_observations`, inside the
`if_table` JSON payload, like every other collector's output. They are not what
the rest of seed reads:

- **A counter is meaningless alone.** Every consumer wants a rate, and a rate
  needs the previous reading plus the discontinuity rules above. Taking the
  delta once, at ingest, puts those rules in one place. Leaving it to readers
  would make each report, alert and overlay re-implement them and disagree at
  the edges.
- **Observations are snapshots, not a series.** `snmp_observations` holds one
  JSON document per collector run, keyed for "latest state of this target" (the
  edge reconciler, the admin UI). Charting one interface means decoding every
  document in the window and picking one row out of each. `metrics` is indexed
  by target and metric type and holds one number per row.
- **Only the timeseries store has tiers.** Hourly and daily rollups, and the
  licence-tier horizons that decide how far back a Starter or Pro install can
  look, exist for `metrics` and `probe_results`. A rate kept in the observation
  table would need a second retention scheme. A rolled-up counter would mean
  nothing: the average of a cumulative counter is not a rate.

### 5. Utilization and the line-rate bound

Utilization is each direction's octet rate as a percentage of the interface's
line rate: `100 × 8 × octets/s ÷ line rate`, stored as `if_in_utilization` and
`if_out_utilization` in `percent`, beside the octet rates and under the same
`target_id`, which adds two rows per interface per poll. The line rate is
`ifHighSpeed` (in Mb/s) converted to bits per second, or `ifSpeed` when the
agent serves no `ifXTable`. An `ifSpeed` at its 32-bit maximum is treated as
unknown, because RFC 2863 has an agent report any faster link that way. With
the line rate unknown, the octets are still rated and no utilization is
stored.

**Full-duplex convention.** Every link is taken to be full duplex: inbound
and outbound are each measured against the whole line rate and never summed.
A saturated full-duplex gigabit port reads 100 percent in both directions, not
200 percent and not 50. IF-MIB does not say whether a link is half duplex, and
on the rare half-duplex link that is still in service the two directions share
the medium, so the real load is the sum of the two figures. That case reads
low, and the EtherLike-MIB collision counters (P-A3) are where it shows.

The line rate also bounds the octet rates, which settles the two consequences
this ADR left to P-A4:

- **Above line rate.** A direction whose rate exceeds the line rate by more
  than 10 percent is a counter reset, not traffic, and the interface's whole
  interval is dropped like any other discontinuity. This catches net-snmp's
  silent 2^32 jump (#3004) wherever the jump is larger than the line could
  carry in the interval, which at the default interval is any port up to
  100 Mb/s. On a faster port it is within the line rate, and this check cannot
  see it. The 10 percent slack exists because a poll's
  interval is timed from when the collector started it, not from when the
  agent read the counters. A poll whose walks ran slower than the last one's
  counts its bytes over a shorter measured interval. Within the slack, the
  octet rate is stored as measured and utilization is capped at 100.
- **32-bit wrap.** A 32-bit octet counter on a link that could carry 2^32
  octets in one interval (line rate × interval ≥ 2^32 × 8 bits) is not rated.
  Its octet and utilization points are omitted, and its error and discard
  rates are still stored. At 300 s that applies to any 32-bit counter on a link
  faster than about 114.5 Mb/s. Such agents are rare, because RFC 2863 has an
  agent serve the 64-bit columns on any interface faster than 20 Mb/s.

## Consequences

- **Trustworthy range of 32-bit octet rates.** An agent that serves only 32-bit
  octets wraps every 2^32 bytes. At 300 s, more than one wrap per interval
  happens above an average of about 114.5 Mb/s, and the Rater cannot tell two
  wraps from one. Where the line rate is known, §5 leaves such an interface
  unrated rather than showing it low. Where the line rate is unknown, the rate
  can still be silently low.
- **Rates above line rate.** net-snmp turns a silent counter reset into a jump
  of 2^32 in `ifHCInOctets` and does not move `ifCounterDiscontinuityTime`
  (#3004). No in-band signal catches it. §5's line-rate bound drops it on ports
  slow enough for the jump to exceed the line rate. On faster ports a spike of
  one interval remains possible, and #3004 stays open for it.
- **Raw observations need their own retention.** `snmp_observations` has a
  purge method and no caller (#3009), so the raw payloads this pipeline enlarged
  grow without bound. That is independent of the rates, which age out through
  `metrics`, and is fixed on its own row.
- **New counters follow this path.** P-A3's EtherLike-MIB counters are
  cumulative Counter32 too. They are rated by the same rules and stored as
  `metrics` points under the same `target_kind`, not given their own table or
  delta logic.
