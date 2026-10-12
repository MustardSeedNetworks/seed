package export_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/export"
)

// fakeSources returns a fixed device summary and a fixed set of cards.
type fakeSources struct {
	refreshErr error
	cards      export.Cards
}

func (f fakeSources) RefreshInterfaces() error { return f.refreshErr }
func (f fakeSources) DeviceMAC(string) string  { return "aa:bb:cc:dd:ee:ff" }
func (f fakeSources) IPMode() string           { return "dhcp" }
func (f fakeSources) Cards(context.Context, string) export.Cards {
	return f.cards
}

func TestBuildComposesDocument(t *testing.T) {
	src := fakeSources{cards: export.Cards{
		Link: &export.LinkCard{LinkUp: true},
		DNS:  &export.DNSCard{Server: "8.8.8.8"},
	}}
	svc := export.NewService(src)

	data, err := svc.Build(context.Background(), "eth0", "1.2.3")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if data.Version != "1.2.3" || data.Device.Interface != "eth0" ||
		data.Device.MAC != "aa:bb:cc:dd:ee:ff" || data.Device.IPMode != "dhcp" {
		t.Errorf("device/meta wrong: %+v", data)
	}
	if data.Timestamp == "" {
		t.Error("timestamp should be stamped")
	}
	if data.Cards.Link == nil || data.Cards.DNS == nil || data.Cards.Gateway != nil {
		t.Errorf("cards not passed through: %+v", data.Cards)
	}
}

func TestBuildAbortsOnRefreshError(t *testing.T) {
	wantErr := errors.New("refresh failed")
	svc := export.NewService(fakeSources{refreshErr: wantErr})
	if _, err := svc.Build(context.Background(), "eth0", "1"); !errors.Is(err, wantErr) {
		t.Errorf("Build should abort on refresh error, got %v", err)
	}
}

// TestCardsWire pins the export document's card keys, which predate the typed
// cards: an absent card is left out, a present one keeps its field names.
func TestCardsWire(t *testing.T) {
	length := 12.5
	native := 10
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	cards := export.Cards{
		Link: &export.LinkCard{LinkUp: true, Speed: "1000Mb/s", Duplex: "full", AutoNeg: true},
		IPConfig: &export.IPConfigCard{
			Addresses: []string{"192.0.2.10/24"},
			Lease: &export.LeaseCard{
				DHCPServer: "192.0.2.1", Gateway: "192.0.2.1", LeaseTime: 3600, DNS: []string{"192.0.2.53"},
			},
		},
		Switch: &export.SwitchCard{Running: true, Neighbors: []export.NeighborRecord{{
			Protocol: "lldp", SystemName: "sw1", PortID: "Gi1/0/1",
			PortDescription: "uplink", ManagementAddress: "192.0.2.2",
		}}},
		DNS: &export.DNSCard{
			Server: "192.0.2.53", TestHostname: "example.com",
			Forward: &export.DNSLookup{Result: []string{"192.0.2.80"}, Time: 4, Status: "good", Error: ""},
		},
		Gateway: &export.GatewayCard{
			Gateway: "192.0.2.1", Reachable: true, Sent: 4, Received: 4, AvgTime: 1.5, Status: "good",
		},
		VLAN: &export.VLANCard{
			NativeVlan: &native, TaggedVlans: []int{20},
			Configured: export.VLANConfigured{Enabled: true, ID: 20},
		},
		WiFi: &export.WiFiCard{
			SSID: "lab", BSSID: "aa:bb:cc:dd:ee:01", Signal: -55, Channel: 36, Frequency: 5180, Security: "WPA2",
		},
		Cable: &export.CableCard{Supported: true, Length: &length, Status: "ok", Faults: []string{}},
		Speedtest: &export.SpeedtestCard{
			Download: 900, Upload: 400, Latency: 8, Server: "s", Location: "l", Host: "h",
			Distance: 3, Timestamp: at, TestDuration: 20,
		},
		Iperf: &export.IperfCard{
			Bandwidth: 940, Transfer: 1120, Protocol: "tcp", Direction: "upload",
			Duration: 10, Server: "192.0.2.9", Port: 5201, Timestamp: at,
		},
	}

	got, err := json.Marshal(cards)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"link":{"linkUp":true,"speed":"1000Mb/s","duplex":"full","autoNeg":true},` +
		`"ipConfig":{"addresses":["192.0.2.10/24"],"lease":{"dhcpServer":"192.0.2.1","gateway":"192.0.2.1",` +
		`"leaseTime":3600,"dns":["192.0.2.53"]}},` +
		`"switch":{"running":true,"neighbors":[{"protocol":"lldp","systemName":"sw1","portId":"Gi1/0/1",` +
		`"portDescription":"uplink","managementAddress":"192.0.2.2"}]},` +
		`"dns":{"server":"192.0.2.53","testHostname":"example.com",` +
		`"forward":{"result":["192.0.2.80"],"time":4,"status":"good","error":""}},` +
		`"gateway":{"gateway":"192.0.2.1","reachable":true,"sent":4,"received":4,"lossPercent":0,` +
		`"avgTime":1.5,"status":"good"},` +
		`"vlan":{"nativeVlan":10,"taggedVlans":[20],"configured":{"enabled":true,"id":20}},` +
		`"wifi":{"ssid":"lab","bssid":"aa:bb:cc:dd:ee:01","signal":-55,"channel":36,"frequency":5180,` +
		`"security":"WPA2"},` +
		`"cable":{"supported":true,"length":12.5,"status":"ok","faults":[]},` +
		`"speedtest":{"download":900,"upload":400,"latency":8,"server":"s","location":"l","host":"h",` +
		`"distance":3,"timestamp":"2026-10-12T09:00:00Z","testDuration":20},` +
		`"iperf":{"bandwidth":940,"transfer":1120,"retransmits":0,"jitter":0,"lostPackets":0,` +
		`"lostPercent":0,"protocol":"tcp","direction":"upload","duration":10,"server":"192.0.2.9",` +
		`"port":5201,"timestamp":"2026-10-12T09:00:00Z"}}`
	if string(got) != want {
		t.Errorf("cards wire\n got: %s\nwant: %s", got, want)
	}

	empty, err := json.Marshal(export.Cards{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(empty) != "{}" {
		t.Errorf("absent cards should be left out, got %s", empty)
	}
}
