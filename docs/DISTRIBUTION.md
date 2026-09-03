# Distribution

**Product:** Seed
**Status:** Current
**Last updated:** 2026-09-03

Companion to [EDITIONS.md](EDITIONS.md), which covers what a tier grants.
This document covers how a build reaches an operator.

> This replaces a document that described distribution as locked down and
> listed a Trial/Standard/Professional/OEM tier table. Neither was true: the
> repository is public, every tagged release has been published, and the tier
> model is the one in EDITIONS.md (#2294).

---

## 1. What ships

Releases are public on GitHub. Every tag produces, for each target:

| Target | Artifact |
| --- | --- |
| Linux amd64 / arm64 | `.deb`, `.rpm`, `.tar.gz` |
| macOS arm64 | `.tar.gz` |
| Windows amd64 / arm64 | `.zip` |

Alongside each artifact:

- a **cosign bundle** (`.cosign.bundle`) — keyless OIDC signature;
- an **SBOM** (`.sbom.json`, itself signed);
- `checksums.txt` and its bundle;
- one `seed-slsa-provenance.intoto.jsonl` for the release.

There is **no macOS x86-64 build**, **no container image** and **no Homebrew
tap**. The `brews:` block was removed on 2026-05-18 because the tap token was
never provisioned, and the reason is recorded in `.goreleaser.yml` so that
restoring it is a decision rather than an archaeology exercise.

## 2. How it is built

`release.yml` on a tag push, inside `goreleaser/goreleaser-cross:v1.27.0`
pinned by digest. The frontend is built outside the container and embedded, so
every published binary reports a non-empty `uiBuildHash` at `/__version` —
that field is how you tell a release build from a bare `go build`.

Versions come from git tags, and tags come from
[release-please](https://github.com/googleapis/release-please) reading
conventional commits. Nothing is tagged by hand.

**There is no local packaging target.** `make build` produces a host binary;
`.deb`, `.rpm`, `.tar.gz` and `.zip` are produced only by the release
workflow. A locally built package would not carry the release ldflags, the
signature or the SBOM, which is why the targets that used to exist were
removed rather than kept as a convenience.

## 3. Installing

```bash
# Debian / Ubuntu
sudo dpkg -i seed_<version>_amd64.deb

# RHEL / Fedora
sudo rpm -i seed-<version>-1.x86_64.rpm
```

Both install a systemd unit. Seed listens on `https://<host>:8443` and has
no plaintext listener; a browser sent to `http://` gets connection refused,
which is intended.

Verify what you installed before trusting it:

```bash
cosign verify-blob --bundle seed_<version>_amd64.deb.cosign.bundle \
  --certificate-identity-regexp 'https://github.com/MustardSeedNetworks/seed/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  seed_<version>_amd64.deb

curl -sk https://localhost:8443/__version | jq
```

`/__version` needs no authentication and returns `version`, `commit`,
`buildTime` and `uiBuildHash`. An empty `uiBuildHash` means the binary was not
built by the release pipeline.

## 4. Licensing at install time

Nothing to do. An unlicensed install runs as Free, and a key is applied
afterwards through the UI or `seed license`. Validation is local, so a Seed
that never reaches the internet works exactly like one that does — see
[EDITIONS.md §3](EDITIONS.md#3-validation-is-local).

## 5. Not yet true

Tracked rather than described as though it ships:

| Item | Issue |
| --- | --- |
| Raspberry Pi deployment documentation | #22 |
| Install verified from published artifacts on every supported platform | S7-1 in the v1 plan |
| Windows install validation (no host granted) | #2104 |
