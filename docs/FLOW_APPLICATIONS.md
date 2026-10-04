# Application identification from flows

The flow collector names the application each flow carries, using its IP
protocol and ports (#3101). A flow on TCP 443 is reported as `https` because
that port conventionally carries HTTPS. Seed reads no payload to decide this,
and the names are heuristic. A flow that matches no signature is reported as
`unknown`. Seed never guesses a name from a nearby port or from how the
traffic behaves, so the size of `unknown` in a breakdown shows how much
traffic the table does not cover.

## How a flow is matched

- **Port-carrying protocols.** For TCP (6), UDP (17) and SCTP (132), a
  signature lists ports. A flow matches when either of its ports is listed.
  When both ports match different signatures, the lower port wins. Servers
  listen on the lower, registered ports, and clients send from the higher
  ephemeral range.
- **Other protocols.** A signature for any other protocol, such as ICMP (1),
  GRE (47) or OSPF (89), lists no ports and matches every flow of that
  protocol.
- **One signature per port.** No port of a protocol may belong to two
  signatures, and a protocol cannot have both a whole-protocol signature and
  port signatures. Every flow therefore has exactly one answer.

The name is fixed when the flow is stored. Editing the table changes how flows
are named from then on, but flows already stored keep the names they were
given. The hourly and daily rollups carry those names past the raw horizon.

## The table

Seed ships with a table of about sixty common services, from DNS, HTTPS and
SSH to Modbus and BACnet. Its source is
[`internal/appid/signatures.json`](../internal/appid/signatures.json). Each
entry has a name, a description, an IP protocol number and, for TCP, UDP and
SCTP, a list of ports:

```json
{
  "signatures": [
    {
      "name": "historian",
      "description": "Plant historian",
      "protocol": 6,
      "ports": ["5450", "5460-5469"]
    }
  ]
}
```

| Field | Rule |
| --- | --- |
| `name` | 1 to 32 lowercase letters, digits or hyphens. `unknown` is reserved. Two entries may share a name, for example DNS over UDP and DNS over TCP. |
| `description` | Free text for the person who reads the table. |
| `protocol` | IP protocol number, from 1 to 255. |
| `ports` | Required for 6, 17 and 132, and forbidden for any other protocol. Each entry is one port (`"443"`) or an inclusive range (`"6000-6063"`), from 1 to 65535. A signature may have up to 64 entries. |

A table holds at most 256 signatures.

## Editing it

There is no Settings screen for the table yet. Edit it through the API:

| Request | Effect | Role |
| --- | --- | --- |
| `GET /api/v1/flows/application-signatures` | The table in effect. `source` is `builtin` or `custom`. | any |
| `PUT /api/v1/flows/application-signatures` | Replaces the whole table with the body. | operator |
| `DELETE /api/v1/flows/application-signatures` | Returns to the table Seed ships with. | operator |

A `PUT` replaces the table and does not merge into it. To add one service,
`GET` the table, append your entry and `PUT` the result. A table that breaks
any rule above is refused with `400`. The response's `details` names the
offending signature and port entry by position. One table serves the whole
installation.

## Reading the breakdown

`GET /api/v1/flows/top-applications` takes the same `range`, `by` and `limit`
parameters as top talkers. It returns the applications that carried the most
traffic, with `unknown` ranked alongside the rest:

```json
{
  "window": { "days": 1, "source": "raw", "clamped": false },
  "by": "bytes",
  "applications": [
    { "name": "https", "bytes": 912345678, "packets": 734120 },
    { "name": "unknown", "bytes": 81234567, "packets": 90211 },
    { "name": "dns", "bytes": 1203345, "packets": 14002 }
  ]
}
```

The response is abridged here. `window` carries every field described for
the history series.
