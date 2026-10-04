# Outbound alert syslog

Seed can forward every alert to a syslog collector you run, which is how most
SIEMs expect to be fed (#3037). It is the third delivery channel beside the
[signed webhook](ALERT_WEBHOOK.md) and [email](ALERT_EMAIL.md), and follows
the same rules: off until configured, never blocks detection, bounded retry,
and the outcome recorded on the alert.

Seed also _receives_ syslog from devices (`listeners.syslog`). The two are
independent: forwarding sends Seed's own alerts, not the device messages it
collected.

## Enabling it

Forwarding is off until an operator names a collector, and nothing is sent,
logged or retried while it is off.

The settings live under `alerts.syslog` in `PUT /api/v1/settings` (operator
role). There is no Settings screen for them yet.

| Field | Meaning |
| --- | --- |
| `host` | Collector host name or address, without scheme or port. Clearing it turns forwarding off. |
| `port` | `0` for the transport's IANA port: 514 for `udp` and `tcp`, 6514 for `tls`. |
| `transport` | `udp` (default), `tcp` or `tls`. |

```json
{
  "alerts": {
    "syslog": {
      "host": "siem.example.com",
      "transport": "tls"
    }
  }
}
```

The change takes effect immediately; no restart is needed. A collector that
could never receive — a host with a port in it, an unknown transport — is
refused at the API with the reason.

### Transports

| Transport | Standard | Notes |
| --- | --- | --- |
| `udp` | RFC 5426 | One datagram per alert, at most 2,048 octets. Nothing acknowledges a datagram, so a collector that is down loses the alert and Seed records it as delivered. |
| `tcp` | RFC 6587 §3.4.1 | Octet-counted framing (`LEN SP MSG`), one connection per alert. A refused connection is recorded as a failure. |
| `tls` | RFC 5425 | `tcp` framing inside TLS 1.2+. The certificate is verified against the system trust store, using `host` as the name. |

Use `tcp` or `tls` when an alert must not be lost silently. With `tls`, a
collector certificate from a private CA needs that CA in the host's trust
store.

## Test send

```text
POST /api/v1/settings/alerts/test   {"channel": "syslog"}
```

Sends a test alert to the saved collector now: `200 {"sent": true}`, or
`502 DELIVERY_FAILED` with the reason, for example
`connect to siem.example.com:6514: tls: failed to verify certificate …`. Over
`udp` a `200` only means the datagram left the host.

## What arrives

One RFC 5424 message per alert:

```text
<130>1 2026-09-14T12:00:00.000000Z probe-01 seed - alert - id=42 severity=critical type=performance rule=gateway.latency source=alert-observation-pipeline device=core-sw-01 title="Gateway latency threshold breached" message="gateway latency 812ms over the 200ms threshold"
```

| Header field | Value |
| --- | --- |
| PRI | Facility `local0` (16) and the alert's severity: `critical` → 2, `error` → 3, `warning` → 4, `info` → 6. |
| TIMESTAMP | When the alert was raised, UTC, microseconds. |
| HOSTNAME | The Seed host's name. |
| APP-NAME | `seed` |
| PROCID | `-` |
| MSGID | `alert` |
| STRUCTURED-DATA | `-` (see below) |

MSG is `key=value` pairs (logfmt), in this order, each present only when the
alert has it:

| Key | Meaning |
| --- | --- |
| `id` | Alert ID, as served by `GET /api/v1/alerts`. Absent on a test send. |
| `severity` | `critical`, `error`, `warning` or `info`. |
| `type` | Alert type, e.g. `performance`, `connectivity`, `security`. |
| `rule` | The rule that raised it, e.g. `iface.down`. |
| `source` | What generated it. |
| `device` | The device it is about. |
| `cause` | ID of an earlier alert that probably caused this one. |
| `summary` | The narrative's summary: what happened and where. |
| `next_check` | The narrative's next check. |
| `title` | One-line summary. |
| `message` | The alert's text. |
| `evidence` | The narrative's evidence, sentences separated by a semicolon and a space. |

An alert that heads a cluster of correlated alerts, and has a rule Seed can
explain, also carries its narrative: the same summary, evidence and next check
the Alerts page shows, in English. The three narrative keys are absent when it
has none. The evidence comes last because a UDP datagram is cut at 2048 bytes
from the end, and it is what a reader can most afford to lose.

A value containing a space, `"`, `=`, `\` or a control character is a
double-quoted string with Go escaping (`\"`, `\\`, `\n`). Text a device
controls therefore cannot break the line or add a field.

The facts are in MSG rather than RFC 5424 structured data because a custom
SD-ID must carry the sender's IANA Private Enterprise Number, and Mustard Seed
Networks does not have one yet.

## Delivery behaviour

The same as the other channels: a 256-alert queue that drops (and records the
drop) rather than blocking detection, three attempts with linear backoff, and
a 10 second bound per attempt. An untrusted certificate is not retried; a
refused connection is.

Each alert's syslog outcome is recorded on the alert and served by
`GET /api/v1/alerts` as one entry of `deliveries`, with `"channel": "syslog"`.

Alert delivery rides on the alert pipelines, which are Pro-tier engines. On Free
and Starter the pipelines do not run, so there are no alerts to deliver.
