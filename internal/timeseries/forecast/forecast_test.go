package forecast_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/forecast"
)

func t0() time.Time { return time.Date(2026, 9, 1, 14, 0, 0, 0, time.UTC) }

// daily returns one point a day from t0, the d-th valued f(d).
func daily(days int, f func(d int) float64) []forecast.Point {
	points := make([]forecast.Point, days)
	for d := range days {
		points[d] = forecast.Point{At: t0().AddDate(0, 0, d), Value: f(d)}
	}
	return points
}

// wobble is a fixed, zero-mean scatter of amplitude a.
func wobble(d int, a float64) float64 {
	return a * []float64{1, -1, 0.5, -0.5, 0}[d%5]
}

func fit(t *testing.T, points []forecast.Point) forecast.Trend {
	t.Helper()
	trend, err := forecast.Fit(points)
	if err != nil {
		t.Fatalf("Fit: %v", err)
	}
	return trend
}

func TestFitExactLine(t *testing.T) {
	t.Parallel()
	// Reversed, to show the order of the points does not matter.
	points := daily(30, func(d int) float64 { return 20 + 2*float64(d) })
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	trend := fit(t, points)

	end := t0().AddDate(0, 0, 29)
	if !trend.From.Equal(t0()) || !trend.To.Equal(end) || trend.Points != 30 {
		t.Errorf("span = %v..%v over %d points", trend.From, trend.To, trend.Points)
	}
	if math.Abs(trend.PerDay-2) > 1e-9 || math.Abs(trend.Level-78) > 1e-9 || trend.PerDayMargin > 1e-9 {
		t.Errorf("trend = %+v, want 2/day to 78 with no margin", trend)
	}
	if want := end.AddDate(0, 0, 29); !trend.Horizon().Equal(want) {
		t.Errorf("Horizon = %v, want %v", trend.Horizon(), want)
	}

	c := trend.Crossing(80)
	want := end.Add(24 * time.Hour)
	if c.Outlook != forecast.Rising || c.Estimate == nil || c.Earliest == nil || c.Latest == nil {
		t.Fatalf("Crossing(80) = %+v, want rising with a full range", c)
	}
	for name, got := range map[string]time.Time{"estimate": *c.Estimate, "earliest": *c.Earliest, "latest": *c.Latest} {
		if got.Sub(want).Abs() > time.Second {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
}

func TestCrossingOutlooks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		points    []forecast.Point
		threshold float64
		want      forecast.Outlook
	}{
		{
			name:      "already over the threshold",
			points:    daily(14, func(d int) float64 { return 70 + float64(d) }),
			threshold: 80,
			want:      forecast.Above,
		},
		{
			name:      "flat with scatter",
			points:    daily(30, func(d int) float64 { return 50 + wobble(d, 5) }),
			threshold: 80,
			want:      forecast.Steady,
		},
		{
			name:      "falling",
			points:    daily(30, func(d int) float64 { return 60 - float64(d)/2 }),
			threshold: 80,
			want:      forecast.Steady,
		},
		{
			// 0.5/day from 20 reaches 80 after 120 days; ten days of
			// history forecast ten days ahead.
			name:      "rising past the horizon",
			points:    daily(10, func(d int) float64 { return 20 + float64(d)/2 }),
			threshold: 80,
			want:      forecast.Beyond,
		},
		{
			// A rise smaller than the scatter is not a rise.
			name:      "rise lost in the scatter",
			points:    daily(10, func(d int) float64 { return 70 + float64(d)/10 + wobble(d, 8) }),
			threshold: 80,
			want:      forecast.Steady,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			trend := fit(t, tt.points)
			c := trend.Crossing(tt.threshold)
			if c.Outlook != tt.want {
				t.Fatalf("Outlook = %s, want %s (trend %+v)", c.Outlook, tt.want, trend)
			}
			if c.Estimate != nil || c.Earliest != nil || c.Latest != nil {
				t.Errorf("a %s crossing carries times: %+v", c.Outlook, c)
			}
		})
	}
}

// TestCrossingRangeBracketsTheTruth: with scatter around a known line the
// estimate lands near the true crossing, the range contains it, and the range
// has no later end when the band's lower edge stays under the threshold to
// the horizon.
func TestCrossingRangeBracketsTheTruth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		days       int
		scatter    float64
		wantLatest bool
	}{
		{name: "tight", days: 30, scatter: 2, wantLatest: true},
		{name: "loose", days: 10, scatter: 3, wantLatest: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// The true line rises 1/day and reaches 80 one day after the
			// last point.
			base := 80 - float64(tt.days)
			trend := fit(t, daily(tt.days, func(d int) float64 { return base + float64(d) + wobble(d, tt.scatter) }))
			truth := t0().AddDate(0, 0, tt.days)

			c := trend.Crossing(80)
			if c.Outlook != forecast.Rising {
				t.Fatalf("Outlook = %s, want rising (trend %+v)", c.Outlook, trend)
			}
			if c.Estimate.Sub(truth).Abs() > 2*24*time.Hour {
				t.Errorf("estimate %v is more than two days from %v", *c.Estimate, truth)
			}
			if c.Earliest.After(truth) || c.Earliest.After(*c.Estimate) {
				t.Errorf("earliest %v is after the truth %v or the estimate %v", *c.Earliest, truth, *c.Estimate)
			}
			if (c.Latest != nil) != tt.wantLatest {
				t.Fatalf("latest = %v, want set %v (trend %+v)", c.Latest, tt.wantLatest, trend)
			}
			if c.Latest != nil && (c.Latest.Before(truth) || c.Latest.After(trend.Horizon())) {
				t.Errorf("latest %v is before the truth %v or past the horizon %v", *c.Latest, truth, trend.Horizon())
			}
		})
	}
}

func TestFitRejectsTooFewPoints(t *testing.T) {
	t.Parallel()
	sameInstant := make([]forecast.Point, forecast.MinPoints)
	for i := range sameInstant {
		sameInstant[i] = forecast.Point{At: t0(), Value: float64(i)}
	}
	for name, points := range map[string][]forecast.Point{
		"none":         nil,
		"one short":    daily(forecast.MinPoints-1, func(d int) float64 { return float64(d) }),
		"same instant": sameInstant,
	} {
		if _, err := forecast.Fit(points); !errors.Is(err, forecast.ErrTooFewPoints) {
			t.Errorf("%s: err = %v, want ErrTooFewPoints", name, err)
		}
	}
}
