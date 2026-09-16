package wifi

import "errors"

// Status describes whether an association was observed or its details were withheld.
type Status string

const (
	// StatusAssociated means the interface is joined to a network and the
	// platform named it. [Observation.Info] is non-nil.
	StatusAssociated Status = "associated"

	// StatusDetailsWithheld means the operating system refused to name the
	// network to this process. Whether the interface is associated is unknown:
	// the same refusal hides both the network and the association.
	StatusDetailsWithheld Status = "detailsWithheld"

	// StatusNotAssociated means the interface is not joined to any network.
	StatusNotAssociated Status = "notAssociated"
)

// Observation includes network details only when Status is StatusAssociated.
type Observation struct {
	Status      Status
	Info        *Info
	Reason      string
	Remediation string
}

// Observe reports the current association and how much of it is knowable.
func (m *Manager) Observe() Observation {
	m.mu.RLock()
	iface := m.interfaceName
	m.mu.RUnlock()

	return observePlatform(iface, m.currentHelper())
}

// ErrDetailsWithheld distinguishes redacted scan results from empty airspace.
var ErrDetailsWithheld = errors.New("wifi: the operating system withheld network identifiers from this process")

// WithheldExplanation returns the reason the identifiers are withheld and what
// an operator can do about it, or two empty strings on a platform that does not
// withhold them.
func WithheldExplanation() (string, string) {
	return withheldExplanationPlatform()
}
