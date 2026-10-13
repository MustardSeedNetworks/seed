package wifianomaly_test

import (
	"testing"

	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
)

func TestRogueAPOnLAN(t *testing.T) {
	const ap = "aa:bb:cc:00:00:01"
	cases := []struct {
		name  string
		bssid string
		wired []string
		want  bool
	}{
		{name: "no wired data", bssid: ap, want: false},
		{name: "BSSID on the wired side", bssid: ap, wired: []string{ap}, want: true},
		{name: "case differs", bssid: ap, wired: []string{"AA:BB:CC:00:00:01"}, want: true},
		{name: "dash separators", bssid: ap, wired: []string{"aa-bb-cc-00-00-01"}, want: true},
		{name: "BSSID not on the wired side", bssid: "de:ad:be:ef:00:99", wired: []string{ap}, want: false},
		{name: "unparseable wired entries", bssid: ap, wired: []string{"", "ip:10.0.0.1"}, want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := bss(c.bssid, "corp", "WPA3", "5 GHz", 36)
			b.PMFRequired = true
			got := hasDef(wifianomaly.NewDetector().Detect(tree("corp", b), c.wired), wifianomaly.DefRogueAPOnLAN)
			if got != c.want {
				t.Errorf("rogue-ap-on-lan fired = %v, want %v", got, c.want)
			}
		})
	}
}

func TestWiredDataRulesListsTheRogueAPRule(t *testing.T) {
	rules := wifianomaly.WiredDataRules()
	if len(rules) != 1 || rules[0].ID != wifianomaly.DefRogueAPOnLAN || rules[0].Title == "" {
		t.Errorf("WiredDataRules() = %v, want only %s with its title", rules, wifianomaly.DefRogueAPOnLAN)
	}
}
