package api

// jobs_capture.go registers #326's packet capture as a job kind and serves
// the file it writes. Starting, following and stopping a capture are the
// runner's own POST /jobs, GET /jobs/{id} and DELETE /jobs/{id}: a stopped
// capture returns what it recorded, so the job succeeds with the file's ID,
// and GET /captures/{id} downloads it.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/packetcapture"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/paths"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

const (
	// packetCaptureJobKind is the registered kind name for a capture.
	packetCaptureJobKind = "packet-capture"
	// capturesPathPrefix is the /captures/{id} download route's prefix.
	capturesPathPrefix = APIVersionPrefix + "/captures/"
)

// packetCaptureRun is packetcapture.Run bound to its opener and store, behind
// a seam so the kind is testable without a capture handle.
type packetCaptureRun func(context.Context, packetcapture.Request, func(float64)) (*packetcapture.Result, error)

// newPacketCaptureHandler returns the job Handler for the kind. A capture
// that names no interface runs on the configured default one, which is the
// interface every other diagnostic uses.
func newPacketCaptureHandler(run packetCaptureRun, defaultInterface func() string) jobs.Handler {
	return func(ctx context.Context, params any, report func(float64)) (any, error) {
		var req packetcapture.Request
		if raw, ok := params.(json.RawMessage); ok && len(raw) > 0 {
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&req); err != nil {
				return nil, fmt.Errorf("invalid packet-capture params: %w", err)
			}
		}
		if req.Interface == "" {
			req.Interface = defaultInterface()
		}
		return run(ctx, req, report)
	}
}

// registerPacketCaptureKind registers the kind with an injectable run.
func (s *Server) registerPacketCaptureKind(run packetCaptureRun) {
	if err := s.jobsRunner().
		Register(packetCaptureJobKind, newPacketCaptureHandler(run, s.defaultInterface)); err != nil {
		logging.GetLogger().Error("failed to register packet-capture job kind", "error", err)
	}
}

// registerDefaultPacketCaptureKind wires the kind to the capture port and a
// store in the data directory, which the download route reads.
func (s *Server) registerDefaultPacketCaptureKind() {
	s.captures = packetcapture.NewStore(filepath.Join(paths.Resolve(paths.ModeAuto).DataDir, "captures"))
	opener := defaultCaptureOpener()
	s.registerPacketCaptureKind(func(
		ctx context.Context,
		req packetcapture.Request,
		report func(float64),
	) (*packetcapture.Result, error) {
		return packetcapture.Run(ctx, opener, s.captures, req, report)
	})
}

// handleCaptureDownload serves GET /api/v1/captures/{id}: the pcap file a
// finished capture wrote. A capture carries whatever crossed the wire,
// credentials in the clear included, so reading one takes the same operator
// role as starting one.
func (s *Server) handleCaptureDownload(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, database.RoleOperator) {
		return
	}
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	id := strings.TrimPrefix(r.URL.Path, capturesPathPrefix)
	f, err := s.captures.Open(id)
	switch {
	case errors.Is(err, packetcapture.ErrNotFound):
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.capture.notFound"), "")
		return
	case errors.Is(err, packetcapture.ErrInProgress):
		sendErrorResponseWithDetails(w, logger, http.StatusConflict,
			ErrCodeConflict, localizer.T("errors.capture.inProgress"), "")
		return
	case err != nil:
		logger.ErrorContext(r.Context(), "Failed to open capture", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.capture.readFailed"), "")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to stat capture", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.capture.readFailed"), "")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", `attachment; filename="seed-capture-`+id+`.pcap"`)
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// captureRoutes returns the capture download route. The ID is a random
// token the store validates before touching the filesystem.
func (s *Server) captureRoutes() []route {
	return []route{
		{
			path:    APIVersionPrefix + "/captures/",
			handler: s.handleCaptureDownload,
			methods: []string{http.MethodGet},
		},
	}
}
