# Outbound alert email

Seed can email every alert through a mail relay you run (#2997). It is the
second delivery channel beside the [signed webhook](ALERT_WEBHOOK.md) and
follows the same rules: off until configured, never blocks detection, bounded
retry, and the outcome recorded on the alert.

## Enabling it

Email is off until an operator names a mail server, and nothing is sent, logged
or retried while it is off. An air-gapped install loses the feature, never
function.

The settings live under `alerts.email` in `PUT /api/v1/settings` (operator
role). The Settings screen for them is planned (UI-SEED-24).

| Field | Meaning |
| --- | --- |
| `host` | Mail server host name, without scheme or port. Clearing it turns email off and discards the stored password. |
| `port` | `0` for the mode's submission port: 587 for `starttls`, 465 for `tls`. |
| `tls` | `starttls` (default) or `tls` (implicit TLS from the first byte). |
| `username` | SMTP AUTH user. Empty for a relay that admits by address. |
| `password` | SMTP AUTH password. Write-only; see below. |
| `from` | Sender, e.g. `Seed <seed@example.com>`. |
| `to` | List of recipient addresses. At least one. |

```json
{
  "alerts": {
    "email": {
      "host": "smtp.example.com",
      "tls": "starttls",
      "username": "seed-alerts",
      "password": "…",
      "from": "Seed <seed@example.com>",
      "to": ["noc@example.com"]
    }
  }
}
```

The change takes effect immediately; no restart is needed. A configuration that
could never deliver — no recipient, an address that does not parse, a host with
a port in it — is refused at the API with the reason.

The password is stored as keyring ciphertext (the `enc:` prefix, ADR-0015) and
is never served back: `GET /api/v1/settings` reports `passwordSet` only.
Omitting the password, or sending an empty one, keeps the stored one, so
re-pointing the host does not require retyping it.

## TLS is required

There is no plaintext mode. An alert carries device names, addresses and the
alert text, and AUTH carries a password. With `starttls`, a server that does not
offer STARTTLS is refused with
`mail server does not offer STARTTLS; refusing to send unencrypted`.

The server's certificate is verified against the system trust store, using the
configured `host` as the name. A relay with a certificate from a private CA
needs that CA in the host's trust store.

## Test send

```text
POST /api/v1/settings/alerts/test   {"channel": "email"}
```

Sends a test alert through the saved relay now and answers with what the relay
said: `200 {"sent": true}`, or `502 DELIVERY_FAILED` with the step and the
server's reply, for example:

| Error | Meaning |
| --- | --- |
| `AUTH: 535 5.7.8 Authentication credentials invalid` | Wrong username or password. |
| `RCPT TO noc@example.com: 550 5.1.1 …` | The relay refuses a recipient. |
| `STARTTLS: tls: failed to verify certificate …` | The certificate is not trusted, or does not name `host`. |
| `connect to smtp.example.com:465: tls: first record does not look like a TLS handshake` | `tls` mode against a STARTTLS port; use `starttls`. |
| `connect to …: connection refused` | Nothing listening on that port. |

The route is operator-gated and rate-limited, and it only ever connects to the
saved relay.

## What arrives

One `text/plain` UTF-8 message per alert:

```text
From: "Seed" <seed@example.com>
To: <noc@example.com>
Subject: [Seed] CRITICAL: Gateway latency threshold breached
X-Seed-Alert-Id: 42
Auto-Submitted: auto-generated

Gateway latency threshold breached

Severity: critical
Type: performance
Source: alert-observation-pipeline
Device: core-sw-01
Rule: gateway.latency
Raised: 2026-09-14T12:00:00Z
Alert ID: 42

gateway latency 812ms over the 200ms threshold
```

An alert that heads a cluster of correlated alerts, and has a rule Seed can
explain, also carries its narrative: the same summary, evidence and next check
the Alerts page shows, in English. It comes first, between the title and the facts:

```text
Interface Gi0/3 down on core-sw1

Interface Gi0/3 on core-sw1 went down while it was administratively up.

Evidence:
- ifOperStatus went from up to down on ifIndex 3 at 2026-09-14T12:00:00Z.
- ifInErrors on Gi0/3 peaked at 12 per second in the 15 minutes before it went down.

Next check: Gi0/3 was counting errors before it went down, so the link is failing physically. Replace or reseat the cable and the optic on Gi0/3, then read the error counters of the far-end port.

Severity: critical
...
```

`Auto-Submitted` (RFC 3834) stops out-of-office replies from answering every
alert. Text a device controls cannot add a header: line breaks in the title are
folded into the subject line.

## Delivery behaviour

The same as the webhook: a 256-alert queue that drops (and records the drop)
rather than blocking detection, three attempts with linear backoff, and a 10
second bound per attempt. A `5xx` reply, an untrusted certificate, or a server
missing STARTTLS or AUTH is not retried — the same server will answer the same
way. A `4xx` reply or a network failure is.

Each alert's email outcome is recorded on the alert and served by
`GET /api/v1/alerts` as one entry of `deliveries`:

```json
"deliveries": [
  {"channel": "email", "status": "failed", "attemptedAt": "…",
   "error": "RCPT TO noc@example.com: 550 5.1.1 no such mailbox"},
  {"channel": "webhook", "status": "delivered", "attemptedAt": "…"}
]
```

An alert raised while no receiver was configured has no entry for that channel.

Alert delivery rides on the alert pipelines, which are Pro-tier engines. On Free
and Starter the pipelines do not run, so there are no alerts to deliver.
