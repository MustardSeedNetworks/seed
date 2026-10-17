# Outbound alert SNMP traps

Seed can send every alert to an SNMP trap receiver you run, which is how most
network management systems expect to be told about a problem (#3037). It is
the fourth delivery channel beside the [signed webhook](ALERT_WEBHOOK.md),
[email](ALERT_EMAIL.md) and [syslog](ALERT_SYSLOG.md), and follows the same
rules: off until configured, never blocks detection, bounded retry, and the
outcome recorded on the alert.

Seed also _receives_ traps from devices (`SEED_SNMP_TRAP_BIND`). The two are
independent: this channel sends Seed's own alerts, not the device traps it
collected.

## Enabling it

Traps are off until an operator names a receiver, and nothing is sent, logged
or retried while they are off.

The settings live under `alerts.trap` in `PUT /api/v1/settings` (operator
role). There is no Settings screen for them yet.

| Field | Meaning |
| --- | --- |
| `host` | Trap receiver host name or address, without scheme or port. Clearing it turns traps off and deletes the stored community. |
| `port` | `0` for the snmptrap port, 162. |
| `community` | The SNMPv2c community the receiver admits. Required. Stored encrypted and never served back: `GET /api/v1/settings` reports only `communitySet`. Omit it to keep the stored one. |

```json
{
  "alerts": {
    "trap": {
      "host": "nms.example.com",
      "community": "seed-alerts"
    }
  }
}
```

The change takes effect immediately; no restart is needed. A receiver that
could never receive (a host with a port in it, a port over 65535, no
community) is refused at the API with the reason.

The trap is SNMPv2c. A v2c community crosses the network in clear text, so
use a community that grants nothing else, and keep the path between Seed and
the receiver on a management network.

## Test send

```text
POST /api/v1/settings/alerts/test   {"channel": "trap"}
```

Sends a test alert to the saved receiver now: `200 {"sent": true}`, or
`502 DELIVERY_FAILED` with the reason, for example a receiver name that does
not resolve. A trap is not acknowledged, so a `200` only means the datagram
left the host.

## What arrives

One SNMPv2-Trap-PDU per alert, over UDP. The notification is
`seedAlertRaised` from `SEED-ALERT-MIB`, under the Mustard Seed Networks
enterprise arc, IANA Private Enterprise Number 67249. Load both modules into
the receiver so it shows names rather than numbers:

- [`docs/mibs/MUSTARD-SEED-SMI.mib`](mibs/MUSTARD-SEED-SMI.mib)
- [`docs/mibs/SEED-ALERT-MIB.mib`](mibs/SEED-ALERT-MIB.mib)

`SEED-ALERT-MIB` imports `SNMPv2-SMI`, `SNMPv2-TC`, `SNMPv2-CONF`,
`SNMP-FRAMEWORK-MIB` and `HCNUM-TC`, which every SNMP toolkit ships.

The varbinds, in this order:

| # | Object | OID | Syntax | Value |
| --- | --- | --- | --- | --- |
| 1 | `sysUpTime.0` | `1.3.6.1.2.1.1.3.0` | TimeTicks | Hundredths of a second since Seed started. |
| 2 | `snmpTrapOID.0` | `1.3.6.1.6.3.1.1.4.1.0` | OID | `seedAlertRaised`, `1.3.6.1.4.1.67249.1.1.1.0.1`. |
| 3 | `seedAlertId.0` | `1.3.6.1.4.1.67249.1.1.1.1.1.0` | Counter64 | Alert ID, as served by `GET /api/v1/alerts`. `0` on a test send. |
| 4 | `seedAlertSeverity.0` | `1.3.6.1.4.1.67249.1.1.1.1.2.0` | INTEGER | `critical(1)`, `error(2)`, `warning(3)` or `info(4)`. An unknown severity is sent as `warning`. |
| 5 | `seedAlertType.0` | `1.3.6.1.4.1.67249.1.1.1.1.3.0` | SnmpAdminString | Alert type, e.g. `performance`, `connectivity`, `security`. |
| 6 | `seedAlertRule.0` | `1.3.6.1.4.1.67249.1.1.1.1.4.0` | SnmpAdminString | The rule that raised it, e.g. `iface.down`. |
| 7 | `seedAlertSource.0` | `1.3.6.1.4.1.67249.1.1.1.1.5.0` | SnmpAdminString | What generated it. |
| 8 | `seedAlertDevice.0` | `1.3.6.1.4.1.67249.1.1.1.1.6.0` | SnmpAdminString | The device it is about. |
| 9 | `seedAlertCause.0` | `1.3.6.1.4.1.67249.1.1.1.1.7.0` | Counter64 | ID of an earlier alert that probably caused this one, `0` when none. |
| 10 | `seedAlertTitle.0` | `1.3.6.1.4.1.67249.1.1.1.1.8.0` | SnmpAdminString | One-line summary. |
| 11 | `seedAlertMessage.0` | `1.3.6.1.4.1.67249.1.1.1.1.9.0` | SnmpAdminString | The alert's text. |
| 12 | `seedAlertSummary.0` | `1.3.6.1.4.1.67249.1.1.1.1.10.0` | SnmpAdminString | The narrative's summary: what happened and where. |
| 13 | `seedAlertNextCheck.0` | `1.3.6.1.4.1.67249.1.1.1.1.11.0` | SnmpAdminString | The narrative's next check. |
| 14 | `seedAlertEvidence.0` | `1.3.6.1.4.1.67249.1.1.1.1.12.0` | SnmpAdminString | The narrative's evidence, sentences separated by a semicolon and a space. |

Every object is always sent. A value the alert does not have is an empty
string or `0`, so a receiver can read the objects by position as well as by
name. The three narrative objects are empty unless the alert heads a cluster
of correlated alerts and has a rule Seed can explain; they then carry the same
summary, evidence and next check the Alerts page shows, in English.

Text is UTF-8 and cut to 255 octets, the `SnmpAdminString` limit, on a
character boundary.

## Delivery behaviour

The same as the other channels: a 256-alert queue that drops (and records the
drop) rather than blocking detection, three attempts with linear backoff, and
a 10 second bound per attempt. Nothing acknowledges a trap, so only a receiver
name that does not resolve, or a datagram the host cannot send, is recorded as
a failure; a receiver that is down loses the alert and Seed records it as
delivered. Use the [webhook](ALERT_WEBHOOK.md) or syslog over `tcp` or `tls`
when an alert must not be lost silently.

Each alert's trap outcome is recorded on the alert and served by
`GET /api/v1/alerts` as one entry of `deliveries`, with `"channel": "trap"`.
An escalation stage may name `trap` ([ALERT_ESCALATION.md](ALERT_ESCALATION.md)).

Alert delivery rides on the alert pipelines, which are Pro-tier engines. On Free
and Starter the pipelines do not run, so there are no alerts to deliver.
