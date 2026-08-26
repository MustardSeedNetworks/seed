package api

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// targetedScanRoutes are the endpoints that reach out to a host the caller
// names. That is the property that matters: an unbounded number of requests
// aimed at anything the caller chooses is a very different exposure from a
// sweep of whatever the local radio can hear.
//
// Listed rather than pattern-matched, because a path ending in "/scan" is a
// weak signal and a new targeted scan should be a deliberate addition here.
func targetedScanRoutes() []string {
	return []string{
		"/security/discovery/portscan",
		"/security/devices/scan",
		"/security/vulnerabilities/scan",
	}
}

// untargetedScanRoutes sweep local airspace or local state and take no target.
// They are not covered by the rate-limit invariant below. Whether they should
// be is a fair question — they are unbounded local operations — but it is a
// separate one from #347.
func untargetedScanRoutes() []string {
	return []string{
		"/security/bluetooth/scan",
		"/security/wifi/discovery/scan",
		"/security/problems/scan",
		"/discovery/engine/scan",
		"/wifi/wifi/scan",
	}
}

// newRoutePolicyServer builds the minimum Server the route registry needs. The
// policy manifest is assembled at registration time and does not depend on any
// of the handlers' dependencies, so nothing else has to be wired.
func newRoutePolicyServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{mux: http.NewServeMux()}
	s.setupRoutes()
	return s
}

// TestTargetedScansAreRateLimited pins the convention the route table already
// mostly followed. portscan carried neither a rate limit nor a role gate while
// its targeted siblings had at least the former, so any authenticated user
// could scan arbitrary hosts as fast as they could ask (#347).
func TestTargetedScansAreRateLimited(t *testing.T) {
	s := newRoutePolicyServer(t)

	byPath := make(map[string]route, len(s.manifest))
	for _, rt := range s.manifest {
		byPath[rt.path] = rt
	}

	for _, suffix := range targetedScanRoutes() {
		path := APIVersionPrefix + suffix
		t.Run(suffix, func(t *testing.T) {
			rt, ok := byPath[path]
			if !ok {
				t.Fatalf("%s is not registered; this list is stale", path)
			}
			if !rt.rateLimited {
				t.Errorf("%s is an active scan with no rate limit", path)
			}
		})
	}
}

// TestPortScanRequiresOperator pins the gate specifically, because it is the
// one this change adds and the one a future refactor is most likely to drop.
func TestPortScanRequiresOperator(t *testing.T) {
	s := newRoutePolicyServer(t)

	for _, rt := range s.manifest {
		if rt.path != APIVersionPrefix+"/security/discovery/portscan" {
			continue
		}
		if rt.minRole != database.RoleOperator {
			t.Errorf("portscan minRole = %q, want %q — scanning arbitrary hosts "+
				"is an action on the network, not a read of it",
				rt.minRole, database.RoleOperator)
		}
		if !rt.rateLimited {
			t.Error("portscan is not rate limited")
		}
		return
	}
	t.Fatal("the portscan route is not registered")
}

// TestInsecureProfileIsWiredToTheSharedList pins that the "insecure" profile
// resolves to the same constant the discovery preset uses. Two definitions of
// "insecure" that drift apart would mean the on-demand scan and the discovery
// scan disagree about what they are looking for.
func TestInsecureProfileIsWiredToTheSharedList(t *testing.T) {
	ports := insecurePorts()
	if len(ports) == 0 {
		t.Fatal("the insecure profile expands to no ports, so it would scan nothing")
	}

	for _, want := range []int{21, 23, 80} {
		if !slices.Contains(ports, want) {
			t.Errorf("the insecure profile omits port %d", want)
		}
	}
}

func TestPortScanWorkers(t *testing.T) {
	for _, tc := range []struct {
		requested, want int
	}{
		{0, defaultPortScanWorkers},
		{-1, defaultPortScanWorkers},
		{5, 5},
		{100, 100},
	} {
		if got := portScanWorkers(tc.requested); got != tc.want {
			t.Errorf("portScanWorkers(%d) = %d, want %d", tc.requested, got, tc.want)
		}
	}
}

// TestScanRoutesAreAllClassified guards both lists: a new route ending in
// "/scan" has to be put in one of them, which forces the question of whether it
// takes a caller-supplied target.
func TestScanRoutesAreAllClassified(t *testing.T) {
	s := newRoutePolicyServer(t)

	// Reads that happen to carry "scan" in their path.
	known := make(map[string]bool)
	for _, suffix := range append(targetedScanRoutes(), untargetedScanRoutes()...) {
		known[APIVersionPrefix+suffix] = true
	}

	for _, rt := range s.manifest {
		if strings.HasSuffix(rt.path, "/scan") && !known[rt.path] {
			t.Errorf("%s ends in /scan but is in neither targetedScanRoutes nor "+
				"untargetedScanRoutes; classify it", rt.path)
		}
	}
}
