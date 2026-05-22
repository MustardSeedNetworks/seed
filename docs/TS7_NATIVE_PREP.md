# TypeScript 7 Native Compiler — Adoption Plan

**Status:** Research / prep only. No package.json changes yet.
**Tracked:** Wave 5 / task #32 (cross-repo).
**Author:** seed / Wave 5 prep, 2026-05-22.

## Background

Microsoft is rewriting `tsc` in Go ("TypeScript Native", project codename `tsc-go`). The
in-development port ships as `@typescript/native-preview` and exposes the binary as
`tsgo`. Goals:

- 10×-class speedups on cold + warm type-checks (Go binary, no Node/V8 startup,
  parallel type checker, no garbage collection on the hot path).
- API parity with the JS `tsc` for the surfaces our toolchain hits (`--noEmit`,
  `--project`, error format, exit codes).
- No source code changes — TypeScript source compiles identically; only the
  compiler binary changes.

The native compiler is targeted to ship as **TypeScript 7.0** GA. As of 2026-05-22
the latest preview is `7.0.0-dev.20260521.1` — daily build, no firm GA date.

## What we measured

Same `tsc --noEmit -p tsconfig.json` across all 3 UI repos, on the same M2 MacBook,
warm cache, with biome and prettier idle:

| Repo | files (rough) | `tsc` (JS) | `tsgo` (native) | speedup |
|------|---|------------|-----------------|---------|
| niac | ~230 | 0.30s | 0.44s | ~equal (codebase too small to amortize the Go startup) |
| stem | ~620 | 2.58s | 0.76s | **3.4×** |
| seed | ~1700 | 14.40s | 2.37s | **6.1×** |

**Error output is identical** between the two compilers. The seed run surfaced two
pre-existing errors (the same in both); the native compiler did not introduce false
positives or miss anything `tsc` caught.

## Adoption plan

### Phase 0 — now (this PR)

- Document the measurement above.
- File a tracking issue per repo.
- **Do NOT** add `@typescript/native-preview` to `devDependencies`. The package
  is a daily dev build (`-dev.YYYYMMDD.N`); pinning to a daily moves goalposts
  every time someone runs `npm install`.

### Phase 1 — when MS promotes a release-candidate dist-tag

When `npm view @typescript/native-preview dist-tags` lists `rc` or `next`:

- Add `@typescript/native-preview@<rc-version>` to seed's devDependencies (exact
  pin per CLAUDE.md).
- Add a `typecheck:fast` script to `ui/package.json`:
  ```json
  "typecheck:fast": "tsgo --noEmit -p tsconfig.json"
  ```
- Leave the existing `typecheck` (`tsc --noEmit`) untouched; CI keeps using it as
  the source of truth.
- Devs opt in for the dev loop via `npm run typecheck:fast`.

### Phase 2 — at TypeScript 7.0 GA

- Replace the `typescript` devDep with `typescript@7.0.0` (the native binary
  becomes the primary `tsc` at that point).
- Drop `@typescript/native-preview` (folded into `typescript`).
- Update CLAUDE.md required-versions: TypeScript `7.0.0`.
- Drop the `typecheck:fast` script — it's now the same as `typecheck`.

### Phase 3 — CI

Once Phase 2 ships and stays stable through one release cycle:

- Bump the CI Type-check step to use the 7.0 binary. Expected wins:
  - seed Frontend job's `tsc` step drops ~12s of wall time per PR.
  - stem similarly drops ~1.8s.
  - niac is small enough that it's noise — net-zero.
- Update `.github/actions/setup-node` cache key if the cache shape changes.

## Risks / things to watch

1. **TypeScript language version coupling.** The native compiler is targeted to
   ship as TypeScript 7.0; that release will also bring TS language changes
   (deprecated APIs removed, new ones added). Adoption needs to be paired with
   reviewing TS 7.0 release notes for syntax/typing regressions.
2. **IDE integration.** VS Code's TS server still uses `tsserver` (the JS
   implementation) for in-editor IntelliSense as of writing. Editor performance
   doesn't change until VS Code adopts the native server too — separate timeline.
3. **Project references / build mode.** We don't use `tsc -b` heavily; if we
   adopt it, validate behavior on tsgo first.
4. **Daily-build instability.** Today's `7.0.0-dev.20260521.1` may regress on
   `7.0.0-dev.20260522.1`. That's why Phase 0 is doc-only — we wait for `rc`.
5. **Biome dependency.** Biome 2.4.15 has a stack-overflow bug pinned around
   (we use 2.4.10). The TS 7 adoption is unrelated, but a parallel issue worth
   tracking: a buggy upstream that can break our toolchain on auto-upgrade.

## Sibling tracking

- stem: filed as a tracking issue.
- niac: filed as a tracking issue.

Both will reference this document. Phase 1+2+3 land identically across all three
repos, but each repo can opt in independently if one is more time-sensitive.
