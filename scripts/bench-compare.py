#!/usr/bin/env python3
"""Compare two benchstat runs and fail on an allocation regression.

Gates on allocs/op, not on sec/op. That split is measured, not assumed: running
the suite twice against identical code (see docs/BENCHMARKS.md) produced

    sec/op      worst spread +/-612%   10 of 40 benchmarks "significantly" changed
    B/op        worst spread +/-4%      3 of 40
    allocs/op   worst spread +/-1%      3 of 40

on an idle laptop. A shared CI runner is worse. Gating on wall time would fail a
quarter of the suite on any given run, and a gate that cries wolf gets disabled —
which is how the previous 50 benchmarks ended up never running at all.

Allocation counts are a property of the code rather than of the machine, so they
compare exactly. The three unstable entries are the host-process benchmarks whose
allocation count legitimately tracks how many processes are running; they are
named in HOST_DEPENDENT rather than covered by a blanket tolerance.
"""

import re
import sys

# Benchmarks whose allocation count depends on the machine's live process list,
# not on our code. Enumerating processes allocates per process, so the count
# moves with whatever else is running. Named individually so that a genuinely
# unstable *new* benchmark is not silently absorbed.
HOST_DEPENDENT = frozenset({
    "GetTopProcessesInternal",
    "Health",
    "HealthParallel",
})

# Five times the measured +/-1% noise floor. Allocation counts are integers, so a
# real regression on a hot path clears this comfortably: one extra allocation in
# a benchmark that made ten is +10%.
THRESHOLD_PCT = 5.0

# benchstat marks a comparison it considers statistically meaningful with a
# p-value; "~" means it found no difference worth reporting.
ROW = re.compile(
    r"^(?P<name>\S+?)-\d+\s+.*?(?P<delta>[-+]\d+\.\d+)%\s*\(p=(?P<p>\d\.\d+)"
)
SECTION = re.compile(r"│\s*(?P<metric>sec/op|B/op|allocs/op)\s*│")


def parse(path):
    """Yield (metric, benchmark, delta_pct, p_value) for each compared row."""
    metric = None
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            header = SECTION.search(line)
            if header and "vs base" in line:
                metric = header.group("metric")
                continue
            if metric is None:
                continue
            row = ROW.match(line.strip())
            if row:
                yield (metric, row.group("name"),
                       float(row.group("delta")), float(row.group("p")))


def main(path):
    regressions = []
    reported = []
    for metric, name, delta, p_value in parse(path):
        base = name.split("/", 1)[0]
        if metric == "sec/op":
            reported.append((name, delta, p_value))
            continue
        if metric != "allocs/op" or base in HOST_DEPENDENT:
            continue
        if delta > THRESHOLD_PCT:
            regressions.append((name, delta, p_value))

    if reported:
        print("Timing changes (reported, never gated — see the module docstring):")
        for name, delta, p_value in sorted(reported, key=lambda r: -abs(r[1])):
            print(f"  {name:52s} {delta:+7.2f}%  (p={p_value})")
        print()

    if not regressions:
        print(f"No allocation regression above {THRESHOLD_PCT}%.")
        return 0

    print(f"Allocation regressions above {THRESHOLD_PCT}%:")
    for name, delta, p_value in sorted(regressions, key=lambda r: -r[1]):
        print(f"  {name:52s} {delta:+7.2f}% allocs/op  (p={p_value})")
    print()
    print("allocs/op is deterministic; this is a real change in what the code")
    print("allocates, not measurement noise. Either justify it in the PR body or")
    print("fix it. If the benchmark's allocation count legitimately depends on the")
    print("host, add it to HOST_DEPENDENT in this script with the reason.")
    return 1


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: bench-compare.py <benchstat-output>", file=sys.stderr)
        sys.exit(2)
    sys.exit(main(sys.argv[1]))
