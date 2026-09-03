# Editions

**Product:** Seed
**Status:** Current
**Owner:** Mustard Seed Networks
**Last updated:** 2026-09-03

Companion to [DISTRIBUTION.md](DISTRIBUTION.md), which covers how builds reach
an operator.

Seed ships as **one binary on three tiers**. There is no separate build, no
separate hardware SKU, and no edition string: what a deployment can do is
decided by the key it holds, and by nothing else.

> This document replaces a two-hardware-profile plan that described an
> `Edition` type, an `IsPro()` helper and a matching config field. None of
> that was ever built, and following it would have bypassed the enforcement
> that was (#2294).

---

## 1. Tiers

| Tier | Price | Key required |
| --- | --- | --- |
| **Free** | — | No. This is what an unlicensed install runs as. |
| **Starter** | $299/yr | Yes |
| **Pro** | $999/yr | Yes |

`Tier` is defined in [`internal/license/policy.go`](../internal/license/policy.go)
as `TierInvalid` / `TierFree` / `TierStarter` / `TierPro`. A signed key carries
a wire tier and a product code; Free carries no key, so an unrecognised or
unsigned token grants nothing rather than falling back to a paid tier.

A **14-day trial** grants Pro (`TrialDays`, `TrialTier`).

Commercial arrangements — custom terms, volume licensing, net-30 — are not a
fourth tier. They are the same binary and the same tiers, arranged through
`/contact`.

## 2. How entitlement is enforced

Two mechanisms, and only two:

- **`requireFeature`** ([`internal/api/middleware_license.go`](../internal/api/middleware_license.go))
  wraps a route with a feature name. Without the feature the route answers 402;
  the handler is never reached.
- **The feature catalogue** in `internal/license/policy.go` maps each tier to
  the feature names it grants. `starterFeatures()` is a list;
  `proFeatures()` is that list plus the Pro additions, so Pro is a superset by
  construction rather than by remembering to copy entries.

Write a Pro-only route as a `requireFeature("...")` at registration. Do not
add a tier check inside a handler: an entitlement decision that is not at the
route is one that a second caller can miss.

**The catalogue is not a feature list you can quote to a customer.** An audit
on 2026-09-02 found catalogue strings with no gating reference anywhere in Go —
some name capabilities that exist but are ungated, others name capabilities
that do not exist at all. Reconciling it, and adding a CI gate so it cannot
drift again, is #2327. Until that lands, read the catalogue as "what the
keygen and the binary agree on", not as "what Seed does".

## 3. Validation is local

Keys are validated **offline**, against a device fingerprint, using the shared
crypto in [`foundation/pkg/license`](https://github.com/MustardSeedNetworks/foundation).
Seed does not contact a licence server, at startup or ever. That is a
requirement rather than a preference: air-gapped clinical, industrial and
government deployments are a target, and a phone-home would disqualify Seed
from all three.

Consequences worth knowing:

- No grace window, because there is nothing to be unreachable.
- Rebinding a key to new hardware is an operator action through the portal, not
  something the daemon negotiates.
- `/__version` reports build metadata only. It does not report the tier, and a
  client must not infer entitlement from anything it can read without
  authenticating.

## 4. UX considerations

- **Minimum supported width: 480px.** Comfortable target: 768px+. This is the
  authoritative statement of that number and it is enforced by the E2E suite
  (#244) rather than only asserted here.
- **The first-run wizard and Settings must work without a mouse.** An operator
  who has just plugged the box in is often on a phone or tablet, on a network
  that is the reason they are there.

## 5. Open questions

| Question | Owner | Needed by |
| --- | --- | --- |
| Catalogue reconciliation: which unbacked strings are deleted and which get gates | Product | #2327 |
| Whether a Raspberry Pi image ships as a supported artifact | Eng | #22 |
| Self-serve key rebind when a device is replaced | Eng + Support | First replacement |
