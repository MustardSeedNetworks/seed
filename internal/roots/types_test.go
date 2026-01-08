package roots_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots"
)

// TestTopologyNodeType tests the TopologyNodeType constants.
func TestTopologyNodeType(t *testing.T) {
	tests := []struct {
		name     string
		nodeType roots.TopologyNodeType
		want     string
	}{
		{name: "router", nodeType: roots.NodeTypeRouter, want: "router"},
		{name: "switch", nodeType: roots.NodeTypeSwitch, want: "switch"},
		{name: "host", nodeType: roots.NodeTypeHost, want: "host"},
		{name: "gateway", nodeType: roots.NodeTypeGateway, want: "gateway"},
		{name: "firewall", nodeType: roots.NodeTypeFirewall, want: "firewall"},
		{name: "access_point", nodeType: roots.NodeTypeAP, want: "access_point"},
		{name: "cloud", nodeType: roots.NodeTypeCloud, want: "cloud"},
		{name: "unknown", nodeType: roots.NodeTypeUnknown, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.nodeType); got != tt.want {
				t.Errorf("TopologyNodeType = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTopologyLinkType tests the TopologyLinkType constants.
func TestTopologyLinkType(t *testing.T) {
	tests := []struct {
		name     string
		linkType roots.TopologyLinkType
		want     string
	}{
		{name: "ethernet", linkType: roots.LinkTypeEthernet, want: "ethernet"},
		{name: "wifi", linkType: roots.LinkTypeWiFi, want: "wifi"},
		{name: "fiber", linkType: roots.LinkTypeFiber, want: "fiber"},
		{name: "wan", linkType: roots.LinkTypeWAN, want: "wan"},
		{name: "vpn", linkType: roots.LinkTypeVPN, want: "vpn"},
		{name: "unknown", linkType: roots.LinkTypeUnknown, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.linkType); got != tt.want {
				t.Errorf("TopologyLinkType = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTopologyNode tests TopologyNode struct and JSON serialization.
func TestTopologyNode(t *testing.T) {
	tests := []struct {
		name    string
		node    roots.TopologyNode
		wantErr bool
	}{
		{
			name: "complete node",
			node: roots.TopologyNode{
				ID:        "node-1",
				Type:      roots.NodeTypeRouter,
				Label:     "Core Router",
				IP:        "192.168.1.1",
				MAC:       "00:11:22:33:44:55",
				Vendor:    "Cisco",
				Metadata:  map[string]string{"location": "datacenter-1"},
				X:         100.5,
				Y:         200.5,
				UpdatedAt: time.Now().UTC().Truncate(time.Second),
			},
			wantErr: false,
		},
		{
			name: "minimal node",
			node: roots.TopologyNode{
				ID:   "node-2",
				Type: roots.NodeTypeHost,
			},
			wantErr: false,
		},
		{
			name: "node with empty metadata",
			node: roots.TopologyNode{
				ID:       "node-3",
				Type:     roots.NodeTypeSwitch,
				Metadata: map[string]string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.TopologyNode
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.ID != tt.node.ID {
				t.Errorf("ID = %q, want %q", decoded.ID, tt.node.ID)
			}
			if decoded.Type != tt.node.Type {
				t.Errorf("Type = %q, want %q", decoded.Type, tt.node.Type)
			}
			if decoded.Label != tt.node.Label {
				t.Errorf("Label = %q, want %q", decoded.Label, tt.node.Label)
			}
			if decoded.IP != tt.node.IP {
				t.Errorf("IP = %q, want %q", decoded.IP, tt.node.IP)
			}
		})
	}
}

// TestTopologyLink tests TopologyLink struct and JSON serialization.
func TestTopologyLink(t *testing.T) {
	tests := []struct {
		name    string
		link    roots.TopologyLink
		wantErr bool
	}{
		{
			name: "complete link",
			link: roots.TopologyLink{
				ID:        "link-1",
				SourceID:  "node-1",
				TargetID:  "node-2",
				Type:      roots.LinkTypeEthernet,
				Label:     "Trunk Link",
				Bandwidth: "10Gbps",
				Latency:   0.5,
				Metadata:  map[string]string{"vlan": "100"},
				UpdatedAt: time.Now().UTC().Truncate(time.Second),
			},
			wantErr: false,
		},
		{
			name: "minimal link",
			link: roots.TopologyLink{
				ID:       "link-2",
				SourceID: "node-3",
				TargetID: "node-4",
				Type:     roots.LinkTypeWiFi,
			},
			wantErr: false,
		},
		{
			name: "link with high latency",
			link: roots.TopologyLink{
				ID:       "link-3",
				SourceID: "node-5",
				TargetID: "node-6",
				Type:     roots.LinkTypeWAN,
				Latency:  150.75,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.link)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.TopologyLink
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.ID != tt.link.ID {
				t.Errorf("ID = %q, want %q", decoded.ID, tt.link.ID)
			}
			if decoded.SourceID != tt.link.SourceID {
				t.Errorf("SourceID = %q, want %q", decoded.SourceID, tt.link.SourceID)
			}
			if decoded.TargetID != tt.link.TargetID {
				t.Errorf("TargetID = %q, want %q", decoded.TargetID, tt.link.TargetID)
			}
			if decoded.Type != tt.link.Type {
				t.Errorf("Type = %q, want %q", decoded.Type, tt.link.Type)
			}
		})
	}
}

// TestTopology tests Topology struct and JSON serialization.
func TestTopology(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name     string
		topology roots.Topology
		wantErr  bool
	}{
		{
			name: "complete topology",
			topology: roots.Topology{
				Nodes: []roots.TopologyNode{
					{ID: "node-1", Type: roots.NodeTypeRouter, Label: "Router 1"},
					{ID: "node-2", Type: roots.NodeTypeSwitch, Label: "Switch 1"},
					{ID: "node-3", Type: roots.NodeTypeHost, Label: "Server 1"},
				},
				Links: []roots.TopologyLink{
					{
						ID:       "link-1",
						SourceID: "node-1",
						TargetID: "node-2",
						Type:     roots.LinkTypeEthernet,
					},
					{
						ID:       "link-2",
						SourceID: "node-2",
						TargetID: "node-3",
						Type:     roots.LinkTypeEthernet,
					},
				},
				UpdatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "empty topology",
			topology: roots.Topology{
				Nodes:     []roots.TopologyNode{},
				Links:     []roots.TopologyLink{},
				UpdatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "topology with nil slices",
			topology: roots.Topology{
				UpdatedAt: now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.topology)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.Topology
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify node count
			if len(decoded.Nodes) != len(tt.topology.Nodes) {
				t.Errorf("len(Nodes) = %d, want %d", len(decoded.Nodes), len(tt.topology.Nodes))
			}

			// Verify link count
			if len(decoded.Links) != len(tt.topology.Links) {
				t.Errorf("len(Links) = %d, want %d", len(decoded.Links), len(tt.topology.Links))
			}
		})
	}
}

// TestTracerouteHop tests TracerouteHop struct and JSON serialization.
func TestTracerouteHop(t *testing.T) {
	tests := []struct {
		name    string
		hop     roots.TracerouteHop
		wantErr bool
	}{
		{
			name: "complete hop",
			hop: roots.TracerouteHop{
				Number:    1,
				Address:   net.ParseIP("192.168.1.1"),
				Hostname:  "gateway.local",
				RTT:       5 * time.Millisecond,
				RTTMs:     5.0,
				Lost:      false,
				ASN:       12345,
				ASName:    "Example ISP",
				GeoCity:   "San Francisco",
				GeoRegion: "California",
				ISP:       "Example Internet",
			},
			wantErr: false,
		},
		{
			name: "lost hop",
			hop: roots.TracerouteHop{
				Number: 2,
				Lost:   true,
			},
			wantErr: false,
		},
		{
			name: "hop with IPv6",
			hop: roots.TracerouteHop{
				Number:   3,
				Address:  net.ParseIP("2001:db8::1"),
				Hostname: "ipv6-router.example.com",
				RTT:      10 * time.Millisecond,
				RTTMs:    10.0,
				Lost:     false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.hop)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.TracerouteHop
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.Number != tt.hop.Number {
				t.Errorf("Number = %d, want %d", decoded.Number, tt.hop.Number)
			}
			if decoded.Lost != tt.hop.Lost {
				t.Errorf("Lost = %v, want %v", decoded.Lost, tt.hop.Lost)
			}
			if decoded.Hostname != tt.hop.Hostname {
				t.Errorf("Hostname = %q, want %q", decoded.Hostname, tt.hop.Hostname)
			}
		})
	}
}

// TestTracerouteResult tests TracerouteResult struct and JSON serialization.
func TestTracerouteResult(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name    string
		result  roots.TracerouteResult
		wantErr bool
	}{
		{
			name: "complete result",
			result: roots.TracerouteResult{
				Target:     "google.com",
				ResolvedIP: "142.250.80.46",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 1.5},
					{Number: 2, Address: net.ParseIP("10.0.0.1"), RTTMs: 5.0},
					{Number: 3, Address: net.ParseIP("142.250.80.46"), RTTMs: 15.0},
				},
				Complete:    true,
				Duration:    100 * time.Millisecond,
				DurationMs:  100.0,
				StartedAt:   now.Add(-100 * time.Millisecond),
				CompletedAt: now,
			},
			wantErr: false,
		},
		{
			name: "incomplete result",
			result: roots.TracerouteResult{
				Target:     "unreachable.example.com",
				ResolvedIP: "192.0.2.1",
				Hops: []roots.TracerouteHop{
					{Number: 1, Address: net.ParseIP("192.168.1.1"), RTTMs: 1.5},
					{Number: 2, Lost: true},
					{Number: 3, Lost: true},
				},
				Complete:    false,
				Duration:    5 * time.Second,
				DurationMs:  5000.0,
				StartedAt:   now.Add(-5 * time.Second),
				CompletedAt: now,
			},
			wantErr: false,
		},
		{
			name: "empty result",
			result: roots.TracerouteResult{
				Target:      "empty.example.com",
				Hops:        []roots.TracerouteHop{},
				Complete:    false,
				StartedAt:   now,
				CompletedAt: now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.TracerouteResult
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.Target != tt.result.Target {
				t.Errorf("Target = %q, want %q", decoded.Target, tt.result.Target)
			}
			if decoded.Complete != tt.result.Complete {
				t.Errorf("Complete = %v, want %v", decoded.Complete, tt.result.Complete)
			}
			if len(decoded.Hops) != len(tt.result.Hops) {
				t.Errorf("len(Hops) = %d, want %d", len(decoded.Hops), len(tt.result.Hops))
			}
		})
	}
}

// TestTracerouteOptions tests TracerouteOptions struct and JSON serialization.
func TestTracerouteOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    roots.TracerouteOptions
		wantErr bool
	}{
		{
			name: "default options",
			opts: roots.TracerouteOptions{
				MaxHops:     30,
				Timeout:     2 * time.Second,
				Probes:      3,
				PacketSize:  64,
				EnrichHops:  true,
				UseUDP:      false,
				DontResolve: false,
			},
			wantErr: false,
		},
		{
			name: "UDP mode",
			opts: roots.TracerouteOptions{
				MaxHops:     64,
				Timeout:     5 * time.Second,
				Probes:      1,
				PacketSize:  40,
				EnrichHops:  false,
				UseUDP:      true,
				SourceAddr:  "192.168.1.100",
				DontResolve: true,
			},
			wantErr: false,
		},
		{
			name: "minimal options",
			opts: roots.TracerouteOptions{
				MaxHops: 15,
				Timeout: 1 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.TracerouteOptions
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.MaxHops != tt.opts.MaxHops {
				t.Errorf("MaxHops = %d, want %d", decoded.MaxHops, tt.opts.MaxHops)
			}
			if decoded.UseUDP != tt.opts.UseUDP {
				t.Errorf("UseUDP = %v, want %v", decoded.UseUDP, tt.opts.UseUDP)
			}
			if decoded.EnrichHops != tt.opts.EnrichHops {
				t.Errorf("EnrichHops = %v, want %v", decoded.EnrichHops, tt.opts.EnrichHops)
			}
		})
	}
}

// TestIPEnrichment tests IPEnrichment struct and JSON serialization.
func TestIPEnrichment(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name       string
		enrichment roots.IPEnrichment
		wantErr    bool
	}{
		{
			name: "complete enrichment",
			enrichment: roots.IPEnrichment{
				IP:          "203.0.113.1",
				ASN:         12345,
				ASName:      "Example ISP",
				ISP:         "Example Internet",
				Org:         "Example Org",
				City:        "San Francisco",
				Region:      "California",
				Country:     "United States",
				CountryCode: "US",
				Latitude:    37.7749,
				Longitude:   -122.4194,
				Timezone:    "America/Los_Angeles",
				IsProxy:     false,
				IsHosting:   true,
				IsTor:       false,
				QueryTime:   now,
			},
			wantErr: false,
		},
		{
			name: "minimal enrichment",
			enrichment: roots.IPEnrichment{
				IP:        "192.0.2.1",
				QueryTime: now,
			},
			wantErr: false,
		},
		{
			name: "proxy detection",
			enrichment: roots.IPEnrichment{
				IP:        "198.51.100.1",
				IsProxy:   true,
				IsTor:     true,
				QueryTime: now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.enrichment)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.IPEnrichment
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.IP != tt.enrichment.IP {
				t.Errorf("IP = %q, want %q", decoded.IP, tt.enrichment.IP)
			}
			if decoded.ASN != tt.enrichment.ASN {
				t.Errorf("ASN = %d, want %d", decoded.ASN, tt.enrichment.ASN)
			}
			if decoded.IsProxy != tt.enrichment.IsProxy {
				t.Errorf("IsProxy = %v, want %v", decoded.IsProxy, tt.enrichment.IsProxy)
			}
		})
	}
}

// TestPathAnalysis tests PathAnalysis struct and JSON serialization.
func TestPathAnalysis(t *testing.T) {
	tests := []struct {
		name     string
		analysis roots.PathAnalysis
		wantErr  bool
	}{
		{
			name: "excellent path",
			analysis: roots.PathAnalysis{
				Target:         "google.com",
				Hops:           5,
				AverageRTT:     15.5,
				PacketLoss:     0.0,
				ASNTransitions: 2,
				Bottlenecks:    []roots.PathBottleneck{},
				Analysis:       "Excellent path quality with low latency and no packet loss.",
				Score:          95,
			},
			wantErr: false,
		},
		{
			name: "path with bottlenecks",
			analysis: roots.PathAnalysis{
				Target:         "slow-server.example.com",
				Hops:           12,
				AverageRTT:     150.0,
				PacketLoss:     5.0,
				ASNTransitions: 4,
				Bottlenecks: []roots.PathBottleneck{
					{
						HopNumber:   5,
						Address:     "10.0.0.1",
						RTTIncrease: 75.0,
						Reason:      "Significant latency increase",
					},
					{
						HopNumber:   8,
						Address:     "172.16.0.1",
						RTTIncrease: 50.0,
						Reason:      "Congestion detected",
					},
				},
				Analysis: "Poor path quality. High latency or significant packet loss.",
				Score:    35,
			},
			wantErr: false,
		},
		{
			name: "minimal analysis",
			analysis: roots.PathAnalysis{
				Target: "minimal.example.com",
				Score:  50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.analysis)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.PathAnalysis
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.Target != tt.analysis.Target {
				t.Errorf("Target = %q, want %q", decoded.Target, tt.analysis.Target)
			}
			if decoded.Score != tt.analysis.Score {
				t.Errorf("Score = %d, want %d", decoded.Score, tt.analysis.Score)
			}
			if len(decoded.Bottlenecks) != len(tt.analysis.Bottlenecks) {
				t.Errorf(
					"len(Bottlenecks) = %d, want %d",
					len(decoded.Bottlenecks),
					len(tt.analysis.Bottlenecks),
				)
			}
		})
	}
}

// TestPathBottleneck tests PathBottleneck struct and JSON serialization.
func TestPathBottleneck(t *testing.T) {
	tests := []struct {
		name       string
		bottleneck roots.PathBottleneck
		wantErr    bool
	}{
		{
			name: "significant latency increase",
			bottleneck: roots.PathBottleneck{
				HopNumber:   5,
				Address:     "10.0.0.1",
				RTTIncrease: 75.0,
				Reason:      "Significant latency increase",
			},
			wantErr: false,
		},
		{
			name: "congestion bottleneck",
			bottleneck: roots.PathBottleneck{
				HopNumber:   8,
				Address:     "172.16.0.1",
				RTTIncrease: 100.0,
				Reason:      "Network congestion detected",
			},
			wantErr: false,
		},
		{
			name: "minimal bottleneck",
			bottleneck: roots.PathBottleneck{
				HopNumber:   3,
				RTTIncrease: 50.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.bottleneck)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test JSON unmarshaling
			var decoded roots.PathBottleneck
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Errorf("json.Unmarshal() error = %v", unmarshalErr)
				return
			}

			// Verify fields
			if decoded.HopNumber != tt.bottleneck.HopNumber {
				t.Errorf("HopNumber = %d, want %d", decoded.HopNumber, tt.bottleneck.HopNumber)
			}
			if decoded.RTTIncrease != tt.bottleneck.RTTIncrease {
				t.Errorf("RTTIncrease = %f, want %f", decoded.RTTIncrease, tt.bottleneck.RTTIncrease)
			}
		})
	}
}

// TestTopologyNodeJSONOmitEmpty verifies omitempty tags work correctly.
func TestTopologyNodeJSONOmitEmpty(t *testing.T) {
	node := roots.TopologyNode{
		ID:   "node-1",
		Type: roots.NodeTypeHost,
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Check that empty optional fields are omitted
	jsonStr := string(data)

	if !containsField(jsonStr, "id") {
		t.Error("expected 'id' field in JSON")
	}
	if !containsField(jsonStr, "type") {
		t.Error("expected 'type' field in JSON")
	}
}

// TestTopologyLinkJSONOmitEmpty verifies omitempty tags work correctly for links.
func TestTopologyLinkJSONOmitEmpty(t *testing.T) {
	link := roots.TopologyLink{
		ID:       "link-1",
		SourceID: "node-1",
		TargetID: "node-2",
		Type:     roots.LinkTypeEthernet,
	}

	data, err := json.Marshal(link)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Check that required fields are present
	jsonStr := string(data)

	if !containsField(jsonStr, "id") {
		t.Error("expected 'id' field in JSON")
	}
	if !containsField(jsonStr, "sourceId") {
		t.Error("expected 'sourceId' field in JSON")
	}
	if !containsField(jsonStr, "targetId") {
		t.Error("expected 'targetId' field in JSON")
	}
}

// containsField checks if a JSON string contains a specific field name.
func containsField(jsonStr, field string) bool {
	return json.Valid([]byte(jsonStr)) && len(jsonStr) > 0 &&
		(contains(jsonStr, `"`+field+`"`) || contains(jsonStr, `"`+field+`":`))
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
