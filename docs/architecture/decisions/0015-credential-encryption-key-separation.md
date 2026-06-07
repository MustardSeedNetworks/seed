# ADR-0015: Separate the credential data-encryption key from `Auth.JWTSecret`

**Status:** Accepted — 2026-06-06
**(Owner greenlit for v1. Cross-workstream sign-off satisfied — see "Coordination" below. Implemented: `internal/config/keyring.go` + `crypto.go`; the two non-signing `JWTSecret` consumers now use the DEK, with the legacy v0 read path retained for one release.)**

## Context

SNMP v3 credentials (auth/priv passwords) are encrypted at rest with AES-256-GCM
before being persisted into the config (`#518`). The implementation in
`internal/config/crypto.go` has three structural problems that an independent
architecture review (2026-06-05, finding #4, verified true) flagged as the
highest-severity item on the hardening list:

1. **The master key *is* the JWT signing secret.** Every call site derives the
   encryption key from `c.Auth.JWTSecret`:
   - `crypto.go` — `EncryptSNMPCredentials` / `DecryptSNMPPassword`
   - `internal/api/handlers_security.go:522,534` — `EncryptCredential(..., s.config.Auth.JWTSecret)`

   This couples two unrelated security domains: **session-token signing** (owned
   by the `internal/auth` / `internal/oauth` workstream) and **data-at-rest
   encryption** (owned by config). The coupling has concrete failure modes:
   - Rotating `JWTSecret` (a routine, *expected* auth operation — e.g. on
     suspected token-forgery) silently makes every stored SNMP credential
     **undecryptable**. There is no migration path; the data is simply lost.
   - A single secret leak compromises *both* domains at once instead of one.
   - The two domains have different rotation cadences, blast radii, and owners,
     yet share one value — an SRP violation at the security layer.

2. **No real KDF.** `deriveKey(masterSecret) = sha256(masterSecret)[:32]` — a
   bare single-pass hash, **no salt, no domain separation, no KDF**. A bare hash
   is not a key-derivation function: it offers no multi-target defence (the same
   secret always yields the same key, so one precomputed table covers every
   deployment) and no separation between "the JWT key" and "the encryption key"
   derived from the same input.

3. **No key versioning.** Ciphertext format is `enc:base64(nonce‖ct‖tag)` with
   no version tag, so the key that produced any given ciphertext is implicit and
   indeterminate. **Key rotation is impossible** without a flag day — there is no
   way for two keys to coexist during a migration window.

Constraints this decision operates under:

- **No phone-home / air-gapped support is mandatory** (`LICENSE_STRATEGY.md`).
  No external KMS may be *required*; the default must work fully offline on a
  diagnostic appliance.
- The config file is persisted as JSON (`config_load.go`). Today `JWTSecret`
  *and* the ciphertext it protects both live in that one file — so at-rest
  encryption currently protects only against *partial* leakage (a log line, a
  truncated backup), not against anyone who has the whole config file. Any fix
  that keeps the key in the same file inherits that weakness.

Out of scope: `internal/license/activation.go`'s `deriveKey` is a **separate,
intentionally device-fingerprint-bound** key domain (license state at rest,
keyed to `fingerprint.Hash() + salt`). It is not data-at-rest *credential*
encryption, is not coupled to `JWTSecret`, and is unaffected by this ADR.

## Decision

Introduce a dedicated, persisted, versioned **Data-Encryption Key (DEK)** for
credential encryption, fully decoupled from `Auth.JWTSecret`.

### 1. Dedicated key material, stored separately from the data it protects

- The DEK is **32 bytes of CSPRNG output**, generated on first use.
- It is persisted in a **dedicated key file**, *not* in `config.json` — default
  `<datadir>/credential.key`, mode `0600`, owned by the `seed` service user.
  Storing the key in the same file as the ciphertext (as `JWTSecret` is today)
  gives no defence against full-file leakage; a separate, tighter-permissioned
  file means a leaked *config* backup no longer leaks the key.
- Override order (first present wins), for 12-factor and BYO-KMS deployments:
  1. `SEED_CREDENTIAL_KEY` — base64-encoded 32-byte key (operator-/secrets-
     manager-supplied; never written to disk).
  2. `<datadir>/credential.key` — the generated, persisted default.
- The key file is a small JSON **keyring**, not a bare blob, to carry versioning
  and per-version salt (see §3).

### 2. Real KDF with salt and domain separation

- The per-version AES-256 key is derived via **HKDF-SHA256** (RFC 5869) from the
  stored 32-byte master, with a **per-version random salt** (16 bytes, stored in
  the keyring) and a fixed `info` domain-separation label
  (`"seed:snmp-credential-encryption:v<N>"`).
- HKDF is the correct primitive here because the input is **high-entropy random
  key material**, not a human passphrase. Argon2id is reserved for the single
  case where the master is supplied as a low-entropy operator *passphrase*
  (a future option, not the default); the keyring records which was used.
- The salt provides multi-target defence and clean separation from any other
  key ever derived from the same material.

### 3. Versioned ciphertext format enabling rotation

- New format: **`enc:v<N>:base64(nonce‖ciphertext‖tag)`**, where `<N>` selects
  the keyring entry (and thus its salt) used to derive the key.
- Legacy **`enc:base64(...)`** (no version segment) is interpreted as **v0** —
  the JWT-derived, unsalted path — and remains *decryptable* (read-only) for
  migration.
- The keyring records the **active version** (used for all new encryption) and
  retains older versions for decryption, so rotation = "add v(N+1), make it
  active, lazily re-encrypt, retire v(N)".

### 4. Transparent one-time migration off the JWT-derived key

- On config load (and on save), any value still in the **v0** format is:
  decrypted via the legacy `JWTSecret`-derived path → re-encrypted with the
  active DEK version → persisted. No operator action; idempotent.
- The legacy v0 read path is retained for **one release** after the migration
  ships, then removed (tracked as a follow-up). After removal, `JWTSecret` has
  zero involvement in credential encryption.

### 5. Auth-boundary outcome

- `internal/config/crypto.go` and `internal/api/handlers_security.go` **stop
  reading `Auth.JWTSecret`** for encryption. The new API takes a `CredentialKey`
  (or a small `Keyring` service), not the JWT secret.
- `Auth.JWTSecret` reverts to a **single responsibility**: signing/verifying JWTs
  (owned by the `internal/auth` / `internal/oauth` workstream). It becomes
  independently rotatable with **no data-loss side effect**.

## Coordination (the cross-workstream gate) — RESOLVED 2026-06-06

`internal/auth` / `internal/oauth` is a **separate workstream**; this ADR touches
the `Auth.JWTSecret` boundary. The gate below is **satisfied**: the audit
confirmed the only non-signing `JWTSecret` consumers were the two
credential-encryption sites (now swapped to the DEK), and `JWTSecret` reverts to
JWT signing only. New encryption no longer reads `JWTSecret`; it remains
referenced solely by the read-only legacy v0 decrypt path, removed one release
after migration. Original sign-off criteria:

1. **No non-signing consumers of `JWTSecret` remain** outside auth. (This ADR's
   audit found only credential encryption; auth confirms there are no others —
   e.g. cookie/session/CSRF secrets must have their own key material, not borrow
   `JWTSecret`.)
2. **`JWTSecret` rotation becomes independent** of credential encryption — auth
   acknowledges this is now safe and is the intended boundary.
3. The DEK is a **config/data-layer** concern (new key file + keyring service),
   not an auth concern; auth owns `JWTSecret` lifecycle only.

Once aligned, implementation proceeds as a small, test-first slice (keyring +
KDF + versioned format + migration + call-site swap), behind the golden HTTP
harness and the existing config tests, with no wire/DTO change.

## Consequences

- **Decoupled blast radius.** A `JWTSecret` leak no longer exposes stored
  credentials, and vice versa; rotating either is independent.
- **Rotatable encryption.** Versioned ciphertext + keyring make DEK rotation a
  normal operation instead of a flag day.
- **Stronger at-rest posture.** Separate-file key with `0600` perms means a
  leaked config (logs, backups, support bundles) does not also leak the key.
  This is a real improvement only to the extent the key file is held more tightly
  than the config — documented as an operator expectation; BYO-KMS via
  `SEED_CREDENTIAL_KEY` is the path for stricter environments.
- **Migration cost is bounded and transparent** — lazy re-encryption on
  load/save; no operator step; legacy read path removed one release later.
- **New operational surface**: a key file to back up *and protect*. Losing it
  makes stored credentials unrecoverable (by design — that is what
  encryption-at-rest means). Documented in the operator setup checklist;
  re-entry of SNMP v3 credentials is the recovery path.
- **Not microservices, not external-KMS-required** — consistent with the
  capability-first modular-monolith direction and the air-gapped constraint;
  KMS is opt-in, not assumed.
