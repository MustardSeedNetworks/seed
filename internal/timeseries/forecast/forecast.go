// Package forecast fits a straight line to a time series and estimates when
// the line reaches a threshold. The fit is ordinary least squares and the
// estimate carries the line's 95% confidence band, so a series whose rise is
// lost in its own scatter gets no estimate at all. Nothing is
// learned or tuned: the same points always give the same answer, and every
// number in it can be checked by hand.
//
// A forecast reaches no further ahead than its points reach back. Thirty days
// of history support a thirty-day forecast; extrapolating a straight line past
// that is guessing.
package forecast

import (
	"errors"
	"math"
	"time"
)

// MinPoints is the fewest points Fit accepts. Fewer leave the slope's
// interval too wide to bound a crossing.
const MinPoints = 7

// ErrTooFewPoints means Fit was given fewer than MinPoints points, or points
// that do not span any time.
var ErrTooFewPoints = errors.New("too few points to fit a trend")

const (
	day = 24 * time.Hour
	// lineParams is the intercept and slope a fitted line spends of the
	// points' degrees of freedom.
	lineParams = 2
	// bisections takes a 90-day search span below a nanosecond.
	bisections = 60
)

// Point is one observation.
type Point struct {
	At    time.Time
	Value float64
}

// Trend is the least-squares line through a series.
type Trend struct {
	From   time.Time // first point
	To     time.Time // last point
	Points int
	// Level is the line's value at To.
	Level float64
	// PerDay is the slope in value units per day, and PerDayMargin the
	// half-width of its 95% interval.
	PerDay       float64
	PerDayMargin float64

	// The band's terms: days from From to the points' mean time, the
	// spread of the points' times, the residual standard error times the
	// t multiplier, and the point count as a float.
	meanX, sxx, margin, n float64
}

// Fit returns the least-squares line through points, which need not be
// sorted.
func Fit(points []Point) (Trend, error) {
	n := len(points)
	if n < MinPoints {
		return Trend{}, ErrTooFewPoints
	}
	from, to := points[0].At, points[0].At
	for _, p := range points[1:] {
		if p.At.Before(from) {
			from = p.At
		}
		if p.At.After(to) {
			to = p.At
		}
	}

	// x is days since the first point, which keeps the sums well scaled.
	x := func(p Point) float64 { return p.At.Sub(from).Hours() / day.Hours() }
	var meanX, meanY float64
	for _, p := range points {
		meanX += x(p)
		meanY += p.Value
	}
	meanX /= float64(n)
	meanY /= float64(n)

	var sxx, sxy float64
	for _, p := range points {
		dx := x(p) - meanX
		sxx += dx * dx
		sxy += dx * (p.Value - meanY)
	}
	if sxx == 0 {
		return Trend{}, ErrTooFewPoints
	}
	slope := sxy / sxx
	intercept := meanY - slope*meanX

	var ssr float64
	for _, p := range points {
		r := p.Value - (intercept + slope*x(p))
		ssr += r * r
	}
	df := n - lineParams
	margin := studentT975(df) * math.Sqrt(ssr/float64(df))

	span := to.Sub(from).Hours() / day.Hours()
	return Trend{
		From:         from,
		To:           to,
		Points:       n,
		Level:        intercept + slope*span,
		PerDay:       slope,
		PerDayMargin: margin / math.Sqrt(sxx),
		meanX:        meanX,
		sxx:          sxx,
		margin:       margin,
		n:            float64(n),
	}, nil
}

// band returns the line's value days after To and the half-width of its 95%
// confidence band there.
func (t *Trend) band(days float64) (float64, float64) {
	x := t.To.Sub(t.From).Hours()/day.Hours() + days
	dx := x - t.meanX
	return t.Level + t.PerDay*days, t.margin * math.Sqrt(1/t.n+dx*dx/t.sxx)
}

// Horizon is the furthest time a crossing may be forecast: as far past To as
// the points reach back from it.
func (t *Trend) Horizon() time.Time {
	return t.To.Add(t.To.Sub(t.From))
}

// Outlook is how a trend stands against a threshold.
type Outlook string

// The outlooks, from most to least pressing.
const (
	// Above: the line is already at or over the threshold at To.
	Above Outlook = "above"
	// Rising: the line crosses the threshold by the horizon, and the whole
	// slope interval is a rise.
	Rising Outlook = "rising"
	// Beyond: the slope interval is a rise, but the line crosses after the
	// horizon.
	Beyond Outlook = "beyond"
	// Steady: the slope interval includes no rise, so no crossing is
	// forecast.
	Steady Outlook = "steady"
)

// Crossing is when a trend reaches a threshold. Estimate and Earliest are set
// for Rising only. Latest is set for Rising when the band's lower edge also
// crosses by the horizon.
type Crossing struct {
	Outlook  Outlook    `json:"outlook"`
	Estimate *time.Time `json:"estimate,omitempty"`
	Earliest *time.Time `json:"earliest,omitempty"`
	Latest   *time.Time `json:"latest,omitempty"`
}

// Crossing returns when the trend reaches threshold. Estimate is where the
// line crosses, and the range runs from where the upper edge of its 95%
// confidence band crosses to where the lower edge does. The band bounds the
// line, not single readings, which scatter around it.
func (t *Trend) Crossing(threshold float64) Crossing {
	if t.Level >= threshold {
		return Crossing{Outlook: Above}
	}
	if t.PerDay-t.PerDayMargin <= 0 {
		return Crossing{Outlook: Steady}
	}

	horizon := t.To.Sub(t.From).Hours() / day.Hours()
	estimate := (threshold - t.Level) / t.PerDay
	if estimate > horizon {
		return Crossing{Outlook: Beyond}
	}
	at := func(days float64) *time.Time {
		v := t.To.Add(time.Duration(days * float64(day)))
		return &v
	}
	upper := func(days float64) float64 { v, w := t.band(days); return v + w }
	lower := func(days float64) float64 { v, w := t.band(days); return v - w }

	c := Crossing{Outlook: Rising, Estimate: at(estimate), Earliest: at(0)}
	if upper(0) < threshold {
		c.Earliest = at(firstReach(upper, threshold, 0, estimate))
	}
	if lower(horizon) >= threshold {
		c.Latest = at(firstReach(lower, threshold, estimate, horizon))
	}
	return c
}

// firstReach returns the days in [lo, hi] at which the increasing f reaches
// threshold, given f(lo) < threshold <= f(hi). Past To both band edges rise
// whenever the slope's own interval is a rise, which Crossing has checked.
func firstReach(f func(float64) float64, threshold, lo, hi float64) float64 {
	const half = 0.5
	for range bisections {
		mid := lo + (hi-lo)*half
		if f(mid) < threshold {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi
}

// studentT975 is the 97.5th percentile of Student's t with df degrees of
// freedom, the multiplier for a two-sided 95% interval. It is the
// Cornish-Fisher expansion of Abramowitz and Stegun 26.7.5 around the normal
// percentile z, within 0.003 of the exact value from df = 5 (MinPoints - 2)
// upward.
func studentT975(df int) float64 {
	const (
		z  = 1.959963984540054  // standard normal 97.5th percentile
		g1 = 2.372271230298562  // (z³ + z) / 4
		g2 = 2.8224986157396112 // (5z⁵ + 16z³ + 3z) / 96
		g3 = 2.555849679507722  // (3z⁷ + 19z⁵ + 17z³ − 15z) / 384
	)
	n := float64(df)
	return z + g1/n + g2/(n*n) + g3/(n*n*n)
}
