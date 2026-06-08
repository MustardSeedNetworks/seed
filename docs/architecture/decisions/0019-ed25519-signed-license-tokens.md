# ADR-0019: Replace the forgeable rotor-cipher license key with Ed25519-signed tokens

**Status:** Accepted — 2026-06-08
**(Back-port of stem #409 / niac #802 to seed. seed is the third and final
product to migrate; the cross-product `MSN1` token format and the pre-launch
keypair are shared. Implemented: `internal/license/signing.go` +
`validator.go` rewrite; `cipher.go` deleted.)**

## Context

Seed's license keys were a 16-character rotor-cipher format (`internal/license/
cipher.go` + `validator.go`), byte-compatible with the same scheme in stem and
niac. The scheme had a fatal property: **the generator shipped inside every
binary**. `license.GenerateLicenseKey` plus the rotor + `CalculateChecksum`
self-checksum meant any copy of Seed could mint a key that its own validator
would accept. "Validation" was a self-consistency check (decode, recompute the
checksum, compare), not an authenticity check — there was no secret an attacker
lacked.

This is the exact vulnerability stem (#409) and niac (#802) closed by moving to
Ed25519-signed tokens. seed was deferred to the owner at the time (key-custody
decision) and is now back-ported to bring all three products onto the same
un-forgeable, offline scheme. The locked strategy
(`LICENSE_STRATEGY.md`) requires **local-only, no-phone-home** license
validation for air-gapped clinical/industrial/government customers, so the
replacement must preserve fully-offline verification.

## Decision

Adopt the cross-product `MSN1` signed-token format (frozen in
`internal/license/signing.go`, identical across seed/stem/niac):

```
MSN1.<base64url(payload)>.<base64url(signature)>
```

- `payload` is the canonical JSON of a `Payload` struct (`v`, `product`, `code`,
  `serial`, `tier`, `maxDevices`, `iat`, `exp`).
- `signature` is `ed25519.Sign(priv, payloadBytes)` over the **exact** signed
  bytes (verification does not re-marshal, so field order is irrelevant).
- The binary embeds **only the Ed25519 public key**
  (`licensePublicKeyB64`). The private key lives solely in the keygen tool
  (`msn-internal-tools/keygen`) and never ships.

Verification (`Verifier.Validate`) checks the signature **first**, then enforces
the product/tier/code matrix in-binary:

- `product` must equal `"seed"` — a correctly signed stem/niac token is rejected.
- `4001` → Starter (tier 1, `starterFeatures()`); `4002` → Pro (tier 2,
  `proFeatures()`). A code/tier mismatch is rejected.
- `exp`, when present, is honoured.

The tier values, product codes, and feature lists are unchanged from the
previous scheme, so `internal/timeseries/retention`, `internal/api`
tier-gating, and `cmd/seed/cmd_license.go` are source-compatible. `FormatKey`
is retained but now only trims whitespace (tokens are single-line and must not
have `-`/`_` stripped — base64url uses them). `GenerateLicenseKey`, the rotor
cipher, and the self-checksum are deleted.

`Manager` gains a `verifier` field. `NewManager`/`NewManagerWithDir` use the
embedded production key; `NewManagerWithVerifier(dir, v)` injects an ephemeral
key so tests can activate tokens without the production private key.

## Consequences

- **Un-forgeable, still offline.** A valid key now requires the keygen private
  key. No network call is introduced; verification is a local Ed25519 check.
- **No customer reissuance.** Pre-launch ⇒ no keys are in customers' hands, so
  rotating from the old scheme costs nothing.
- **Key custody / rotation.** The embedded pre-launch key
  (`O+o8n4qHHp/X//JrRXSdgGSWa2Fqz79OtgUkcylNxZg=`) was generated in a dev
  sandbox and its private half transited a tooling session, so it **must be
  rotated before GA** (keygen-side, `MSN_ED25519_LICENSE_SPEC.md`). When it
  rotates, the embedded key and the `TestKeygenContract` vectors in
  `license_test.go` are regenerated in lockstep.
- **Forge/tamper/expiry are tested fail-closed.** `license_test.go` proves an
  attacker-signed token, a payload-swapped token, a cross-product token, and an
  expired token are all rejected, and pins production-signed contract vectors
  that validate against the embedded key.

## Coordination

seed has concurrent UI/re-architecture sessions; this change is scoped entirely
to `internal/license/` + this ADR and touches no UI or API surface. The
issuance side (keygen) and the matching stem/niac validators are tracked in
`MSN_ED25519_LICENSE_SPEC.md`.
