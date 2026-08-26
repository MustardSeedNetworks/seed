# Benchmarks

Fifty `func Benchmark…` exist across four packages. Until #510 none of them had
ever run in CI — they compiled, and that was all. This describes what now runs,
and why it gates on the metric it does.

## Running them

```bash
make bench                                  # run and print
make bench-save BENCH_OUT=head.txt          # run and save
make bench-compare BASE=base.txt HEAD=head.txt
```

`BENCH_PKGS` in `mk/test.mk` lists the four packages that hold benchmarks.
Listing them beats `./...`, which still builds and runs a test binary for every
package and spends minutes producing nothing.

## The gate is on allocations, not on time

`scripts/bench-compare.py` fails on a regression in **`allocs/op`** and never on
`sec/op`. Timing is printed for humans to read; it is not a gate.

That split was measured rather than assumed. Running the full suite twice
against **identical code** on an idle M2 laptop, then comparing the two runs
with `benchstat`:

| metric | worst spread | benchmarks benchstat called "significantly changed" |
| --- | --- | --- |
| `sec/op` | ±612% | **10 of 40** |
| `B/op` | ±4% | 3 of 40 |
| `allocs/op` | ±1% | 3 of 40 |

Ten of forty benchmarks showed a statistically significant change (p < 0.05)
between two runs of the same code — including `-24.07%` at p=0.004 and `+25.39%`
at p=0.041. On a shared CI runner with noisy neighbours it is worse. A gate on
wall time would fail roughly a quarter of the suite on any given run.

That matters more than it looks. A gate that fires on noise gets ignored, then
disabled — which is precisely how fifty benchmarks came to sit in the tree
running never. A gate is only worth having if a red one means something.

Allocation counts, by contrast, are a property of the code and not of the
machine. They are integers, they compare exactly, and an extra allocation on a
hot path is a real regression worth blocking.

### The three that are not stable

`GetTopProcessesInternal`, `Health` and `HealthParallel` enumerate the host's
live process list, which allocates per process — so their allocation counts move
with whatever else is running on the machine. They are named in `HOST_DEPENDENT`
in `scripts/bench-compare.py` rather than covered by a wide blanket tolerance,
so that a genuinely unstable _new_ benchmark is not silently absorbed.

### Threshold

5%, which is five times the measured ±1% noise floor. Allocation counts are
integers, so a real regression clears it easily: one extra allocation in a
benchmark that made ten is +10%.

## In CI

The `bench` job runs on `pull_request` when Go files change, and gates merge
through `CI Complete`.

Base and head are benchmarked **on the same runner inside one job**. Comparing
across runners would reintroduce the machine-to-machine variance the whole
design exists to avoid. The baseline is the **merge base**, not the base branch
tip — comparing against a moving tip would attribute someone else's change to
this PR.

The full comparison is uploaded as the `bench-comparison` artifact, on failure
as well as success: it is most wanted exactly when the gate has just gone red.

## If the gate fails

Read `bench-verdict.txt` in the artifact. It names the benchmark and the
allocation delta. `allocs/op` is deterministic, so it is a real change in what
the code allocates — either justify it in the PR body or fix it. If a benchmark's
allocation count legitimately depends on the host, add it to `HOST_DEPENDENT`
with the reason.

## Explicitly not

Production performance telemetry. Seed is a real-time diagnostic; none of this
ships in the product binary's runtime path.
