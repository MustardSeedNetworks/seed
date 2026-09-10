# 0032 — Forwarding-database edges: one MAC on an unclaimed port

Status: Accepted
Date: 2026-09-09

## Context

Endpoints — infusion pumps, imaging systems, nurse stations, printers,
cameras — run no discovery protocol. LLDP, CDP and FDP describe switch-to-switch
cables and nothing else, so a topology built from those three protocols is
switches only. On the niac hospital pack (S4-4, #2454) seed drew 62 of the 88
authored links: the 26 it missed were every switch-to-endpoint edge, and the
EtherScope nXG drew all 26 from the same devices.

The evidence seed was already collecting and never reading is the switch's
forwarding database. The `fdb` collector is in every target's chain and its rows
land in `snmp_observations`; the edge reconciler consumed `lldp`, `cdp` and
`fdp` only.

A forwarding database is not a topology. Every MAC in a site appears on the
uplink of every switch between it and the observer, so consuming the table
naively draws an edge from each endpoint to each switch on its path. The rule
below is what separates "attached to this port" from "reachable through this
port".

## Decision

The edge reconciler gains an `fdb` pass writing links of type `fdb`. A learned
MAC becomes an edge only when all of these hold:

- the row is `dot1qTpFdbStatus = learned(3)` — `self` and `mgmt` rows are the
  switch's own addresses;
- the bridge port translates to a non-zero `ifIndex`;
- exactly one **distinct** MAC is learned on that `ifIndex`, deduplicated across
  VLANs — `dot1qTpFdbTable` is per-VLAN, so a trunk carrying one MAC in ten
  VLANs otherwise reads as ten access ports;
- no neighbour protocol has already claimed that local port, and the resolved
  node is not already a neighbour of this switch — an uplink quiet enough to
  hold a single MAC is still an uplink;
- the MAC resolves to a node seed already knows, and that node is not the
  switch itself.

The port-cardinality test runs before any database lookup: a core switch holds
thousands of MACs on a handful of trunk ports, and none of them should become a
query.

MAC-to-node resolution goes through `TopologyRepository.NodeForMAC`, which
matches `topology_nodes.primary_mac` **and** `topology_interfaces.if_phys_addr`.
No producer writes `primary_mac`, so the interface's `ifPhysAddress` is in
practice the only MAC seed knows for a device; a second MAC-keyed lookup over a
different table is how the two would drift.

The pass keeps its own high-water mark, `topology.edge.fdb.high_water`. Sharing
the neighbour mark would skip every `fdb` observation already on disk at upgrade
and leave the access layer blank until every switch is polled again.

## Consequences

- A seed map of an access layer shows what is plugged into it.
- The ARP reconciler's `primary_ip` backfill starts working: it resolves through
  the same `NodeForMAC`, and had matched only the never-written `primary_mac`.
- An endpoint that moves keeps its old edge until the switch ages the entry out
  and the reconciler re-reads the port. Accepted: a forwarding database is a
  cache, and the alternative is an aging model seed cannot observe.
- An endpoint seed has never polled has no node and therefore no edge, the same
  rule the neighbour pass applies to an unpolled neighbour.
- An IP phone with a PC daisy-chained behind it is two MACs on one access port
  and draws no edge. That is the rule working as written, not a defect: a port
  with more than one MAC cannot say which device is on the cable.
- Local ports on `fdb` edges are labelled `ifIndex-N`, identical to the
  neighbour pass. #2455 replaces that label with the interface's real name and
  must change both passes together: the uplink test compares its own label
  against the labels the neighbour pass wrote.
