package anomaly

import "testing"

// TestSeverityRankOrdering pins the ladder ordering info < warning < error <
// critical (ADR-0021 ph5) and that an unknown severity ranks below all of them.
func TestSeverityRankOrdering(t *testing.T) {
	ordered := []Severity{SeverityInfo, SeverityWarning, SeverityError, SeverityCritical}
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1].rank() >= ordered[i].rank() {
			t.Errorf("rank(%q)=%d should be < rank(%q)=%d",
				ordered[i-1], ordered[i-1].rank(), ordered[i], ordered[i].rank())
		}
	}
	if (Severity("nope")).rank() != rankNone {
		t.Errorf("unknown severity rank = %d, want rankNone(%d)", Severity("nope").rank(), rankNone)
	}
}

// TestSeverityValid confirms every ladder level is valid and an unknown one is not.
func TestSeverityValid(t *testing.T) {
	for _, s := range []Severity{SeverityInfo, SeverityWarning, SeverityError, SeverityCritical} {
		if !s.valid() {
			t.Errorf("%q should be valid", s)
		}
	}
	if (Severity("nope")).valid() {
		t.Error(`"nope" should be invalid`)
	}
}
