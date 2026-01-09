package roots_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots"
)

// TestTopologyService_Creation validates service creation.
func TestTopologyService_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "creation with nil config and nil db",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := roots.NewTopologyService(nil, nil)
			if (svc == nil) != tt.wantNil {
				t.Errorf("NewTopologyService() nil = %v, want nil = %v", svc == nil, tt.wantNil)
			}
		})
	}
}

// TestTopologyService_Start validates Start method.
func TestTopologyService_Start(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ctxTimeout  time.Duration
		cancelFirst bool
		wantErr     bool
	}{
		{
			name:       "normal start",
			ctxTimeout: 100 * time.Millisecond,
			wantErr:    false,
		},
		{
			name:        "start with cancelled context",
			ctxTimeout:  100 * time.Millisecond,
			cancelFirst: true,
			wantErr:     false, // Current implementation doesn't check context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := roots.NewTopologyService(nil, nil)
			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			if tt.cancelFirst {
				cancel()
			}

			err := svc.Start(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestTopologyService_Stop validates Stop method.
func TestTopologyService_Stop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		startFirst bool
	}{
		{
			name:       "stop without start",
			startFirst: false,
		},
		{
			name:       "stop after start",
			startFirst: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := roots.NewTopologyService(nil, nil)

			if tt.startFirst {
				ctx := context.Background()
				_ = svc.Start(ctx)
			}

			// Stop should not panic
			svc.Stop()
		})
	}
}

// TestTopologyService_StopMultiple validates multiple Stop calls don't panic.
func TestTopologyService_StopMultiple(t *testing.T) {
	t.Parallel()

	svc := roots.NewTopologyService(nil, nil)
	ctx := context.Background()
	_ = svc.Start(ctx)

	// Multiple stops should not panic
	for i := 0; i < 3; i++ {
		svc.Stop()
	}
}

// TestTopologyService_GetTopology_NotImplemented validates GetTopology returns error.
func TestTopologyService_GetTopology_NotImplemented(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "returns not implemented error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := roots.NewTopologyService(nil, nil)
			ctx := context.Background()

			result, err := svc.GetTopology(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTopology() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if result != nil {
				t.Errorf("GetTopology() should return nil result, got %+v", result)
			}
			if !errors.Is(err, roots.ErrNotImplemented) {
				t.Errorf("error = %v, want %v", err, roots.ErrNotImplemented)
			}
		})
	}
}

// TestTopologyService_ConcurrentOperations validates thread safety.
func TestTopologyService_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	svc := roots.NewTopologyService(nil, nil)
	ctx := context.Background()

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = svc.Start(ctx)
		}()
		go func() {
			defer wg.Done()
			svc.Stop()
		}()
		go func() {
			defer wg.Done()
			_, _ = svc.GetTopology(ctx)
		}()
	}

	wg.Wait()
}

// TestTopologyNodeType_Values validates TopologyNodeType constants.
func TestTopologyNodeType_Values(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			if string(tt.nodeType) != tt.want {
				t.Errorf("TopologyNodeType = %q, want %q", tt.nodeType, tt.want)
			}
		})
	}
}

// TestTopologyNodeType_Uniqueness validates all node types are unique.
func TestTopologyNodeType_Uniqueness(t *testing.T) {
	t.Parallel()

	nodeTypes := []roots.TopologyNodeType{
		roots.NodeTypeRouter,
		roots.NodeTypeSwitch,
		roots.NodeTypeHost,
		roots.NodeTypeGateway,
		roots.NodeTypeFirewall,
		roots.NodeTypeAP,
		roots.NodeTypeCloud,
		roots.NodeTypeUnknown,
	}

	seen := make(map[roots.TopologyNodeType]bool)
	for _, nt := range nodeTypes {
		if seen[nt] {
			t.Errorf("duplicate TopologyNodeType: %s", nt)
		}
		seen[nt] = true
	}
}

// TestTopologyLinkType_Values validates TopologyLinkType constants.
func TestTopologyLinkType_Values(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			if string(tt.linkType) != tt.want {
				t.Errorf("TopologyLinkType = %q, want %q", tt.linkType, tt.want)
			}
		})
	}
}

// TestTopologyLinkType_Uniqueness validates all link types are unique.
func TestTopologyLinkType_Uniqueness(t *testing.T) {
	t.Parallel()

	linkTypes := []roots.TopologyLinkType{
		roots.LinkTypeEthernet,
		roots.LinkTypeWiFi,
		roots.LinkTypeFiber,
		roots.LinkTypeWAN,
		roots.LinkTypeVPN,
		roots.LinkTypeUnknown,
	}

	seen := make(map[roots.TopologyLinkType]bool)
	for _, lt := range linkTypes {
		if seen[lt] {
			t.Errorf("duplicate TopologyLinkType: %s", lt)
		}
		seen[lt] = true
	}
}

// TestTopology_Struct validates Topology struct fields.
func TestTopology_Struct(t *testing.T) {
	t.Parallel()

	now := time.Now()
	topology := roots.Topology{
		Nodes: []roots.TopologyNode{
			{ID: "node-1", Type: roots.NodeTypeRouter},
			{ID: "node-2", Type: roots.NodeTypeSwitch},
		},
		Links: []roots.TopologyLink{
			{ID: "link-1", SourceID: "node-1", TargetID: "node-2", Type: roots.LinkTypeEthernet},
		},
		UpdatedAt: now,
	}

	if len(topology.Nodes) != 2 {
		t.Errorf("len(Nodes) = %d, want 2", len(topology.Nodes))
	}
	if len(topology.Links) != 1 {
		t.Errorf("len(Links) = %d, want 1", len(topology.Links))
	}
	if !topology.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", topology.UpdatedAt, now)
	}
}

// TestTopology_EmptySlices validates Topology with empty slices.
func TestTopology_EmptySlices(t *testing.T) {
	t.Parallel()

	topology := roots.Topology{
		Nodes: []roots.TopologyNode{},
		Links: []roots.TopologyLink{},
	}

	if topology.Nodes == nil {
		t.Error("Nodes should not be nil")
	}
	if topology.Links == nil {
		t.Error("Links should not be nil")
	}
	if len(topology.Nodes) != 0 {
		t.Errorf("len(Nodes) = %d, want 0", len(topology.Nodes))
	}
	if len(topology.Links) != 0 {
		t.Errorf("len(Links) = %d, want 0", len(topology.Links))
	}
}

// TestTopology_NilSlices validates Topology with nil slices.
func TestTopology_NilSlices(t *testing.T) {
	t.Parallel()

	var topology roots.Topology

	if topology.Nodes != nil {
		t.Error("zero Nodes should be nil")
	}
	if topology.Links != nil {
		t.Error("zero Links should be nil")
	}
}

// TestTopologyNode_Struct validates TopologyNode struct fields.
func TestTopologyNode_Struct(t *testing.T) {
	t.Parallel()

	now := time.Now()
	node := roots.TopologyNode{
		ID:        "node-1",
		Type:      roots.NodeTypeRouter,
		Label:     "Gateway Router",
		IP:        "192.168.1.1",
		MAC:       "00:11:22:33:44:55",
		Vendor:    "Cisco",
		Metadata:  map[string]string{"model": "ISR4451"},
		X:         100.0,
		Y:         200.0,
		UpdatedAt: now,
	}

	if node.ID != "node-1" {
		t.Errorf("ID = %q, want %q", node.ID, "node-1")
	}
	if node.Type != roots.NodeTypeRouter {
		t.Errorf("Type = %q, want %q", node.Type, roots.NodeTypeRouter)
	}
	if node.Label != "Gateway Router" {
		t.Errorf("Label = %q, want %q", node.Label, "Gateway Router")
	}
	if node.IP != "192.168.1.1" {
		t.Errorf("IP = %q, want %q", node.IP, "192.168.1.1")
	}
	if node.MAC != "00:11:22:33:44:55" {
		t.Errorf("MAC = %q, want %q", node.MAC, "00:11:22:33:44:55")
	}
	if node.Vendor != "Cisco" {
		t.Errorf("Vendor = %q, want %q", node.Vendor, "Cisco")
	}
	if node.Metadata["model"] != "ISR4451" {
		t.Errorf("Metadata[model] = %q, want %q", node.Metadata["model"], "ISR4451")
	}
	if node.X != 100.0 {
		t.Errorf("X = %f, want %f", node.X, 100.0)
	}
	if node.Y != 200.0 {
		t.Errorf("Y = %f, want %f", node.Y, 200.0)
	}
	if !node.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", node.UpdatedAt, now)
	}
}

// TestTopologyLink_Struct validates TopologyLink struct fields.
func TestTopologyLink_Struct(t *testing.T) {
	t.Parallel()

	now := time.Now()
	link := roots.TopologyLink{
		ID:        "link-1",
		SourceID:  "node-1",
		TargetID:  "node-2",
		Type:      roots.LinkTypeEthernet,
		Label:     "Primary Link",
		Bandwidth: "1Gbps",
		Latency:   0.5,
		Metadata:  map[string]string{"cable": "Cat6"},
		UpdatedAt: now,
	}

	if link.ID != "link-1" {
		t.Errorf("ID = %q, want %q", link.ID, "link-1")
	}
	if link.SourceID != "node-1" {
		t.Errorf("SourceID = %q, want %q", link.SourceID, "node-1")
	}
	if link.TargetID != "node-2" {
		t.Errorf("TargetID = %q, want %q", link.TargetID, "node-2")
	}
	if link.Type != roots.LinkTypeEthernet {
		t.Errorf("Type = %q, want %q", link.Type, roots.LinkTypeEthernet)
	}
	if link.Label != "Primary Link" {
		t.Errorf("Label = %q, want %q", link.Label, "Primary Link")
	}
	if link.Bandwidth != "1Gbps" {
		t.Errorf("Bandwidth = %q, want %q", link.Bandwidth, "1Gbps")
	}
	if link.Latency != 0.5 {
		t.Errorf("Latency = %f, want %f", link.Latency, 0.5)
	}
	if link.Metadata["cable"] != "Cat6" {
		t.Errorf("Metadata[cable] = %q, want %q", link.Metadata["cable"], "Cat6")
	}
	if !link.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", link.UpdatedAt, now)
	}
}
