// Package autotest runs the wired port checks a technician makes on arrival —
// link, cable, DHCP, DNS, gateway, VLAN and internet reachability — as one
// action with one verdict. Every check already exists on its own; this package
// only sequences them and grades the set.
//
// A stage that fails never stops the run. A port with no link fails DHCP,
// DNS, the gateway and the internet too, and the operator is better served by
// seeing each of those answers than by a run that stopped at the first one.
package autotest

import (
	"context"
	"time"
)

// Outcome grades one stage, and the run as a whole.
type Outcome string

// Stage outcomes. Skipped means the stage could not give an answer here (no
// TDR support, nothing to compare against); it never fails the run.
const (
	OutcomePass    Outcome = "pass"
	OutcomeWarn    Outcome = "warn"
	OutcomeFail    Outcome = "fail"
	OutcomeSkipped Outcome = "skipped"
)

// StageName identifies a stage on the wire.
type StageName string

// The stages, in the order a run performs them.
const (
	StageLink     StageName = "link"
	StageCable    StageName = "cable"
	StageDHCP     StageName = "dhcp"
	StageDNS      StageName = "dns"
	StageGateway  StageName = "gateway"
	StageVLAN     StageName = "vlan"
	StageInternet StageName = "internet"
)

// Reason says why a stage got its outcome. It is a code rather than a
// sentence so the UI can word it in the operator's language.
type Reason string

// Stage reasons.
const (
	ReasonOK              Reason = "ok"
	ReasonNoCarrier       Reason = "noCarrier"
	ReasonHalfDuplex      Reason = "halfDuplex"
	ReasonLinkUp          Reason = "linkUp"
	ReasonUnsupported     Reason = "unsupported"
	ReasonCableFault      Reason = "cableFault"
	ReasonIncompleteLease Reason = "incompleteLease"
	ReasonNoAddress       Reason = "noAddress"
	ReasonLeaseExpired    Reason = "leaseExpired"
	ReasonNoLeaseFound    Reason = "noLeaseFound"
	ReasonSlow            Reason = "slow"
	ReasonNoResolvers     Reason = "noResolvers"
	ReasonNoAnswer        Reason = "noAnswer"
	ReasonLossy           Reason = "lossy"
	ReasonNoGateway       Reason = "noGateway"
	ReasonUnreachable     Reason = "unreachable"
	ReasonNotAdvertised   Reason = "notAdvertised"
	ReasonVLANMismatch    Reason = "vlanMismatch"
	ReasonPartial         Reason = "partial"
	ReasonNoTargets       Reason = "noTargets"
	ReasonCancelled       Reason = "cancelled"
	ReasonError           Reason = "error"
)

// stageTimeout bounds one stage, so a check that hangs costs the run that
// stage and not the stages after it.
const stageTimeout = 30 * time.Second

// Spec is what a run is asked to check.
type Spec struct {
	// ExpectedVLAN is the native VLAN the port should be on. Zero means no
	// expectation: the VLAN stage then reports what the switch advertised.
	ExpectedVLAN int `json:"expectedVlan,omitempty" jsonschema:"minimum=0,maximum=4094"`
}

// Reading is one labelled reading behind a stage's outcome, such as the
// negotiated speed or the gateway's address.
type Reading struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// StageReport is one stage's answer.
type StageReport struct {
	Stage   StageName `json:"stage"   jsonschema:"enum=link,enum=cable,enum=dhcp,enum=dns,enum=gateway,enum=vlan,enum=internet"`
	Outcome Outcome   `json:"outcome" jsonschema:"enum=pass,enum=warn,enum=fail,enum=skipped"`
	Reason  Reason    `json:"reason"  jsonschema:"enum=ok,enum=noCarrier,enum=halfDuplex,enum=linkUp,enum=unsupported,enum=cableFault,enum=incompleteLease,enum=noAddress,enum=leaseExpired,enum=noLeaseFound,enum=slow,enum=noResolvers,enum=noAnswer,enum=lossy,enum=noGateway,enum=unreachable,enum=notAdvertised,enum=vlanMismatch,enum=partial,enum=noTargets,enum=cancelled,enum=error"`
	// Message is the underlying check's own error text, when it gave one.
	Message    string    `json:"message,omitempty"`
	Readings   []Reading `json:"readings,omitempty"`
	DurationMs int64     `json:"durationMs"`
}

// Report is a whole run: the verdict and every stage behind it.
type Report struct {
	Interface string `json:"interface"`
	// Verdict is fail when any stage failed, warn when any warned, else pass.
	Verdict    Outcome       `json:"verdict"    jsonschema:"enum=pass,enum=warn,enum=fail"`
	StartedAt  time.Time     `json:"startedAt"`
	DurationMs int64         `json:"durationMs"`
	Stages     []StageReport `json:"stages"`
}

// Stage is one check. Run returns its outcome; it is given a context bounded
// by stageTimeout.
type Stage struct {
	Name StageName
	Run  func(ctx context.Context, iface string, req Spec) StageReport
}

// Run performs every stage in order on iface and grades the set. Progress is
// reported after each stage. Once ctx is cancelled the remaining stages are
// recorded as skipped rather than run.
func Run(ctx context.Context, iface string, req Spec, stages []Stage, report func(float64)) *Report {
	start := time.Now()
	res := &Report{Interface: iface, StartedAt: start, Stages: make([]StageReport, 0, len(stages))}
	for i, st := range stages {
		res.Stages = append(res.Stages, runStage(ctx, iface, req, st))
		report(float64(i+1) / float64(len(stages)))
	}
	res.DurationMs = time.Since(start).Milliseconds()
	res.Verdict = verdict(res.Stages)
	return res
}

func runStage(ctx context.Context, iface string, req Spec, st Stage) StageReport {
	if ctx.Err() != nil {
		return StageReport{Stage: st.Name, Outcome: OutcomeSkipped, Reason: ReasonCancelled}
	}
	stageCtx, cancel := context.WithTimeout(ctx, stageTimeout)
	defer cancel()
	start := time.Now()
	r := st.Run(stageCtx, iface, req)
	r.Stage = st.Name
	r.DurationMs = time.Since(start).Milliseconds()
	return r
}

func verdict(stages []StageReport) Outcome {
	v := OutcomePass
	for _, s := range stages {
		switch s.Outcome {
		case OutcomeFail:
			return OutcomeFail
		case OutcomeWarn:
			v = OutcomeWarn
		case OutcomePass, OutcomeSkipped:
		}
	}
	return v
}
