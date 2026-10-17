package delivery

import (
	"math"
	"testing"
	"time"
)

func TestTimeTicksCountsHundredthsAndWrapsLikeSysUpTime(t *testing.T) {
	for name, tc := range map[string]struct {
		uptime time.Duration
		want   uint32
	}{
		"start":                 {0, 0},
		"one second":            {time.Second, 100},
		"part of a tick":        {15 * time.Millisecond, 1},
		"last tick before wrap": {math.MaxUint32 * timeTick, math.MaxUint32},
		"wraps to zero":         {(math.MaxUint32 + 1) * timeTick, 0},
		"wraps past zero":       {(math.MaxUint32 + 6) * timeTick, 5},
		"clock behind start":    {-time.Second, 0},
	} {
		t.Run(name, func(t *testing.T) {
			if got := timeTicks(tc.uptime); got != tc.want {
				t.Errorf("timeTicks(%v) = %d, want %d", tc.uptime, got, tc.want)
			}
		})
	}
}
