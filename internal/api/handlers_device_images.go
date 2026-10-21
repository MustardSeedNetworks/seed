package api

// Operating system image endpoints (P-D3, #3499, ADR-0035 §3):
//
//   POST   /api/v1/device-images?name=&platform=&hash=  upload an image (the body is the image)
//   GET    /api/v1/device-images                        the client's images
//   DELETE /api/v1/device-images/{imageId}              delete an image
//   POST   /api/v1/device-images/{imageId}/stage        stage an image on a target (a job)
//   POST   /api/v1/device-images/{imageId}/reload       reload a target onto a staged image (a job)
//
// Every route is Pro (config_push). Uploading, deleting and staging are admin
// only; listing is operator+, like reading push plans. An upload is refused
// unless its bytes match the vendor's published SHA-256 or SHA-512 (hash).
// Staging copies the image to the target with its push login, has the device
// verify the copy and sets it as the boot image; it never reloads the device.
// Like a plan it may be started by a personal access token, since nothing
// runs the image until a reload. A reload is its own admin action and, like
// an apply, refuses a token: it takes the device down and boots the image.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

const (
	deviceImagesPath = APIVersionPrefix + "/device-images"

	deviceImageStageJobKind  = "device-image-stage"
	deviceImageReloadJobKind = "device-image-reload"
	// imageUploadWindow is how long an upload may take: the largest image
	// over a 5 Mbit/s management link, where the server's usual 15 s read
	// timeout would cut every image off.
	imageUploadWindow = 2 * time.Hour
)

var deviceImageIDPattern = regexp.MustCompile(`^dim-[0-9a-f]{12}$`)

func (s *Server) deviceImageRoutes() []route.Route {
	admin := roles.Admin
	return []route.Route{
		{
			Path:         APIVersionPrefix + "/device-images",
			Handler:      s.handleDeviceImages,
			Methods:      []string{http.MethodGet, http.MethodPost},
			Scope:        admin,
			Feature:      configPushFeature,
			Limiter:      limitEndpoint,
			MaxBodyBytes: deviceconfig.MaxImageBytes,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:    APIVersionPrefix + "/device-images/{imageId}",
			Handler: s.handleDeviceImageByID,
			Methods: []string{http.MethodDelete},
			Scope:   admin,
			Feature: configPushFeature,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-images/{imageId}/stage",
			Handler: s.handleDeviceImageStage,
			Methods: []string{http.MethodPost},
			Scope:   admin,
			Feature: configPushFeature,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-images/{imageId}/reload",
			Handler: s.handleDeviceImageReload,
			Methods: []string{http.MethodPost},
			Scope:   admin,
			Feature: configPushFeature,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
	}
}

// imageTargetInput is the stage and reload POST body: the target to stage on
// or reload.
type imageTargetInput struct {
	TargetID string `json:"targetId"`
}

// imageStageJob is the staging job's params, a Go value like pushPlanJob.
type imageStageJob struct {
	Request deviceconfig.StageRequest
}

// imageReloadJob is the reload job's params.
type imageReloadJob struct {
	Request deviceconfig.ReloadRequest
}

func (s *Server) handleDeviceImages(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.listDeviceImages(w, r)
		return
	}
	clientID, ok := s.deviceImageCaller(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	up := deviceconfig.ImageUpload{
		ClientID:   clientID,
		Name:       q.Get("name"),
		Platform:   deviceconfig.Platform(q.Get("platform")),
		VendorHash: q.Get("hash"),
		Body:       r.Body,
		User:       usernameFromContext(r),
		Remote:     s.getClientIP(r),
		UserAgent:  r.UserAgent(),
	}
	if err := s.deviceImages.CheckUpload(r.Context(), up); err != nil {
		writeDeviceImageError(w, r, err)
		return
	}
	rc := http.NewResponseController(w)
	deadline := time.Now().Add(imageUploadWindow)
	if err := errors.Join(rc.SetReadDeadline(deadline), rc.SetWriteDeadline(deadline)); err != nil {
		logging.FromContext(r.Context()).WarnContext(r.Context(),
			"image upload keeps the server's read timeout", "error", err)
	}
	img, err := s.deviceImages.Upload(r.Context(), up)
	if err != nil {
		writeDeviceImageError(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusCreated, img)
}

func (s *Server) listDeviceImages(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, roles.Operator) {
		return
	}
	clientID, ok := s.deviceImageCaller(w, r)
	if !ok {
		return
	}
	list, err := s.deviceImages.ListImages(r.Context(), clientID)
	if err != nil {
		writeDeviceImageError(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, map[string]any{
		jsonKeyCount: len(list),
		"images":     list,
	})
}

func (s *Server) handleDeviceImageByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("imageId")
	if !deviceImageIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid image id")
		return
	}
	clientID, ok := s.deviceImageCaller(w, r)
	if !ok {
		return
	}
	if err := s.deviceImages.DeleteImage(r.Context(), clientID, id,
		usernameFromContext(r), s.getClientIP(r), r.UserAgent()); err != nil {
		writeDeviceImageError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeviceImageStage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("imageId")
	if !deviceImageIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid image id")
		return
	}
	clientID, ok := s.deviceConfigPushCaller(w, r)
	if !ok {
		return
	}
	in, ok := decodeImageTarget(w, r)
	if !ok {
		return
	}
	req := deviceconfig.StageRequest{
		ClientID:  clientID,
		ImageID:   id,
		TargetID:  in.TargetID,
		User:      usernameFromContext(r),
		Remote:    s.getClientIP(r),
		UserAgent: r.UserAgent(),
	}
	if err := s.deviceConfigPushes.CheckStage(r.Context(), req); err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	s.submitClientJob(w, r, clientID, deviceImageStageJobKind, imageStageJob{Request: req}, func(jobID string) {
		s.auditDeviceConfig(r, "device_config.image.stage", "Image staging started",
			"job_id", jobID, "image_id", id, "target_id", in.TargetID)
	})
}

func (s *Server) handleDeviceImageReload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("imageId")
	if !deviceImageIDPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid image id")
		return
	}
	if auth.IsAPITokenAuth(r.Context()) {
		logging.FromContext(r.Context()).WarnContext(r.Context(), "Forbidden: a token cannot reload a device",
			"event", "auth.forbidden",
			"reason", "scope",
			"username", usernameFromContext(r),
			"client_ip", GetClientIP(r),
			"path", singleLogLine(r.URL.Path),
			"method", r.Method,
		)
		writeError(w, r, http.StatusForbidden, ErrCodeForbidden,
			"A personal access token cannot reload a device; sign in to reload")
		return
	}
	clientID, ok := s.deviceConfigPushCaller(w, r)
	if !ok {
		return
	}
	in, ok := decodeImageTarget(w, r)
	if !ok {
		return
	}
	req := deviceconfig.ReloadRequest{
		ClientID:  clientID,
		ImageID:   id,
		TargetID:  in.TargetID,
		User:      usernameFromContext(r),
		Remote:    s.getClientIP(r),
		UserAgent: r.UserAgent(),
	}
	if err := s.deviceConfigPushes.CheckReload(r.Context(), req); err != nil {
		writeDeviceConfigError(w, r, err)
		return
	}
	s.submitClientJob(w, r, clientID, deviceImageReloadJobKind, imageReloadJob{Request: req}, func(jobID string) {
		s.auditDeviceConfig(r, "device_config.image.reload", "Device reload started",
			"job_id", jobID, "image_id", id, "target_id", in.TargetID)
	})
}

func decodeImageTarget(w http.ResponseWriter, r *http.Request) (imageTargetInput, bool) {
	var in imageTargetInput
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return in, false
	}
	if !deviceConfigTargetIDPattern.MatchString(in.TargetID) {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "targetId is required")
		return in, false
	}
	return in, true
}

// deviceImageCaller resolves the caller's client and answers 503 when the
// image use-case is not wired.
func (s *Server) deviceImageCaller(w http.ResponseWriter, r *http.Request) (string, bool) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return "", false
	}
	if s.deviceImages == nil {
		writeDeviceConfigError(w, r, deviceconfig.ErrUnavailable)
		return "", false
	}
	return clientID, true
}

func writeDeviceImageError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		writeError(w, r, http.StatusRequestEntityTooLarge, ErrCodeValidation, "The image is too large")
	case errors.Is(err, deviceconfig.ErrImageExists):
		writeError(w, r, http.StatusConflict, ErrCodeConflict,
			"An image with this name already exists; delete it first")
	default:
		if mismatch, ok := errors.AsType[deviceconfig.ImageHashMismatchError](err); ok {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, mismatch.Error())
			return
		}
		writeDeviceConfigError(w, r, err)
	}
}

// newDeviceImageStageHandler returns the job handler for staging. Params that
// are not an imageStageJob came through POST /jobs.
func newDeviceImageStageHandler(svc func() *deviceconfig.Pushes) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		p, ok := params.(imageStageJob)
		if !ok {
			return nil, fmt.Errorf("stage an image with POST %s/{imageId}/stage", deviceImagesPath)
		}
		pushes := svc()
		if pushes == nil {
			return nil, deviceconfig.ErrUnavailable
		}
		return pushes.Stage(ctx, p.Request)
	}
}

// newDeviceImageReloadHandler returns the job handler for a reload.
func newDeviceImageReloadHandler(svc func() *deviceconfig.Pushes) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		p, ok := params.(imageReloadJob)
		if !ok {
			return nil, fmt.Errorf("reload a device with POST %s/{imageId}/reload", deviceImagesPath)
		}
		pushes := svc()
		if pushes == nil {
			return nil, deviceconfig.ErrUnavailable
		}
		return pushes.Reload(ctx, p.Request)
	}
}
