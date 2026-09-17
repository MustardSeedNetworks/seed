// SPDX-License-Identifier: BUSL-1.1

package dns

// Scope says whose resolvers a result's server list describes.
//
// The Network page used to show the host's resolvers whatever interface was
// selected, beside a gateway taken from the system default route (#2690).
// Naming the scope is what lets the card stop implying that a system-wide
// answer belongs to the selected link.
type Scope string

const (
	// ScopeInterface means the list is the selected interface's own
	// resolvers. An empty list under this scope is an honest absence: the
	// interface has none, not "we could not tell".
	ScopeInterface Scope = "interface"
	// ScopeSystem means this host cannot attribute resolvers to an
	// interface, or none is selected, so the list is the host-wide one.
	ScopeSystem Scope = "system"
)

// resolverSource holds the two resolver reads the scoping rule needs. The
// Tester carries one so the rule can be exercised without a host whose
// resolver configuration happens to say the right thing; production is always
// built from systemResolvers.
type resolverSource struct {
	system   func() []string
	forIface func(string) ([]string, bool)
}

func systemResolvers() resolverSource {
	return resolverSource{system: GetSystemDNS, forIface: interfaceResolversPlatform}
}

// resolversFor returns the resolvers to show and test for iface, and whose
// they are.
//
// Three answers, not two. An interface with resolvers of its own gets them; an
// interface with none gets an empty interface-scoped list, because a link
// without a resolver cannot resolve and saying so is the point; a host that
// cannot attribute resolvers to interfaces at all gets the system list under
// ScopeSystem, so the card can say that is what it is showing.
func (r resolverSource) resolversFor(iface string) ([]string, Scope) {
	if iface == "" {
		return r.system(), ScopeSystem
	}

	servers, attributable := r.forIface(iface)
	if !attributable {
		return r.system(), ScopeSystem
	}
	return servers, ScopeInterface
}

// InterfaceResolvers returns the resolvers configured for iface and whether
// this host can attribute resolvers to an interface at all. Implementation is
// platform-specific (scoped_darwin.go, scoped_linux.go, scoped_windows.go).
func InterfaceResolvers(iface string) ([]string, bool) {
	return interfaceResolversPlatform(iface)
}
