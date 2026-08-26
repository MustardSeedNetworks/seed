#!/usr/bin/env python3
"""Self-tests for the benchmark regression gate.

The gate's value rests on two claims: it stays silent on timing noise, and it
fires on an allocation regression. Both are asserted here against real benchstat
output, so a change to the parser cannot quietly turn the gate into a no-op.
"""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


COMPARER = Path(__file__).with_name("bench-compare.py")

# Real benchstat output, captured from two runs of identical code. Every one of
# these timing rows is noise, and the gate must ignore all of them.
NOISE = """goos: darwin
goarch: arm64
pkg: github.com/MustardSeedNetworks/seed/internal/diagnostics/link
cpu: Apple M2
                                    │ /tmp/noise-a.txt │          /tmp/noise-b.txt           │
                                    │      sec/op      │    sec/op      vs base              │
ParseDuplex-8                             10.15n ±  6%   10.44n ±   4%       ~ (p=0.240 n=6)
ParseState-8                              13.30n ± 32%   12.01n ±   3%       ~ (p=0.065 n=6)
GetFlapCount-8                            251.8n ±  4%   249.9n ±  29%       ~ (p=0.818 n=6)
ListInterfaces-8                          55.55m ± 25%   56.30m ±  33%       ~ (p=0.937 n=6)
GetStatus-8                               2.817m ± 20%   2.638m ± 108%       ~ (p=0.589 n=6)
IsPhysicalInterface-8                     14.84n ±  4%   15.43n ±  11%       ~ (p=0.093 n=6)
CheckAndNotify-8                          46.51µ ± 47%   46.17µ ±   9%       ~ (p=0.937 n=6)
CheckAndNotifyWithCallbacks-8             45.51µ ± 33%   42.69µ ±   3%  -6.20% (p=0.002 n=6)
WaitForState-8                            14.20n ±  5%   13.99n ±   2%       ~ (p=0.394 n=6)
DarwinCheckLinkStatePlatform-8            47.36µ ± 91%   44.64µ ±   3%       ~ (p=0.132 n=6)
DarwinGetSpeedDuplex-8                    2.918m ± 13%   2.716m ±  53%       ~ (p=0.180 n=6)
DarwinIsPhysicalInterfacePlatform-8       15.15n ± 14%   15.30n ±   5%       ~ (p=0.909 n=6)
DarwinParseSpeedPlatform-8                3.948µ ±  7%   3.875µ ±  12%       ~ (p=0.937 n=6)
GetState-8                                12.89n ±  9%   13.53n ±  12%       ~ (p=0.132 n=6)
IsUp-8                                    12.69n ±  2%   13.19n ±   8%       ~ (p=0.370 n=6)
SpeedString-8                             232.8n ±  3%   237.2n ±   9%       ~ (p=0.310 n=6)
GetHistory-8                              453.4n ±  7%   417.2n ±  25%       ~ (p=0.093 n=6)
geomean                                   1.409µ         1.384µ         -1.74%

                                    │ /tmp/noise-a.txt │           /tmp/noise-b.txt           │
                                    │       B/op       │     B/op      vs base                │
ParseDuplex-8                             0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
ParseState-8                              0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetFlapCount-8                            0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
ListInterfaces-8                        1.380Mi ± 3%     1.403Mi ± 2%       ~ (p=0.240 n=6)
GetStatus-8                             85.86Ki ± 2%     85.04Ki ± 3%       ~ (p=0.394 n=6)
IsPhysicalInterface-8                     0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
CheckAndNotify-8                        18.88Ki ± 0%     18.88Ki ± 0%       ~ (p=1.000 n=6) ¹
CheckAndNotifyWithCallbacks-8           18.88Ki ± 0%     18.88Ki ± 0%       ~ (p=1.000 n=6) ¹
WaitForState-8                            0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
DarwinCheckLinkStatePlatform-8          18.88Ki ± 0%     18.88Ki ± 0%       ~ (p=1.000 n=6) ¹
DarwinGetSpeedDuplex-8                  66.62Ki ± 2%     66.89Ki ± 1%       ~ (p=0.818 n=6)
DarwinIsPhysicalInterfacePlatform-8       0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
DarwinParseSpeedPlatform-8              9.845Ki ± 0%     9.850Ki ± 0%       ~ (p=0.667 n=6)
GetState-8                                0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
IsUp-8                                    0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
SpeedString-8                             32.00 ± 0%       32.00 ± 0%       ~ (p=1.000 n=6) ¹
GetHistory-8                            4.000Ki ± 0%     4.000Ki ± 0%       ~ (p=1.000 n=6) ¹
geomean                                              ²                 +0.07%               ²
¹ all samples are equal
² summaries must be >0 to compute geomean

                                    │ /tmp/noise-a.txt │          /tmp/noise-b.txt           │
                                    │    allocs/op     │  allocs/op   vs base                │
ParseDuplex-8                             0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
ParseState-8                              0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetFlapCount-8                            0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
ListInterfaces-8                         4.661k ± 0%     4.663k ± 0%       ~ (p=0.621 n=6)
GetStatus-8                               388.0 ± 0%      388.0 ± 0%       ~ (p=1.000 n=6) ¹
IsPhysicalInterface-8                     0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
CheckAndNotify-8                          175.0 ± 0%      175.0 ± 0%       ~ (p=1.000 n=6) ¹
CheckAndNotifyWithCallbacks-8             175.0 ± 0%      175.0 ± 0%       ~ (p=1.000 n=6) ¹
WaitForState-8                            0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
DarwinCheckLinkStatePlatform-8            175.0 ± 0%      175.0 ± 0%       ~ (p=1.000 n=6) ¹
DarwinGetSpeedDuplex-8                    211.0 ± 0%      211.0 ± 0%       ~ (p=1.000 n=6) ¹
DarwinIsPhysicalInterfacePlatform-8       0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
DarwinParseSpeedPlatform-8                88.00 ± 0%      88.00 ± 0%       ~ (p=1.000 n=6) ¹
GetState-8                                0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
IsUp-8                                    0.000 ± 0%      0.000 ± 0%       ~ (p=1.000 n=6) ¹
SpeedString-8                             4.000 ± 0%      4.000 ± 0%       ~ (p=1.000 n=6) ¹
GetHistory-8                              1.000 ± 0%      1.000 ± 0%       ~ (p=1.000 n=6) ¹
geomean                                              ²                +0.00%               ²
¹ all samples are equal
² summaries must be >0 to compute geomean

pkg: github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan
                                                    │ /tmp/noise-a.txt │           /tmp/noise-b.txt           │
                                                    │      sec/op      │    sec/op      vs base               │
GetVlanInfoVariations/en0-8                              3.493µ ±  39%   2.526µ ±  45%        ~ (p=0.180 n=6)
GetVlanInfoVariations/vlan100-8                          2.547µ ±  88%   2.337µ ± 187%        ~ (p=0.937 n=6)
GetVlanInfoVariations/vlan4094-8                         2.556µ ±  15%   2.717µ ±  28%        ~ (p=0.589 n=6)
GetVlanInfoVariations/lo0-8                              2.414µ ± 146%   2.306µ ± 103%        ~ (p=0.937 n=6)
GetVlanInfoVariations/bridge0-8                          3.353µ ± 612%   2.759µ ± 107%        ~ (p=0.937 n=6)
DetectVlanSubinterfacesPlatformVariations/en0-8          48.98µ ± 176%   44.72µ ±  23%        ~ (p=0.093 n=6)
DetectVlanSubinterfacesPlatformVariations/en1-8          50.43µ ±  68%   42.72µ ±   6%  -15.29% (p=0.004 n=6)
DetectVlanSubinterfacesPlatformVariations/lo0-8          50.40µ ±  10%   42.46µ ±   8%  -15.75% (p=0.004 n=6)
DetectVlanSubinterfacesPlatformVariations/bridge0-8      59.71µ ± 132%   45.34µ ±  15%  -24.07% (p=0.004 n=6)
DetectVlanSubinterfacesPlatformVariations/awdl0-8        49.26µ ±  11%   44.00µ ±   4%  -10.67% (p=0.026 n=6)
CreateDeleteVlanInterface/Create-8                       2.006n ±   9%   1.932n ±   1%   -3.66% (p=0.002 n=6)
CreateDeleteVlanInterface/Delete-8                       1.960n ±   5%   2.050n ±   6%   +4.64% (p=0.041 n=6)
GetVlanInfo/non-vlan_interface-8                         2.251µ ±  37%   2.822µ ±  38%  +25.39% (p=0.041 n=6)
GetVlanInfo/vlan_interface-8                             1.899µ ± 100%   2.729µ ±  51%        ~ (p=0.132 n=6)
DetectVlanSubinterfacesPlatform-8                        44.87µ ±  32%   46.39µ ±  13%        ~ (p=0.937 n=6)
geomean                                                  3.264µ          3.099µ          -5.04%

                                                    │ /tmp/noise-a.txt │           /tmp/noise-b.txt           │
                                                    │       B/op       │     B/op      vs base                │
GetVlanInfoVariations/en0-8                               0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/vlan100-8                           0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/vlan4094-8                          0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/lo0-8                               0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/bridge0-8                           0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/en0-8         18.81Ki ± 0%     18.81Ki ± 0%       ~ (p=1.000 n=6)
DetectVlanSubinterfacesPlatformVariations/en1-8         18.81Ki ± 0%     18.81Ki ± 0%       ~ (p=1.000 n=6)
DetectVlanSubinterfacesPlatformVariations/lo0-8         18.81Ki ± 0%     18.81Ki ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/bridge0-8     18.81Ki ± 0%     18.81Ki ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/awdl0-8       18.81Ki ± 0%     18.81Ki ± 0%       ~ (p=1.000 n=6) ¹
CreateDeleteVlanInterface/Create-8                        0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
CreateDeleteVlanInterface/Delete-8                        0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfo/non-vlan_interface-8                          0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfo/vlan_interface-8                              0.000 ± 0%       0.000 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatform-8                       18.81Ki ± 0%     18.81Ki ± 0%       ~ (p=1.000 n=6) ¹
geomean                                                              ²                 +0.00%               ²
¹ all samples are equal
² summaries must be >0 to compute geomean

                                                    │ /tmp/noise-a.txt │          /tmp/noise-b.txt          │
                                                    │    allocs/op     │ allocs/op   vs base                │
GetVlanInfoVariations/en0-8                               0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/vlan100-8                           0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/vlan4094-8                          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/lo0-8                               0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfoVariations/bridge0-8                           0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/en0-8           174.0 ± 0%     174.0 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/en1-8           174.0 ± 0%     174.0 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/lo0-8           174.0 ± 0%     174.0 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/bridge0-8       174.0 ± 0%     174.0 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatformVariations/awdl0-8         174.0 ± 0%     174.0 ± 0%       ~ (p=1.000 n=6) ¹
CreateDeleteVlanInterface/Create-8                        0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
CreateDeleteVlanInterface/Delete-8                        0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfo/non-vlan_interface-8                          0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
GetVlanInfo/vlan_interface-8                              0.000 ± 0%     0.000 ± 0%       ~ (p=1.000 n=6) ¹
DetectVlanSubinterfacesPlatform-8                         174.0 ± 0%     174.0 ± 0%       ~ (p=1.000 n=6) ¹
geomean                                                              ²               +0.00%               ²
¹ all samples are equal
² summaries must be >0 to compute geomean

pkg: github.com/MustardSeedNetworks/seed/internal/reporting/aggregator
                │ /tmp/noise-a.txt │         /tmp/noise-b.txt          │
                │      sec/op      │   sec/op     vs base              │
Calculate-8           65.48µ ± 16%   63.57µ ± 4%       ~ (p=0.589 n=6)
Percentile-8          14.88µ ±  7%   14.19µ ± 6%       ~ (p=0.699 n=6)
MovingAverage-8       23.00µ ± 10%   22.11µ ± 4%       ~ (p=0.180 n=6)
geomean               28.19µ         27.12µ       -3.78%

                │ /tmp/noise-a.txt │           /tmp/noise-b.txt           │
                │       B/op       │     B/op      vs base                │
Calculate-8           80.00Ki ± 0%   80.00Ki ± 0%       ~ (p=1.000 n=6)
Percentile-8          80.00Ki ± 0%   80.00Ki ± 0%       ~ (p=1.000 n=6) ¹
MovingAverage-8       80.00Ki ± 0%   80.00Ki ± 0%       ~ (p=1.000 n=6) ¹
geomean               80.00Ki        80.00Ki       +0.00%
¹ all samples are equal

                │ /tmp/noise-a.txt │          /tmp/noise-b.txt          │
                │    allocs/op     │ allocs/op   vs base                │
Calculate-8             1.000 ± 0%   1.000 ± 0%       ~ (p=1.000 n=6) ¹
Percentile-8            1.000 ± 0%   1.000 ± 0%       ~ (p=1.000 n=6) ¹
MovingAverage-8         1.000 ± 0%   1.000 ± 0%       ~ (p=1.000 n=6) ¹
geomean                 1.000        1.000       +0.00%
¹ all samples are equal

pkg: github.com/MustardSeedNetworks/seed/internal/system
                          │ /tmp/noise-a.txt │          /tmp/noise-b.txt           │
                          │      sec/op      │    sec/op     vs base               │
GetTopProcessesInternal-8       18.90m ± 23%   19.49m ± 10%        ~ (p=0.310 n=6)
Health-8                        32.24µ ± 35%   29.33µ ±  2%   -9.04% (p=0.002 n=6)
HealthParallel-8                44.55µ ± 12%   39.75µ ±  5%  -10.78% (p=0.002 n=6)
HealthJSONMarshal-8             804.0n ± 16%   741.8n ±  7%        ~ (p=0.132 n=6)
ProcessInfoJSONMarshal-8        280.3n ±  3%   303.6n ± 11%        ~ (p=0.394 n=6)
geomean                         22.77µ         21.97µ         -3.51%

                          │ /tmp/noise-a.txt │           /tmp/noise-b.txt            │
                          │       B/op       │     B/op      vs base                 │
GetTopProcessesInternal-8       3.988Mi ± 0%   3.952Mi ± 0%   -0.89% (p=0.002 n=6)
Health-8                        2.504Ki ± 3%   1.945Ki ± 0%  -22.32% (p=0.002 n=6)
HealthParallel-8                2.746Ki ± 4%   1.960Ki ± 0%  -28.63% (p=0.002 n=6)
HealthJSONMarshal-8               320.0 ± 0%     320.0 ± 0%        ~ (p=1.000 n=6) ¹
ProcessInfoJSONMarshal-8          128.0 ± 0%     128.0 ± 0%        ~ (p=1.000 n=6) ¹
geomean                         4.056Ki        3.598Ki       -11.29%
¹ all samples are equal

                          │ /tmp/noise-a.txt │           /tmp/noise-b.txt           │
                          │    allocs/op     │  allocs/op   vs base                 │
GetTopProcessesInternal-8        30.78k ± 0%   30.40k ± 0%   -1.22% (p=0.002 n=6)
Health-8                          39.00 ± 0%    35.00 ± 0%  -10.26% (p=0.002 n=6)
HealthParallel-8                  40.50 ± 1%    35.00 ± 0%  -13.58% (p=0.002 n=6)
HealthJSONMarshal-8               1.000 ± 0%    1.000 ± 0%        ~ (p=1.000 n=6) ¹
ProcessInfoJSONMarshal-8          2.000 ± 0%    2.000 ± 0%        ~ (p=1.000 n=6) ¹
geomean                           39.59         37.53        -5.19%
¹ all samples are equal
"""

# Real benchstat output from a deliberately injected extra allocation.
REGRESSION = """goos: darwin
goarch: arm64
pkg: github.com/MustardSeedNetworks/seed/internal/reporting/aggregator
cpu: Apple M2
             │ /tmp/reg-a.txt │        /tmp/reg-b.txt        │
             │     sec/op     │   sec/op     vs base         │
Percentile-8    16.56µ ± 214%   18.95µ ± 4%  ~ (p=0.394 n=6)

             │ /tmp/reg-a.txt │            /tmp/reg-b.txt             │
             │      B/op      │     B/op       vs base                │
Percentile-8     80.00Ki ± 0%   160.02Ki ± 0%  +100.03% (p=0.002 n=6)

             │ /tmp/reg-a.txt │           /tmp/reg-b.txt           │
             │   allocs/op    │ allocs/op   vs base                │
Percentile-8       1.000 ± 0%   3.000 ± 0%  +200.00% (p=0.002 n=6)
"""

# Same allocation regression, but on a benchmark whose allocation count tracks
# the host's live process list rather than our code.
HOST_DEPENDENT = REGRESSION.replace("Percentile-8", "GetTopProcessesInternal-8")


class BenchCompareTest(unittest.TestCase):
    def run_gate(self, report: str) -> subprocess.CompletedProcess[str]:
        with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False) as handle:
            handle.write(report)
            path = handle.name
        try:
            return subprocess.run(
                [sys.executable, str(COMPARER), path],
                capture_output=True, text=True, check=False,
            )
        finally:
            Path(path).unlink()

    def test_timing_noise_does_not_fail_the_build(self) -> None:
        """Real swings of -24% and +25% at p<0.05 must not gate."""
        result = self.run_gate(NOISE)
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("No allocation regression", result.stdout)

    def test_timing_is_still_reported(self) -> None:
        """Not gating is not the same as hiding — humans still want to see it."""
        result = self.run_gate(NOISE)
        self.assertIn("-24.07%", result.stdout)
        self.assertIn("+25.39%", result.stdout)

    def test_allocation_regression_fails_the_build(self) -> None:
        result = self.run_gate(REGRESSION)
        self.assertEqual(result.returncode, 1, result.stdout)
        self.assertIn("Percentile", result.stdout)
        self.assertIn("+200.00%", result.stdout)

    def test_host_dependent_benchmarks_are_exempt(self) -> None:
        """Their allocation count follows the machine's process list, not us."""
        result = self.run_gate(HOST_DEPENDENT)
        self.assertEqual(result.returncode, 0, result.stdout)

    def test_bytes_alone_do_not_gate(self) -> None:
        """B/op moved +100% here; only allocs/op decides, and it is unchanged."""
        bytes_only = REGRESSION.rsplit("allocs/op", 2)[0]
        result = self.run_gate(bytes_only)
        self.assertEqual(result.returncode, 0, result.stdout)


if __name__ == "__main__":
    unittest.main()
