package api

// Operating system image endpoints (P-D3, #3499, ADR-0035 §3):
//
//   POST   /api/v1/device-images?name=&platform=&hash=  upload an image (the body is the image)
//   GET    /api/v1/device-images                        the client's images
//   DELETE /api/v1/device-images/{imageId}              delete an image
//
// Every route is Pro (config_push). Uploading and deleting are admin only;
// listing is operator+, like reading push plans. An upload is refused unless
// its bytes match the vendor's published SHA-256 or SHA-512 (hash). Staging
// an image on a device is a later slice; nothing here talks to a device.

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	deviceImagesPath = APIVersionPrefix + "/device-images"
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
	}
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
