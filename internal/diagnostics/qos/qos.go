// Package qos answers whether the path between two probes preserves the DSCP
// marking a sender puts on its traffic (#400).
//
// The fault it isolates is the common QoS misconfiguration: a switch port
// that does not trust the marking, or an access point that maps WMM priority
// to the wrong wired DSCP, rewrites a voice call's EF to best effort and the
// call degrades only under load. The application cannot see that; the
// receiver can, because the marking arrives in the IP header.
//
// It is two halves on two hosts. Send puts a short burst of probes on the
// wire, each class with its own DSCP; Listen, on the far side, reads the DSCP
// every probe arrived with. Each probe carries the marking it was sent with
// and the set of classes in its run, so the listener alone can say which
// classes were preserved, rewritten or lost, without the two hosts talking.
package qos

import (
	"errors"
	"fmt"
	"net/netip"
	"time"
)

const (
	// DefaultWindow is how long a listen runs when the caller does not say:
	// enough to start the sender by hand on the far host.
	DefaultWindow = 10 * time.Second

	// MaxWindow bounds a listen. It holds a port open for probes from anyone,
	// so it closes on its own.
	MaxWindow = 60 * time.Second

	// DefaultCount is how many probes each class sends when the caller does
	// not say. Five survives a lost probe and still reads a remark that
	// applies to some packets and not others.
	DefaultCount = 5

	// MaxCount bounds the probes per class; this is a check, not a load test.
	MaxCount = 100

	// maxDSCP is the largest six-bit DSCP value.
	maxDSCP = 63
)

var (
	// ErrPort rejects a port outside 1-65535.
	ErrPort = errors.New("port must be between 1 and 65535")
	// ErrTarget rejects a send target that is not a unicast IP address.
	ErrTarget = errors.New("target must be a unicast IP address")
	// ErrDSCP rejects a DSCP value outside 0-63.
	ErrDSCP = errors.New("dscp values must be between 0 and 63")
	// ErrCount rejects a probe count outside 1-MaxCount.
	ErrCount = fmt.Errorf("count must be between 1 and %d", MaxCount)
	// ErrFamily rejects a listen family other than ipv4 or ipv6.
	ErrFamily = errors.New(`family must be "ipv4" or "ipv6"`)
)

// defaultClasses are the classes a send marks when the caller names none:
// voice, the four assured-forwarding classes an enterprise policy maps
// video, signalling and bulk data to, and best effort as the control.
func defaultClasses() []int { return []int{46, 34, 26, 18, 10, 0} }

// Verdict is what happened to one class between sender and listener.
type Verdict string

const (
	// VerdictPreserved means every probe of the class arrived marked as sent.
	VerdictPreserved Verdict = "preserved"
	// VerdictRemarked means every probe arrived with one other marking.
	VerdictRemarked Verdict = "remarked"
	// VerdictMixed means probes of the class arrived with differing
	// markings: a rewrite on some paths or queues and not others.
	VerdictMixed Verdict = "mixed"
	// VerdictLost means the run named the class but none of its probes
	// arrived, which is a policy that drops the class rather than remarks it.
	VerdictLost Verdict = "lost"
	// VerdictUnobserved means probes arrived but this platform cannot read
	// the marking they carried.
	VerdictUnobserved Verdict = "unobserved"
)

// Name returns the conventional name of a DSCP value (RFC 4594), or "" when
// it has none.
// Class selectors are multiples of eight (CS0-CS7); assured forwarding
// class x, drop precedence y is 8x+2y for x 1-4 and y 1-3 (AF11-AF43).
func Name(dscp int) string {
	const (
		ef          = 46
		voiceAdmit  = 44
		classStep   = 8
		lastClassCS = 56
		firstAF     = 10
		lastAF      = 38
		dropStep    = 2
	)
	switch {
	case dscp == ef:
		return "EF"
	case dscp == voiceAdmit:
		return "VOICE-ADMIT"
	case dscp%classStep == 0 && dscp >= 0 && dscp <= lastClassCS:
		return fmt.Sprintf("CS%d", dscp/classStep)
	case dscp >= firstAF && dscp <= lastAF && dscp%dropStep == 0:
		return fmt.Sprintf("AF%d%d", dscp/classStep, dscp%classStep/dropStep)
	}
	return ""
}

// SendRequest is one burst of marked probes, as a caller asks for it.
type SendRequest struct {
	// Target is the listening host's address.
	Target string `json:"target"`
	// Port is the port the far host's listen is on.
	Port int `json:"port"`
	// DSCP lists the classes to send, each 0-63; the defaults when empty.
	DSCP []int `json:"dscp,omitempty"`
	// Count is the probes per class; DefaultCount when zero.
	Count int `json:"count,omitempty"`
}

// SendResult is what a send put on the wire.
type SendResult struct {
	Target string `json:"target"`
	Port   int    `json:"port"`
	// RunID names this burst in the listener's result.
	RunID   string      `json:"runId"`
	Count   int         `json:"count"`
	Classes []SentClass `json:"classes"`
	// Marked is false on Windows, which ignores a socket's TOS unless policy
	// allows it: every probe then leaves unmarked, and a listener reading 0
	// for every class is seeing the sender, not the path.
	Marked bool `json:"marked"`
}

// SentClass is one class of a send.
type SentClass struct {
	DSCP int    `json:"dscp"`
	Name string `json:"name,omitempty"`
	Sent int    `json:"sent"`
}

// ListenRequest is one listen, as a caller asks for it.
type ListenRequest struct {
	Port int `json:"port"`
	// Family is "ipv4" (the default) or "ipv6".
	Family          string `json:"family,omitempty"          jsonschema:"enum=ipv4,enum=ipv6"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
}

// ListenResult is what a listen read off the probes that reached it.
type ListenResult struct {
	Port       int    `json:"port"`
	Family     string `json:"family"     jsonschema:"enum=ipv4,enum=ipv6"`
	ListenedMs int64  `json:"listenedMs"`
	// Probes counts the probes that reached the listen, duplicates included.
	Probes uint64 `json:"probes"`
	// Ignored counts datagrams on the port that were not probes, or were
	// probes of a run the result had no room to track.
	Ignored uint64 `json:"ignored"`
	// Runs are the bursts heard, one per sender and run.
	Runs []RunResult `json:"runs"`
	// RunsTruncated reports that more runs arrived than are tracked.
	RunsTruncated bool `json:"runsTruncated"`
	// DSCPObserved is false where the platform cannot read the marking a
	// datagram arrived with (Windows): every class is then "unobserved".
	DSCPObserved bool `json:"dscpObserved"`
}

// RunResult is one sender's burst as the listener received it.
type RunResult struct {
	RunID   string        `json:"runId"`
	Sender  string        `json:"sender"`
	Classes []ClassResult `json:"classes"`
	// Preserved is true when every class of the run was preserved.
	Preserved bool `json:"preserved"`
}

// ClassResult is one class of a run as the listener received it.
type ClassResult struct {
	SentDSCP int    `json:"sentDscp"`
	SentName string `json:"sentName,omitempty"`
	// Expected is the probes the sender sent for the class.
	Expected int `json:"expected"`
	// Received is the distinct probes of the class that arrived.
	Received int            `json:"received"`
	Observed []ObservedDSCP `json:"observed"`
	Verdict  Verdict        `json:"verdict"  jsonschema:"enum=preserved,enum=remarked,enum=mixed,enum=lost,enum=unobserved"`
}

// ObservedDSCP is one marking the class's probes arrived with.
type ObservedDSCP struct {
	DSCP  int    `json:"dscp"`
	Name  string `json:"name,omitempty"`
	Count int    `json:"count"`
}

// sendSpec is a validated SendRequest.
type sendSpec struct {
	target netip.AddrPort
	mask   uint64
	count  uint16
}

// classes returns the spec's DSCP values in ascending order.
func (s sendSpec) classes() []uint8 {
	var out []uint8
	for d := range uint8(maxDSCP + 1) {
		if s.mask&(1<<d) != 0 {
			out = append(out, d)
		}
	}
	return out
}

func parseSend(req SendRequest) (sendSpec, error) {
	addr, err := netip.ParseAddr(req.Target)
	if err != nil {
		return sendSpec{}, fmt.Errorf("%w: %q", ErrTarget, req.Target)
	}
	addr = addr.Unmap()
	if addr.IsUnspecified() || addr.IsMulticast() || addr == netip.AddrFrom4([4]byte{255, 255, 255, 255}) {
		return sendSpec{}, fmt.Errorf("%w: %s", ErrTarget, addr)
	}
	if req.Port < 1 || req.Port > 65535 {
		return sendSpec{}, fmt.Errorf("%w: %d", ErrPort, req.Port)
	}
	classes := req.DSCP
	if len(classes) == 0 {
		classes = defaultClasses()
	}
	var mask uint64
	for _, d := range classes {
		if d < 0 || d > maxDSCP {
			return sendSpec{}, fmt.Errorf("%w: %d", ErrDSCP, d)
		}
		mask |= 1 << d
	}
	count := DefaultCount
	if req.Count != 0 {
		count = req.Count
	}
	if count < 1 || count > MaxCount {
		return sendSpec{}, fmt.Errorf("%w: %d", ErrCount, count)
	}
	return sendSpec{target: netip.AddrPortFrom(addr, uint16(req.Port)), mask: mask, count: uint16(count)}, nil
}

// listenSpec is a validated ListenRequest.
type listenSpec struct {
	port   int
	v6     bool
	window time.Duration
}

func (s listenSpec) family() string {
	if s.v6 {
		return "ipv6"
	}
	return "ipv4"
}

func (s listenSpec) network() string {
	if s.v6 {
		return "udp6"
	}
	return "udp4"
}

func parseListen(req ListenRequest) (listenSpec, error) {
	if req.Port < 1 || req.Port > 65535 {
		return listenSpec{}, fmt.Errorf("%w: %d", ErrPort, req.Port)
	}
	var v6 bool
	switch req.Family {
	case "", "ipv4":
	case "ipv6":
		v6 = true
	default:
		return listenSpec{}, fmt.Errorf("%w: %q", ErrFamily, req.Family)
	}
	window := DefaultWindow
	if req.DurationSeconds > 0 {
		window = min(time.Duration(req.DurationSeconds)*time.Second, MaxWindow)
	}
	return listenSpec{port: req.Port, v6: v6, window: window}, nil
}
