package roots

import (
	"net"
	"time"
)

// TracerouteHop represents a single hop in a traceroute.
type TracerouteHop struct {
	Number    int           `json:"number"`
	Address   net.IP        `json:"address,omitempty"`
	Hostname  string        `json:"hostname,omitempty"`
	RTT       time.Duration `json:"rtt"`
	RTTMs     float64       `json:"rttMs"`
	Lost      bool          `json:"lost"`
	ASN       uint32        `json:"asn,omitempty"`
	ASName    string        `json:"asName,omitempty"`
	GeoCity   string        `json:"geoCity,omitempty"`
	GeoRegion string        `json:"geoRegion,omitempty"`
	ISP       string        `json:"isp,omitempty"`
}

// TracerouteResult contains the full result of a traceroute.
type TracerouteResult struct {
	Target      string          `json:"target"`
	ResolvedIP  string          `json:"resolvedIp"`
	Hops        []TracerouteHop `json:"hops"`
	Complete    bool            `json:"complete"`
	Duration    time.Duration   `json:"duration"`
	DurationMs  float64         `json:"durationMs"`
	StartedAt   time.Time       `json:"startedAt"`
	CompletedAt time.Time       `json:"completedAt"`
}

// TracerouteOptions configures a traceroute execution.
type TracerouteOptions struct {
	MaxHops     int           `json:"maxHops"`
	Timeout     time.Duration `json:"timeout"`
	Probes      int           `json:"probes"`
	PacketSize  int           `json:"packetSize"`
	EnrichHops  bool          `json:"enrichHops"` // Add ASN/geo data
	UseUDP      bool          `json:"useUdp"`
	SourceAddr  string        `json:"sourceAddr,omitempty"`
	DontResolve bool          `json:"dontResolve"`
}

// TopologyNode represents a node in the network topology.
type TopologyNode struct {
	ID        string            `json:"id"`
	Type      TopologyNodeType  `json:"type"`
	Label     string            `json:"label"`
	IP        string            `json:"ip,omitempty"`
	MAC       string            `json:"mac,omitempty"`
	Vendor    string            `json:"vendor,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	X         float64           `json:"x,omitempty"`
	Y         float64           `json:"y,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// TopologyNodeType categorizes network nodes.
type TopologyNodeType string

// TopologyNodeType values.
const (
	NodeTypeRouter   TopologyNodeType = "router"
	NodeTypeSwitch   TopologyNodeType = "switch"
	NodeTypeHost     TopologyNodeType = "host"
	NodeTypeGateway  TopologyNodeType = "gateway"
	NodeTypeFirewall TopologyNodeType = "firewall"
	NodeTypeAP       TopologyNodeType = "access_point"
	NodeTypeCloud    TopologyNodeType = "cloud"
	NodeTypeUnknown  TopologyNodeType = "unknown"
)

// TopologyLink represents a connection between two nodes.
type TopologyLink struct {
	ID        string            `json:"id"`
	SourceID  string            `json:"sourceId"`
	TargetID  string            `json:"targetId"`
	Type      TopologyLinkType  `json:"type"`
	Label     string            `json:"label,omitempty"`
	Bandwidth string            `json:"bandwidth,omitempty"`
	Latency   float64           `json:"latencyMs,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// TopologyLinkType categorizes link types.
type TopologyLinkType string

// TopologyLinkType values.
const (
	LinkTypeEthernet TopologyLinkType = "ethernet"
	LinkTypeWiFi     TopologyLinkType = "wifi"
	LinkTypeFiber    TopologyLinkType = "fiber"
	LinkTypeWAN      TopologyLinkType = "wan"
	LinkTypeVPN      TopologyLinkType = "vpn"
	LinkTypeUnknown  TopologyLinkType = "unknown"
)

// Topology represents the complete network topology graph.
type Topology struct {
	Nodes     []TopologyNode `json:"nodes"`
	Links     []TopologyLink `json:"links"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// IPEnrichment contains enriched information about an IP address.
type IPEnrichment struct {
	IP          string    `json:"ip"`
	ASN         uint32    `json:"asn,omitempty"`
	ASName      string    `json:"asName,omitempty"`
	ISP         string    `json:"isp,omitempty"`
	Org         string    `json:"org,omitempty"`
	City        string    `json:"city,omitempty"`
	Region      string    `json:"region,omitempty"`
	Country     string    `json:"country,omitempty"`
	CountryCode string    `json:"countryCode,omitempty"`
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	Timezone    string    `json:"timezone,omitempty"`
	IsProxy     bool      `json:"isProxy"`
	IsHosting   bool      `json:"isHosting"`
	IsTor       bool      `json:"isTor"`
	QueryTime   time.Time `json:"queryTime"`
}

// PathAnalysis contains analysis results for a network path.
type PathAnalysis struct {
	Target         string           `json:"target"`
	Hops           int              `json:"hops"`
	AverageRTT     float64          `json:"averageRttMs"`
	PacketLoss     float64          `json:"packetLossPercent"`
	ASNTransitions int              `json:"asnTransitions"`
	Bottlenecks    []PathBottleneck `json:"bottlenecks,omitempty"`
	Analysis       string           `json:"analysis"`
	Score          int              `json:"score"` // 0-100 path quality score
}

// PathBottleneck identifies a potential bottleneck in the path.
type PathBottleneck struct {
	HopNumber   int     `json:"hopNumber"`
	Address     string  `json:"address"`
	RTTIncrease float64 `json:"rttIncreaseMs"`
	Reason      string  `json:"reason"`
}
