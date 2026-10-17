package api

import (
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

const testImage = "cat9k_iosxe image bytes"

func newDeviceImageTestServer(t *testing.T) (*Server, *database.DB) {
	t.Helper()
	s, db := newDeviceConfigTestServer(t)
	seedRoledUser(t, db, dcAdmin, roles.Admin)
	images, err := app.NewDeviceConfigImages(s.persistence)
	if err != nil {
		t.Fatalf("build image use-case: %v", err)
	}
	s.deviceImages = images
	return s, db
}

func uploadImage(t *testing.T, s *Server, name, hash, body string) *httptest.ResponseRecorder {
	t.Helper()
	q := url.Values{"name": {name}, "platform": {"cisco-ios"}, "hash": {hash}}
	return callDeviceConfigAs(t, s.handleDeviceImages, dcAdmin, http.MethodPost, deviceImagesPath+"?"+q.Encode(), body)
}

func callDeleteImage(t *testing.T, s *Server, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(newAuthedRequest(http.MethodDelete, deviceImagesPath+"/"+id, nil, dcAdmin))
	req.SetPathValue("imageId", id)
	rec := httptest.NewRecorder()
	s.handleDeviceImageByID(rec, req)
	return rec
}

func imageSHA512(s string) string {
	sum := sha512.Sum512([]byte(s))
	return hex.EncodeToString(sum[:])
}

// TestDeviceImageUploadThroughTheAPI is the slice's acceptance through the
// API: an admin uploads an image with its vendor hash, the file lands beside
// the database, an operator lists it and a viewer cannot, the admin deletes
// it, and both are audited.
func TestDeviceImageUploadThroughTheAPI(t *testing.T) {
	s, db := newDeviceImageTestServer(t)

	rec := uploadImage(t, s, "cat9k_iosxe.17.12.04.SPA.bin", imageSHA512(testImage), testImage)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	img := decodeInto[deviceconfig.Image](t, rec)
	if img.Size != int64(len(testImage)) || img.SHA512 != imageSHA512(testImage) || img.UploadedBy != dcAdmin {
		t.Errorf("uploaded image = %+v", img)
	}
	file := filepath.Join(filepath.Dir(db.Path()), "device-images", img.ID)
	if got, err := os.ReadFile(file); err != nil || string(got) != testImage {
		t.Fatalf("image file %s = %q, %v", file, got, err)
	}

	list := callDeviceConfigAs(t, s.handleDeviceImages, dcOperator, http.MethodGet, deviceImagesPath, "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), img.ID) {
		t.Errorf("list as an operator: %d %s", list.Code, list.Body.String())
	}
	if viewer := callDeviceConfigAs(
		t,
		s.handleDeviceImages,
		dcViewer,
		http.MethodGet,
		deviceImagesPath,
		"",
	); viewer.Code != http.StatusForbidden {
		t.Errorf("list as a viewer: %d, want 403", viewer.Code)
	}

	if del := callDeleteImage(t, s, img.ID); del.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", del.Code, del.Body.String())
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("image file after delete: %v, want gone", err)
	}
	if again := callDeleteImage(t, s, img.ID); again.Code != http.StatusNotFound {
		t.Errorf("second delete: %d, want 404", again.Code)
	}

	requireImageAudit(t, db, img.ID, "device_config.image.uploaded", "device_config.image.deleted")
}

func requireImageAudit(t *testing.T, db *database.DB, imageID string, want ...string) {
	t.Helper()
	rows, err := db.Query(t.Context(), `SELECT action, user FROM audit_log
		WHERE resource_type = ? AND resource_id = ? ORDER BY id`, deviceconfig.ImageAuditResource, imageID)
	if err != nil {
		t.Fatalf("read image audit: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var got []string
	for rows.Next() {
		var action, user string
		if err = rows.Scan(&action, &user); err != nil {
			t.Fatalf("scan image audit: %v", err)
		}
		if user != dcAdmin {
			t.Errorf("%s audited as %q, want %q", action, user, dcAdmin)
		}
		got = append(got, action)
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("iterate image audit: %v", err)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("image audit = %v, want %v", got, want)
	}
}

func TestDeviceImageUploadRefusals(t *testing.T) {
	s, _ := newDeviceImageTestServer(t)
	if rec := uploadImage(t, s, "taken.bin", imageSHA512(testImage), testImage); rec.Code != http.StatusCreated {
		t.Fatalf("first upload: %d %s", rec.Code, rec.Body.String())
	}

	for _, tc := range []struct {
		name, image, hash, body, want string
		code                          int
	}{
		{"corrupt download", "other.bin", imageSHA512("the real image"), testImage, "does not match", http.StatusBadRequest},
		{"a name taken", "taken.bin", imageSHA512("v2"), "v2", "already exists", http.StatusConflict},
		{"a path for a name", "../etc/passwd", imageSHA512(testImage), testImage, "file name", http.StatusBadRequest},
		{"MD5 only", "md5.bin", strings.Repeat("0", 32), testImage, "SHA-256 or SHA-512", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := uploadImage(t, s, tc.image, tc.hash, tc.body)
			if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.want) {
				t.Errorf("upload: %d %s, want %d with %q", rec.Code, rec.Body.String(), tc.code, tc.want)
			}
		})
	}
	if rec := callDeleteImage(t, s, "dim-nope"); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed id: %d, want 400", rec.Code)
	}
}
