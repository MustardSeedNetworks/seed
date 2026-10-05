package voip

import "testing"

// TestRFactorTracksFullG107 holds the reduced model to the full G.107
// equations: the want values come from an independent Python
// transcription of G.107 (06/2015) with every other parameter at its
// default, and the reduction must stay within 0.25 R of them up to 300 ms.
func TestRFactorTracksFullG107(t *testing.T) {
	t.Parallel()
	g711, _ := codecFor(ptPCMU)
	g729, _ := codecFor(ptG729)
	cases := []struct {
		name    string
		codec   Codec
		delayMs float64
		lossPct float64
		want    float64
	}{
		{"defaults", g711, 0, 0, 93.206},
		{"50 ms", g711, 50, 0, 91.756},
		{"150 ms", g711, 150, 0, 89.539},
		{"300 ms", g711, 300, 0, 72.666},
		{"G.711 1 percent loss", g711, 0, 1, 89.566},
		{"G.711 2 percent loss 40 ms", g711, 40, 2, 84.987},
		{"G.729 2 percent loss 40 ms", g729, 40, 2, 72.998},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := RFactor(tc.codec, tc.delayMs, tc.lossPct); got < tc.want-0.25 || got > tc.want+0.25 {
				t.Fatalf("RFactor(%s, %v ms, %v %%) = %.3f, want %.3f ± 0.25",
					tc.codec.Name, tc.delayMs, tc.lossPct, got, tc.want)
			}
		})
	}
}

func TestMOSMatchesG107AnnexB(t *testing.T) {
	t.Parallel()
	// The G.109 category boundaries, to two places.
	cases := []struct{ r, want float64 }{
		{-5, 1}, {0, 1}, {50, 2.58}, {60, 3.10}, {70, 3.60}, {80, 4.02}, {90, 4.34}, {100, 4.5}, {120, 4.5},
	}
	for _, tc := range cases {
		if got := MOS(tc.r); got < tc.want-0.01 || got > tc.want+0.01 {
			t.Errorf("MOS(%v) = %.3f, want %.2f", tc.r, got, tc.want)
		}
	}
}

func TestCodecForScoresOnlyStaticNarrowband(t *testing.T) {
	t.Parallel()
	for pt, want := range map[uint8]string{0: "PCMU", 8: "PCMA", 18: "G729"} {
		if c, ok := codecFor(pt); !ok || c.Name != want {
			t.Errorf("codecFor(%d) = %q, %v; want %q", pt, c.Name, ok, want)
		}
	}
	for _, pt := range []uint8{3, 9, 13, 96, 111, 127} {
		if _, ok := codecFor(pt); ok {
			t.Errorf("codecFor(%d) scored; want unscored", pt)
		}
	}
}
