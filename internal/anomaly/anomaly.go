// Package anomaly is the general, network-wide anomaly engine (ADR-0011): one
// typed stream of detections, a data-driven catalog of anomaly definitions, and
// a source-neutral engine that dedups, escalates, ages out, and projects a
// deterministic view. Wi-Fi is its first rule source (internal/wifi/anomaly),
// but the engine has no dependency on any source — it is exercised entirely with
// synthetic detections. CGO-free, no I/O, no hidden clock (callers pass time).
package anomaly

import "time"

// Severity is the anomaly urgency, aligned with the fleet alert vocabulary
// (internal/database AlertSeverity*). info < warning < critical.
type Severity string

const (
	// SeverityInfo is advisory — worth surfacing, not urgent.
	SeverityInfo Severity = "info"
	// SeverityWarning is a degradation or risk that should be addressed.
	SeverityWarning Severity = "warning"
	// SeverityCritical is an active problem or security exposure.
	SeverityCritical Severity = "critical"
)

// severity ranks order severities for escalation/coalescing (higher = more
// urgent). rankNone is the zero value for an unknown/invalid severity.
const (
	rankNone = iota
	rankInfo
	rankWarning
	rankCritical
)

// rank returns the ordering weight of s.
func (s Severity) rank() int {
	switch s {
	case SeverityCritical:
		return rankCritical
	case SeverityWarning:
		return rankWarning
	case SeverityInfo:
		return rankInfo
	default:
		return rankNone
	}
}

// valid reports whether s is a known severity.
func (s Severity) valid() bool { return s.rank() > 0 }

// Category encodes the problem DOMAIN, not the data source — one stream is
// filtered by category + severity (ADR-0011). Sources across Wi-Fi, wired/SNMP,
// and test outcomes map their detections onto this shared set.
type Category string

const (
	// CategorySecurity covers access-protection and attack exposure
	// (open/WEP/WPS, evil-twin, deauth flood, PMF gaps).
	CategorySecurity Category = "security"
	// CategoryRF covers PHY/RF conditions (co-channel, overlap, retries, noise).
	CategoryRF Category = "rf"
	// CategoryRoaming covers 802.11k/v/r and client mobility problems.
	CategoryRoaming Category = "roaming"
	// CategoryCapacity covers saturation/overload (BSS load, association caps).
	CategoryCapacity Category = "capacity"
	// CategoryStandards covers capability/standard inconsistencies across a BSS set.
	CategoryStandards Category = "standards"
	// CategoryAuthorization covers the rogue/allowlist authorization framework.
	CategoryAuthorization Category = "authorization"
	// CategoryNetHealth covers wired/SNMP device health (future sources).
	CategoryNetHealth Category = "nethealth"
)

// FollowUpKind selects how a follow-up narrows an ambiguous detection.
type FollowUpKind string

const (
	// FollowUpAuto runs a deeper test automatically — only where the platform/
	// adapter registers the named Capability; otherwise it degrades to a prompt.
	FollowUpAuto FollowUpKind = "auto"
	// FollowUpPrompt is a guided manual step for the user.
	FollowUpPrompt FollowUpKind = "prompt"
)

// FollowUp narrows the diagnosis for an anomaly type. An "auto" follow-up is
// capability-gated (ADR-0002): the engine degrades it to a prompt when its
// Capability is not registered, so the guidance is always actionable.
type FollowUp struct {
	Kind  FollowUpKind `json:"kind"`
	Label string       `json:"label"`
	// Action is the test to run (auto) or the step to take (prompt).
	Action string `json:"action"`
	// Capability names the platform capability an "auto" follow-up needs; empty
	// means always-auto. Ignored for prompt follow-ups.
	Capability string `json:"capability,omitempty"`
}

// Def is a catalog entry: the definition of an anomaly TYPE, separate
// from any single detection of it. Copy is authored originally with IEEE/802.11
// citations (ADR-0011) and lives as data so it is tunable and reviewable.
type Def struct {
	ID              string     `json:"id"`
	Category        Category   `json:"category"`
	DefaultSeverity Severity   `json:"defaultSeverity"`
	Standards       []string   `json:"standards,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Impact          string     `json:"impact"`
	Recommendation  string     `json:"recommendation"`
	FollowUps       []FollowUp `json:"followUps,omitempty"`
}

// SubjectKind classifies what an anomaly is about, so cross-source correlation
// can key on the same subject (e.g. a BSSID also seen in the wired ARP table).
type SubjectKind string

const (
	// SubjectSSID identifies an advertised network name.
	SubjectSSID SubjectKind = "ssid"
	// SubjectBSSID identifies one radio/BSSID.
	SubjectBSSID SubjectKind = "bssid"
	// SubjectClient identifies a client station MAC.
	SubjectClient SubjectKind = "client"
	// SubjectChannel identifies an RF channel.
	SubjectChannel SubjectKind = "channel"
	// SubjectDevice identifies a wired/discovered device (future sources).
	SubjectDevice SubjectKind = "device"
	// SubjectInterface identifies a device interface (future sources).
	SubjectInterface SubjectKind = "interface"
	// SubjectProbe identifies one configured active-monitoring probe by its
	// ProbeID — the correlation subject for probe breaches (ADR-0025).
	SubjectProbe SubjectKind = "probe"
)

// SubjectRef points at the entity an anomaly concerns.
type SubjectRef struct {
	Kind SubjectKind `json:"kind"`
	ID   string      `json:"id"`
}

// Detection is what a rule source emits to the engine: which catalog entry
// fired, about what subject, with the live evidence (measured values). An empty
// Severity means "use the catalog default".
type Detection struct {
	DefKey   string
	Subject  SubjectRef
	Severity Severity
	Evidence map[string]string
}

// key identifies the live instance a detection coalesces into: one per
// (anomaly type, subject). Re-detecting the same pair updates the instance
// rather than creating a duplicate.
func (d Detection) key() instanceKey {
	return instanceKey{def: d.DefKey, kind: d.Subject.Kind, id: d.Subject.ID}
}

type instanceKey struct {
	def  string
	kind SubjectKind
	id   string
}

// recordID renders the key as the stable persistence id (store.go). It must
// equal RecordID(def, SubjectRef{kind, id}) — the engine key and the stored id
// are the same identity, which is what makes restart re-load idempotent.
func (k instanceKey) recordID() string {
	return k.def + "|" + string(k.kind) + "|" + k.id
}

// Anomaly is the projected, JSON-serializable view of a live detection: the
// catalog copy (title/description/recommendation/standards) merged with the
// instance evidence and lifecycle (firstSeen/lastSeen/count). Wire tags are
// camelCase (ADR-0010).
type Anomaly struct {
	DefKey         string            `json:"defKey"`
	Category       Category          `json:"category"`
	Severity       Severity          `json:"severity"`
	Subject        SubjectRef        `json:"subject"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Impact         string            `json:"impact,omitempty"`
	Recommendation string            `json:"recommendation"`
	Standards      []string          `json:"standards,omitempty"`
	Evidence       map[string]string `json:"evidence,omitempty"`
	FollowUps      []FollowUp        `json:"followUps,omitempty"`
	FirstSeen      time.Time         `json:"firstSeen"`
	LastSeen       time.Time         `json:"lastSeen"`
	Count          int               `json:"count"`
}
