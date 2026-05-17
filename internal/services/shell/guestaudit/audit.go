// Package guestaudit implements the on-demand guest-network isolation test
// (#397). In healthcare and other regulated environments, guest Wi-Fi must
// not be able to reach sensitive internal hosts (EMR, PACS, etc.). The
// technician connects the appliance to the guest network and runs this
// audit; it probes the configured target list and flags any reachable host
// as a Critical isolation failure.
package guestaudit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/validation"
)

// DefaultPorts returns the set of TCP ports probed against each target when
// the caller doesn't supply an explicit list. Tuned for services typically
// firewalled off from guest networks: web, RDP, SMB, databases, AD/LDAP.
func DefaultPorts() []int {
	return []int{80, 443, 445, 3389, 22, 3306, 5432, 1433, 8080, 8443}
}

const (
	tcpProbeTimeout   = 1500 * time.Millisecond
	icmpProbeTimeout  = 1500 * time.Millisecond
	defaultMaxWorkers = 16
)

// Target is the unit the audit probes.
type Target struct {
	IP    string `json:"ip"`
	Label string `json:"label,omitempty"`
}

// PortResult records the outcome of probing a single port on a target.
type PortResult struct {
	Port  int    `json:"port"`
	Open  bool   `json:"open"`
	Error string `json:"error,omitempty"`
}

// TargetResult collects the audit outcome for one target. Reachable=true
// means the audit considers isolation BROKEN for that target.
type TargetResult struct {
	Target          Target       `json:"target"`
	Reachable       bool         `json:"reachable"`
	PingResponded   bool         `json:"pingResponded"`
	OpenPorts       []int        `json:"openPorts"`
	PortResults     []PortResult `json:"portResults"`
	DurationSeconds float64      `json:"durationSeconds"`
}

// Report aggregates results for an audit run.
type Report struct {
	StartedAt        time.Time      `json:"startedAt"`
	CompletedAt      time.Time      `json:"completedAt"`
	IsolationFailed  bool           `json:"isolationFailed"`
	ReachableTargets int            `json:"reachableTargets"`
	TotalTargets     int            `json:"totalTargets"`
	Results          []TargetResult `json:"results"`
}

// Options configures an audit run.
type Options struct {
	Targets     []Target
	Ports       []int
	TCPTimeout  time.Duration // Per-port; 0 = DefaultTCPTimeout
	ICMPTimeout time.Duration // 0 = DefaultICMPTimeout
	Workers     int           // Concurrency cap across all (target, port) pairs; 0 = default
}

// Run executes the audit and returns a populated Report. Cancellation via ctx
// short-circuits cleanly; partial results are still returned.
//
//nolint:gocognit,cyclop,funlen // Fan-out probe orchestration is naturally branchy.
func Run(ctx context.Context, opts Options) (*Report, error) {
	if len(opts.Targets) == 0 {
		return nil, errors.New("no targets configured")
	}
	for _, t := range opts.Targets {
		if !validation.IsValidIP(t.IP) {
			return nil, fmt.Errorf("invalid target IP %q", t.IP)
		}
	}

	ports := opts.Ports
	if len(ports) == 0 {
		ports = DefaultPorts()
	}
	tcpTO := opts.TCPTimeout
	if tcpTO <= 0 {
		tcpTO = tcpProbeTimeout
	}
	icmpTO := opts.ICMPTimeout
	if icmpTO <= 0 {
		icmpTO = icmpProbeTimeout
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = defaultMaxWorkers
	}

	report := &Report{
		StartedAt:    time.Now(),
		TotalTargets: len(opts.Targets),
		Results:      make([]TargetResult, len(opts.Targets)),
	}

	// Worker pool that serves (target-index, port) jobs across all targets.
	type job struct {
		tIdx int
		port int
	}
	type result struct {
		tIdx int
		port int
		pr   PortResult
	}
	jobs := make(chan job)
	results := make(chan result, len(opts.Targets)*len(ports))

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for j := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				results <- result{
					tIdx: j.tIdx,
					port: j.port,
					pr:   probeTCP(ctx, opts.Targets[j.tIdx].IP, j.port, tcpTO),
				}
			}
		})
	}

	go func() {
		defer close(jobs)
		for tIdx := range opts.Targets {
			for _, port := range ports {
				select {
				case <-ctx.Done():
					return
				case jobs <- job{tIdx: tIdx, port: port}:
				}
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	starts := make([]time.Time, len(opts.Targets))
	for i := range starts {
		starts[i] = time.Now()
		report.Results[i] = TargetResult{
			Target:      opts.Targets[i],
			PortResults: make([]PortResult, 0, len(ports)),
		}
	}

	for r := range results {
		tr := &report.Results[r.tIdx]
		tr.PortResults = append(tr.PortResults, r.pr)
		if r.pr.Open {
			tr.OpenPorts = append(tr.OpenPorts, r.port)
			tr.Reachable = true
		}
	}

	// Ping each target (best-effort, doesn't block portscan). Reachable is
	// set true if ICMP responds OR any TCP port is open.
	for i := range report.Results {
		tr := &report.Results[i]
		tr.PingResponded = pingHost(ctx, tr.Target.IP, icmpTO)
		if tr.PingResponded {
			tr.Reachable = true
		}
		tr.DurationSeconds = time.Since(starts[i]).Seconds()
		if tr.Reachable {
			report.ReachableTargets++
		}
	}

	report.CompletedAt = time.Now()
	report.IsolationFailed = report.ReachableTargets > 0
	return report, nil
}

// probeTCP attempts a TCP connect to addr:port and reports open/closed.
// "Open" means the dial completed within the timeout. Anything else (refused,
// reset, filtered, timed out) counts as closed for isolation-audit purposes.
func probeTCP(ctx context.Context, ip string, port int, timeout time.Duration) PortResult {
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(dctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		// Distinguish "connection refused" (host responded but port closed)
		// from "timeout" (filtered) only in the error message; we still
		// treat both as not-open for the purposes of this audit.
		msg := err.Error()
		if strings.Contains(msg, "connection refused") {
			return PortResult{Port: port, Open: false}
		}
		return PortResult{Port: port, Open: false, Error: msg}
	}
	_ = conn.Close()
	return PortResult{Port: port, Open: true}
}

// pingHost attempts an ICMP echo via the system `ping` binary so we don't
// need CAP_NET_RAW. Returns true if the host responded.
func pingHost(ctx context.Context, ip string, timeout time.Duration) bool {
	return pingHostImpl(ctx, ip, timeout)
}
