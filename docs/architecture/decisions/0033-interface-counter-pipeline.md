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
   counter in `snmp_observations`.

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
Octets are in `octets/s`, errors and discards in `packets/s`. `interface_name`
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
reason raw retention stays at 7 days on every tier.

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

## Consequences

- **Trustworthy range of 32-bit octet rates.** An agent that serves only 32-bit
  octets wraps every 2^32 bytes. At 300 s, more than one wrap per interval
  happens above an average of about 114.5 Mb/s, and the Rater cannot tell two
  wraps from one. Such a rate is silently low. P-A4 owns the bound: with
  `ifHighSpeed` known, a 32-bit interface whose line rate can exceed one wrap
  per interval is reported as unrated rather than shown low.
- **Rates above line rate.** net-snmp turns a silent counter reset into a jump
  of 2^32 in `ifHCInOctets` and does not move `ifCounterDiscontinuityTime`
  (#3004). No in-band signal catches it. The plausibility bound (a rate above
  the interface's line rate is discarded) belongs with P-A4's use of
  `ifHighSpeed`.
- **Raw observations need their own retention.** `snmp_observations` has a
  purge method and no caller (#3009), so the raw payloads this pipeline enlarged
  grow without bound. That is independent of the rates, which age out through
  `metrics`, and is fixed on its own row.
- **New counters follow this path.** P-A3's EtherLike-MIB counters are
  cumulative Counter32 too. They are rated by the same rules and stored as
  `metrics` points under the same `target_kind`, not given their own table or
  delta logic.
