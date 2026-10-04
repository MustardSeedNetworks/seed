# Alert escalation

Seed can send an alert again when nobody acknowledges it (P-B2, #3032). Each
alert rule can have a ladder. A stage fires a fixed time after the alert was
raised and sends the alert on the channels it names: the
[webhook](ALERT_WEBHOOK.md), [email](ALERT_EMAIL.md) or both. Acknowledging or
resolving the alert stops the ladder.

## Configuring a ladder

Ladders live under `alerts.escalations` in `PUT /api/v1/settings` (operator
role). The list replaces the stored one, and an empty list turns escalation
off. The Settings screen for ladders is UI work that has not been built yet.

| Field | Meaning |
| --- | --- |
| `rule` | The alert's rule, as `rule` on `GET /api/v1/alerts`: a built-in such as `iface.down` or `bgp.flap`, or `db.<id>` for an operator rule. One ladder per rule. |
| `stages[].afterSeconds` | When the stage fires, counted from when the alert was raised. At least 60, and later than the stage before it. |
| `stages[].channels` | `webhook`, `email` or both. |
| `repeatSeconds` | Re-send the last stage at this period until the alert is acknowledged or resolved. `0` (the default) sends it once; otherwise at least 60. |

At most five stages per ladder.

```json
{
  "alerts": {
    "escalations": [
      {
        "rule": "iface.down",
        "stages": [
          {"afterSeconds": 900, "channels": ["webhook"]},
          {"afterSeconds": 3600, "channels": ["webhook", "email"]}
        ],
        "repeatSeconds": 3600
      }
    ]
  }
}
```

The change takes effect immediately; no restart is needed. A ladder that could
not run is refused at the API with the reason. Seed does not run part of a
hand-edited config with a broken ladder: it logs the error, and no alert is
escalated until the ladders are fixed.

## What happens

Seed checks open alerts every 15 seconds. When a stage is due, Seed records
the stage on the alert, then sends the alert on that stage's channels. A
channel with no receiver configured is skipped.

- The alert carries `escalationStage` and `escalatedAt` in
  `GET /api/v1/alerts` and in the webhook payload.
- Email subjects read
  `[Seed] CRITICAL, unacknowledged, escalation 2: <title>`, and the body adds
  an `Escalation:` line.
- The channel's entry in `deliveries` shows how the latest send went.

The stage is stored on the alert, so a restart neither re-sends a stage nor
forgets one. After downtime, an alert that has passed several stages goes to
the latest one only; it is not paged once per missed stage. The stage is
recorded only while the alert is still open, so an alert acknowledged during
a pass is not sent.

Escalation runs on the Pro tier, like the alert pipelines that raise the
alerts.
