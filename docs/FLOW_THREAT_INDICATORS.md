# Threat indicators in flows

The flow collector checks every flow it receives against a list of hostile
addresses that you supply (#3106). When a host on your network exchanges
traffic with a listed address, Seed raises a critical security alert that
names the host, the listed address and the list entry it matched.

Seed ships no list and fetches none. The list exists only when you supply it,
so an air-gapped install never makes an outbound attempt because of this
feature. Updating the list from a published feed is a step you take, on your
schedule, through the API below.

## What is matched

- **Only non-internal addresses.** A private (RFC 1918), shared (CGNAT,
  100.64.0.0/10), loopback, link-local, unique-local, multicast or reserved
  address is a host on some local network, never a hostile service on the
  internet. The collector never checks those addresses. The list refuses
  entries that reach into those ranges, because such entries could never
  match.
- **Both directions.** A flow from a listed address to one of your hosts
  matches the same way as a flow from your host to it.
- **The most specific entry wins.** When a listed address falls inside two
  entries, the alert names the longer prefix.
- **From the next flow on.** A change to the list applies to flows collected
  after it. Flows already stored are not checked again.

Matching runs whether or not the flow reached the database, so a database
fault does not hide traffic to a listed address.

## The alert

The collector combines the flows of each one-second batch into one event per
host and listed address. The listener alert pipeline turns each event into an
alert:

| Field | Value |
| --- | --- |
| Rule | `flow.indicator` |
| Type | `security` |
| Severity | `critical` |
| Source | The host that exchanged traffic with the listed address |
| Metadata | The host, listed address, matched entry, exporter, flow, byte and packet counts, and first and last seen times |

Alerts are suppressed per host for the pipeline's window, five minutes by
default. A host that reaches several listed addresses within that window
raises one alert, and the events for the others remain in the listener event
log. Delivery, escalation and the alert inbox treat the alert like any other.

This rule always runs. Unlike the default syslog and trap rules, operator
alert rules do not replace it. The list is itself your instruction to alert,
and an unrelated rule edit must not silence it.

## The list

There is no Settings screen for the list yet. Edit it through the API:

| Request | Effect | Role |
| --- | --- | --- |
| `GET /api/v1/flows/threat-indicators` | The list in effect. | any |
| `PUT /api/v1/flows/threat-indicators` | Replaces the whole list with the body. An empty list turns matching off. | operator |

```json
{
  "indicators": ["198.51.100.7", "203.0.113.0/24", "2001:db8:bad::/48"]
}
```

Each entry is an IPv4 or IPv6 address or prefix, with no host bits set.
Write IPv4 addresses as IPv4, not in their IPv4-mapped IPv6 form. A list
holds at most 10,000 entries, which covers the common published blocklists.
The request body is limited to 256 KiB. Seed returns each entry in canonical
form: a single address is written without `/32` or `/128`, and IPv6 is
written in lowercase.

A `PUT` replaces the list and does not merge into it. A list that breaks any
rule above is refused with `400`, and the stored list stays as it was. The
response's `details` names the offending entry by position and never quotes
it. One list serves the whole installation.

Most feeds publish one address or prefix per line, with `#` comments. To
load such a file:

```sh
grep -Ev '^[[:space:]]*(#|$)' blocklist.txt \
  | jq -R . | jq -s '{indicators: .}' > indicators.json
curl -sk -X PUT https://seed.example:8443/api/v1/flows/threat-indicators \
  -H "Authorization: Bearer $SEED_TOKEN" -H 'Content-Type: application/json' \
  --data @indicators.json
```

## Licence

Flow collection is a Pro capability, so matching runs only on a Pro install
with the collector enabled (`SEED_FLOW_BIND`). Reading and editing the list
works on every tier.
