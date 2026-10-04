package forecast

import (
	"math"
	"testing"
)

// TestStudentT975 compares the expansion with published two-sided 95% t
// values across the degrees of freedom Fit can produce.
func TestStudentT975(t *testing.T) {
	t.Parallel()
	for df, want := range map[int]float64{
		5: 2.5706, 6: 2.4469, 8: 2.3060, 10: 2.2281, 15: 2.1314,
		20: 2.0860, 30: 2.0423, 60: 2.0003, 88: 1.9873, 120: 1.9799,
	} {
		if got := studentT975(df); math.Abs(got-want) > 0.003 {
			t.Errorf("studentT975(%d) = %.4f, want %.4f", df, got, want)
		}
	}
}
