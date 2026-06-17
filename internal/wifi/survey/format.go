package survey

func getPriorityLabel(p RecommendationPriority) string {
	switch p {
	case PriorityHigh:
		return "HIGH"
	case PriorityMedium:
		return "MED"
	case PriorityLow:
		return "LOW"
	}
	return "LOW" // Default fallback for unknown priority
}

func getStatusColor(status Status) []int {
	switch status {
	case StatusCompleted:
		return []int{40, 167, 69} // Green
	case StatusInProgress:
		return []int{0, 123, 255} // Blue
	case StatusPaused:
		return []int{255, 193, 7} // Yellow
	case StatusCreated:
		return []int{108, 117, 125} // Gray
	}
	return []int{108, 117, 125} // Default fallback
}

func getCoverageGrade(score float64) string {
	switch {
	case score >= coverageScoreGood:
		return "A"
	case score >= coverageGradeB:
		return "B"
	case score >= coverageGradeC:
		return "C"
	case score >= coverageGradeD:
		return "D"
	default:
		return "F"
	}
}

func getGradeColor(grade string) []int {
	switch grade {
	case "A":
		return []int{40, 167, 69}
	case "B":
		return []int{144, 238, 144}
	case "C":
		return []int{255, 193, 7}
	case "D":
		return []int{255, 128, 0}
	default:
		return []int{220, 53, 69}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// getPassiveSampleFromPoint extracts PassiveSample data from a SamplePoint.
// Returns nil if the SampleData is not a PassiveSample.
func getPassiveSampleFromPoint(sp *SamplePoint) *PassiveSample {
	if sp == nil || sp.SampleData == nil {
		return nil
	}
	switch data := sp.SampleData.(type) {
	case *PassiveSample:
		return data
	case PassiveSample:
		return &data
	default:
		return nil
	}
}
