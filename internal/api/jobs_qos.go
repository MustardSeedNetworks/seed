package api

// jobs_qos.go registers #400's DSCP preservation check as three job kinds.
// Across two hosts: qos-listen holds a port open on the far side for a bounded
// window and reads the marking every probe arrives with; qos-send puts the
// marked probes on the wire. On one host with a Wi-Fi and a wired interface,
// qos-single-host does both, sending out of one and capturing on the other.
// They are jobs rather than routes for the same reason as multicast-listen:
// each holds a socket for seconds, and the runner already provides the
// cancel, the concurrency ceiling and the handle a caller polls.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/qos"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

const (
	// qosSendJobKind is the registered kind name for a marked-probe send.
	qosSendJobKind = "qos-send"
	// qosListenJobKind is the registered kind name for the receiving half.
	qosListenJobKind = "qos-listen"
	// qosSingleHostJobKind is the registered kind name for the one-host check.
	qosSingleHostJobKind = "qos-single-host"
	// dscpVerificationFeature is the Pro licence feature every kind needs
	// (owner 2026-09-24).
	dscpVerificationFeature = "dscp_verification"
)

// errQoSParams is returned when a kind is submitted with no params: each
// needs at least a port or its interfaces.
var errQoSParams = errors.New("qos job requires params")

// qosSend, qosListen and qosSingleHost are the qos package's checks, behind
// seams so the kinds are testable without a socket.
type (
	qosSend       func(context.Context, qos.SendRequest) (*qos.SendResult, error)
	qosListen     func(context.Context, qos.ListenRequest) (*qos.ListenResult, error)
	qosSingleHost func(context.Context, qos.SingleHostRequest) (*qos.SingleHostResult, error)
)

// newQoSHandler returns the job Handler for one kind. A cancelled check
// returns what it has, so the job succeeds with a partial result.
func newQoSHandler[Req, Res any](kind string, run func(context.Context, Req) (*Res, error)) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		raw, ok := params.(json.RawMessage)
		if !ok || len(raw) == 0 {
			return nil, errQoSParams
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var req Req
		if err := dec.Decode(&req); err != nil {
			return nil, fmt.Errorf("invalid %s params: %w", kind, err)
		}
		return run(ctx, req)
	}
}

// registerQoSKinds registers every kind with injectable implementations.
func (s *Server) registerQoSKinds(send qosSend, listen qosListen, singleHost qosSingleHost) {
	if err := s.jobsRunner().Register(qosSendJobKind, newQoSHandler(qosSendJobKind, send)); err != nil {
		logging.GetLogger().Error("failed to register qos-send job kind", "error", err)
	}
	if err := s.jobsRunner().Register(qosListenJobKind, newQoSHandler(qosListenJobKind, listen)); err != nil {
		logging.GetLogger().Error("failed to register qos-listen job kind", "error", err)
	}
	singleHostKind := newQoSHandler(qosSingleHostJobKind, singleHost)
	if err := s.jobsRunner().Register(qosSingleHostJobKind, singleHostKind); err != nil {
		logging.GetLogger().Error("failed to register qos-single-host job kind", "error", err)
	}
}
