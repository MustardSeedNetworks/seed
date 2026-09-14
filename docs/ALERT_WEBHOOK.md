# Outbound alert webhook

Seed detects, records and displays alerts. The webhook is the one way it
**sends** one: a signed JSON POST to a receiver you run, so Seed can be bridged
into the Slack, PagerDuty, or SIEM you already have.

It is deliberately one transport. There is no SMTP, no per-user delivery
preference, no escalation policy, no on-call schedule and no digest batching —
those belong to the notification system you already operate.

## Enabling it

The webhook is off until an operator configures it, and nothing is sent, logged
or retried while it is off. An air-gapped install loses the feature, never
function.

Two environment variables on the `seed` daemon turn it on:

| Variable | Meaning |
| --- | --- |
| `SEED_ALERT_WEBHOOK_URL` | Absolute `http`/`https` receiver URL. No userinfo (`https://user:pw@…` is refused). |
| `SEED_ALERT_WEBHOOK_SECRET` | Shared signing material for the HMAC. Required whenever the URL is set. |

With systemd, put them in an `EnvironmentFile` the unit reads, owned by root and
mode `0600`. The secret is deliberately **not** a config-file field: a signing
key in `config.json` is a key in every backup of `config.json`.

If the URL is set but unusable — malformed, wrong scheme, or no secret — Seed
logs an error at startup and runs without delivery. It does not refuse to start:
a typo in an optional integration must not take down the diagnostics the box was
installed for. Look for `alert webhook not configured` in the log.

On a successful start the log line is `alert webhook configured` with an
`endpoint` field holding scheme and host only. The path and query are withheld
because most receivers (Slack, Teams, PagerDuty) carry their per-channel secret
in the path.

Alert delivery rides on the alert pipelines, which are Pro-tier engines. On Free
and Starter the pipelines do not run, so there are no alerts to deliver.

## What arrives

One POST per alert, `Content-Type: application/json`:

```json
{
  "alert": {
    "id": 42,
    "type": "performance",
    "severity": "critical",
    "title": "Filesystem / critical on t-1",
    "message": "Usage crossed 95%: 97.0% of 1000 bytes",
    "source": "alert-observation-pipeline",
    "acknowledged": false,
    "resolved": false,
    "createdAt": "2026-09-14T12:00:00Z"
  },
  "sentAt": "2026-09-14T12:00:01Z"
}
```

The alert object is the same shape `GET /api/v1/alerts` returns. It is nested
under `alert` so later envelope fields cannot collide with an alert field.

Headers:

| Header | Value |
| --- | --- |
| `X-Seed-Signature` | `sha256=<hex>` — HMAC-SHA256, described below |
| `X-Seed-Timestamp` | Unix seconds at which the attempt was signed |
| `X-Seed-Alert-Id` | The alert's numeric id, for de-duplicating retries |

## Verifying the signature

The signed message is the timestamp, a literal `.`, then the exact request body:

```text
HMAC-SHA256(secret, "<X-Seed-Timestamp>" + "." + <raw body bytes>)
```

Compare in **constant time** (`hmac.compare_digest` in Python, `hmac.Equal` in
Go, `crypto.timingSafeEqual` in Node) — a byte-by-byte `==` leaks the expected
value one character at a time. Verify against the raw bytes you received, before
any JSON re-encoding: re-serializing changes key order and whitespace and the
signature will not match.

The timestamp is inside the signed message so a captured delivery cannot be
replayed indefinitely. Reject anything older than a few minutes.

```python
import hashlib, hmac

def verify(body: bytes, headers, secret: str) -> bool:
    signed = headers["X-Seed-Timestamp"].encode() + b"." + body
    expected = "sha256=" + hmac.new(secret.encode(), signed, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, headers["X-Seed-Signature"])
```

## Delivery behaviour

- **Never blocks detection.** Deliveries are queued and sent on their own
  goroutine. If the receiver is slow enough to fill the queue (256 pending), the
  overflow is dropped and counted. The alert is already in the inbox; the
  pipeline tick does not wait on someone else's endpoint.
- **Bounded retry, not a queue.** Three attempts per alert with linear backoff
  (0s, 2s, 4s). A receiver that is down loses the delivery — Seed is a
  diagnostic appliance, not a message broker.
- **5xx, 408 and 429 are retried; other 4xx are not.** A receiver that rejects
  this body will reject it again, and retrying only duplicates the load.
- **2xx is success.** Any 2xx counts as delivered; the response body is ignored.

Answer quickly and do the work asynchronously. Each attempt times out after 10
seconds, and a receiver that takes longer than the queue drains costs
deliveries.

## Receiver checklist

- Terminate TLS; the signature proves origin, not confidentiality.
- Treat `X-Seed-Alert-Id` as the idempotency key — a retry after a timeout can
  arrive twice.
- Return 2xx as soon as the delivery is durable, before you process it.
- Return a 4xx you mean: it stops the retries.
