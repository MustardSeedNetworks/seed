package survey

import (
	"fmt"
	"sort"
)

// SurveyStats holds calculated statistics for a survey.
//
//nolint:revive // Renaming to Stats would be a breaking API change
type SurveyStats struct {
	TotalSamples     int
	AvgRSSI          int
	MinRSSI          int
	MaxRSSI          int
	CoverageScore    float64
	WeakAreas        int
	DeadZones        int
	ExcellentPercent float64
	GoodPercent      float64
	FairPercent      float64
	PoorPercent      float64
	DeadPercent      float64
}

func calculateSurveyStats(samples []*SamplePoint) SurveyStats {
	stats := SurveyStats{
		TotalSamples: len(samples),
		MinRSSI:      0,
		MaxRSSI:      minRSSIValue,
	}

	if len(samples) == 0 {
		return stats
	}

	var sumRSSI int
	var excellent, good, fair, poor, dead int

	for _, sample := range samples {
		ps := getPassiveSampleFromPoint(sample)
		if ps == nil || len(ps.Networks) == 0 {
			continue
		}

		// Use best RSSI from networks in the passive sample
		bestRSSI := minRSSIValue
		for _, net := range ps.Networks {
			if net.Signal > bestRSSI {
				bestRSSI = net.Signal
			}
		}

		sumRSSI += bestRSSI
		if bestRSSI > stats.MaxRSSI {
			stats.MaxRSSI = bestRSSI
		}
		if bestRSSI < stats.MinRSSI || stats.MinRSSI == 0 {
			stats.MinRSSI = bestRSSI
		}

		switch {
		case bestRSSI > ExcellentSignal:
			excellent++
		case bestRSSI > GoodSignal:
			good++
		case bestRSSI > FairSignal:
			fair++
		case bestRSSI > PoorSignal:
			poor++
		default:
			dead++
		}
	}

	if len(samples) > 0 {
		stats.AvgRSSI = sumRSSI / len(samples)
		total := float64(len(samples))
		stats.ExcellentPercent = float64(excellent) / total * percentMultiplier
		stats.GoodPercent = float64(good) / total * percentMultiplier
		stats.FairPercent = float64(fair) / total * percentMultiplier
		stats.PoorPercent = float64(poor) / total * percentMultiplier
		stats.DeadPercent = float64(dead) / total * percentMultiplier
		stats.CoverageScore = stats.ExcellentPercent + stats.GoodPercent + (stats.FairPercent * fairCoverageWeight)
		stats.WeakAreas = poor
		stats.DeadZones = dead
	}

	return stats
}

func calculateFloorStats(samples []*SamplePoint) SurveyStats {
	return calculateSurveyStats(samples)
}

// ChannelInfo represents WiFi channel usage information.
type ChannelInfo struct {
	Channel int
	Count   int
}

func getChannelUsage(samples []*SamplePoint) []ChannelInfo {
	channelMap := make(map[int]int)

	for _, sample := range samples {
		ps := getPassiveSampleFromPoint(sample)
		if ps == nil {
			continue
		}
		for _, net := range ps.Networks {
			if net.Channel > 0 {
				channelMap[net.Channel]++
			}
		}
	}

	channels := make([]ChannelInfo, 0, len(channelMap))
	for ch, count := range channelMap {
		channels = append(channels, ChannelInfo{Channel: ch, Count: count})
	}

	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Count > channels[j].Count
	})

	// Limit to top channels
	if len(channels) > topChannelsLimit {
		channels = channels[:topChannelsLimit]
	}

	return channels
}

// RecommendationPriority indicates the urgency of a recommendation.
type RecommendationPriority int

// Recommendation priority levels.
const (
	PriorityLow RecommendationPriority = iota
	PriorityMedium
	PriorityHigh
)

// Recommendation represents an improvement suggestion.
type Recommendation struct {
	Text     string
	Priority RecommendationPriority
}

func generateSurveyRecommendations(stats *SurveyStats) []Recommendation {
	var recommendations []Recommendation

	// Coverage-based recommendations
	switch {
	case stats.CoverageScore < coverageScoreCritical:
		recommendations = append(recommendations, Recommendation{
			Text:     "Critical coverage issues detected. Consider a complete WiFi infrastructure redesign with additional access points.",
			Priority: PriorityHigh,
		})
	case stats.CoverageScore < coverageScorePoor:
		recommendations = append(recommendations, Recommendation{
			Text:     "Poor overall coverage. Add 2-3 additional access points in strategic locations to improve connectivity.",
			Priority: PriorityHigh,
		})
	case stats.CoverageScore < coverageScoreModerate:
		recommendations = append(recommendations, Recommendation{
			Text:     "Moderate coverage detected. Consider adding 1-2 access points to strengthen weak areas.",
			Priority: PriorityMedium,
		})
	}

	// Dead zone recommendations
	if stats.DeadZones > 0 {
		recommendations = append(recommendations, Recommendation{
			Text: fmt.Sprintf(
				"Found %d dead zone(s) with signal below -85 dBm. Prioritize these areas for immediate AP placement.",
				stats.DeadZones,
			),
			Priority: PriorityHigh,
		})
	}

	// Weak area recommendations
	if stats.WeakAreas > 0 {
		recommendations = append(recommendations, Recommendation{
			Text: fmt.Sprintf(
				"Found %d weak area(s) with poor signal. Consider power adjustments or additional coverage.",
				stats.WeakAreas,
			),
			Priority: PriorityMedium,
		})
	}

	// Sample density recommendations
	if stats.TotalSamples < minSampleThreshold {
		recommendations = append(recommendations, Recommendation{
			Text:     "Limited sample data. Collect more samples for accurate analysis, especially in edge areas and around obstacles.",
			Priority: PriorityLow,
		})
	}

	// No issues found
	if len(recommendations) == 0 && stats.CoverageScore >= coverageScoreGood {
		recommendations = append(recommendations, Recommendation{
			Text:     "WiFi coverage meets quality standards. Continue monitoring for any future degradation.",
			Priority: PriorityLow,
		})
	}

	return recommendations
}
