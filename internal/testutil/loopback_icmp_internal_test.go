package testutil

import "testing"

func TestTotalLossMatchesOnlyAPingThatGotNoReply(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want bool
	}{
		{
			name: "macOS with stealth mode on (seed#2905)",
			out:  "1 packets transmitted, 0 packets received, 100.0% packet loss",
			want: true,
		},
		{name: "Linux, no reply", out: "1 packets transmitted, 0 received, 100% packet loss, time 0ms", want: true},
		{name: "Windows, no reply", out: "Packets: Sent = 1, Received = 0, Lost = 1 (100% loss),", want: true},
		{name: "Linux, reply", out: "1 packets transmitted, 1 received, 0% packet loss, time 0ms", want: false},
		{name: "macOS, reply", out: "1 packets transmitted, 1 packets received, 0.0% packet loss", want: false},
		{name: "ping could not open a socket", out: "ping: socket: Operation not permitted", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := totalLoss.MatchString(tt.out); got != tt.want {
				t.Errorf("totalLoss.MatchString(%q) = %v, want %v", tt.out, got, tt.want)
			}
		})
	}
}
