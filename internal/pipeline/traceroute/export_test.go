package traceroute

// This file is only compiled during testing (due to _test.go suffix)
// and provides access to internal implementation details.

// ExportAnalyzeHops exposes analyzeHops for testing.
func ExportAnalyzeHops(hops []Hop, bottlenecks *[]Bottleneck) (float64, int) {
	return analyzeHops(hops, bottlenecks)
}

// ExportDetectBottleneck exposes detectBottleneck for testing.
func ExportDetectBottleneck(hopIndex int, hop Hop, previousRTT, currentRTT float64) *Bottleneck {
	return detectBottleneck(hopIndex, hop, previousRTT, currentRTT)
}

// ExportCalculateScore exposes calculateScore for testing.
func ExportCalculateScore(analysis *PathAnalysis) int {
	return calculateScore(analysis)
}
