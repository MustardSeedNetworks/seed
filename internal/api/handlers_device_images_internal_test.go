package api

import (
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
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

func callStageImage(t *testing.T, s *Server, imageID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(newAuthedRequest(http.MethodPost, deviceImagesPath+"/"+imageID+"/stage", []byte(body), dcAdmin))
	req.SetPathValue("imageId", imageID)
	rec := httptest.NewRecorder()
	s.handleDeviceImageStage(rec, req)
	return rec
}

// TestDeviceImageStageThroughTheAPI is the slice's acceptance through the
// API: an admin stages an uploaded image on an IOS target with pushing on,
// the job copies, verifies and sets the boot image with the push login, and
// every step is audited against the image.
func TestDeviceImageStageThroughTheAPI(t *testing.T) {
	s, db, device := newDeviceConfigPushTestServer(t)
	tgt, pushCred := pushReadyTargetOn(t, s, deviceconfig.PlatformCiscoIOS)
	const name = "cat9k_iosxe.17.12.04.SPA.bin"
	up := uploadImage(t, s, name, imageSHA512(testImage), testImage)
	if up.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", up.Code, up.Body.String())
	}
	img := decodeInto[deviceconfig.Image](t, up)

	rec := callStageImage(t, s, img.ID, `{"targetId":"`+tgt.ID+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("stage: %d %s", rec.Code, rec.Body.String())
	}
	jobID := decodeJob(t, rec.Body).ID
	var job jobs.Job
	waitFor(t, "staging finishes", func() bool {
		job, _ = s.jobsRunner().Get(jobID)
		return job.State.Terminal()
	})
	res, ok := job.Result.(deviceconfig.StageResult)
	if job.State != jobs.StateSucceeded || !ok || res.Outcome != deviceconfig.StageBootSet {
		t.Fatalf("staging job = %s %+v (%s), want boot_set", job.State, job.Result, job.Err)
	}
	if res.BootImage != "flash:"+name || res.RunningVersion != "17.09.04a" {
		t.Errorf("staging result = %+v", res)
	}

	device.mu.Lock()
	steps, requests := slices.Clone(device.steps), slices.Clone(device.requests)
	device.mu.Unlock()
	if want := []string{"facts", "copy flash:" + name, "verify", "boot"}; !slices.Equal(steps[len(steps)-4:], want) {
		t.Errorf("device steps = %v, want them to end %v", steps, want)
	}
	if last := requests[len(requests)-1]; last.Password != "wr1te" {
		t.Errorf("staging logged in with %q, want the push credential %s", last.Password, pushCred)
	}
	requireImageAudit(t, db, img.ID, "device_config.image.uploaded", "device_config.image.stage_started",
		"device_config.image.staged", "device_config.image.verified", "device_config.image.boot_set")
}

func TestDeviceImageStageRefusals(t *testing.T) {
	s, _, _ := newDeviceConfigPushTestServer(t)
	tgt, _ := pushReadyTarget(t, s)
	up := uploadImage(t, s, "cat9k.bin", imageSHA512(testImage), testImage)
	if up.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", up.Code, up.Body.String())
	}
	img := decodeInto[deviceconfig.Image](t, up)

	for _, tc := range []struct {
		name, imageID, body, want string
		code                      int
	}{
		{"bad image id", "nope", `{"targetId":"` + tgt.ID + `"}`, "invalid image id", http.StatusBadRequest},
		{"no target", img.ID, `{}`, "targetId is required", http.StatusBadRequest},
		{"unknown image", "dim-000000000000", `{"targetId":"` + tgt.ID + `"}`, "Not found", http.StatusNotFound},
		{
			"another platform", img.ID, `{"targetId":"` + tgt.ID + `"}`,
			"the image is for cisco-ios and the target is netconf", http.StatusBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := callStageImage(t, s, tc.imageID, tc.body)
			if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.want) {
				t.Errorf("stage = %d %s, want %d with %q", rec.Code, rec.Body.String(), tc.code, tc.want)
			}
		})
	}
}

func callReloadImage(t *testing.T, s *Server, req *http.Request, imageID string) *httptest.ResponseRecorder {
	t.Helper()
	req.SetPathValue("imageId", imageID)
	rec := httptest.NewRecorder()
	s.handleDeviceImageReload(rec, req)
	return rec
}

// TestDeviceImageReloadThroughTheAPI: a token cannot reload; an admin's
// reload of a target that boots the staged image next is a job that tells
// the device to reload with the push login and is audited against the
// image. The wait for the device to come back is the use-case's, tested
// there; here the job is cancelled once the device took the reload.
func TestDeviceImageReloadThroughTheAPI(t *testing.T) {
	s, db, device := newDeviceConfigPushTestServer(t)
	tgt, _ := pushReadyTargetOn(t, s, deviceconfig.PlatformCiscoIOS)
	const name = "cat9k_iosxe.17.12.04.SPA.bin"
	up := uploadImage(t, s, name, imageSHA512(testImage), testImage)
	if up.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", up.Code, up.Body.String())
	}
	img := decodeInto[deviceconfig.Image](t, up)
	stageID := decodeJob(t, callStageImage(t, s, img.ID, `{"targetId":"`+tgt.ID+`"}`).Body).ID
	waitFor(t, "staging finishes", func() bool {
		job, _ := s.jobsRunner().Get(stageID)
		return job.State.Terminal()
	})

	body := []byte(`{"targetId":"` + tgt.ID + `"}`)
	path := deviceImagesPath + "/" + img.ID + "/reload"
	pat := withClaim(newAuthedRequest(http.MethodPost, path, body, dcAdmin))
	pat = pat.WithContext(auth.WithAPITokenAuth(pat.Context()))
	if rec := callReloadImage(t, s, pat, img.ID); rec.Code != http.StatusForbidden ||
		!strings.Contains(rec.Body.String(), "personal access token") {
		t.Fatalf("reload by token: %d %s, want 403", rec.Code, rec.Body.String())
	}
	if rec := callReloadImage(t, s, withClaim(newAuthedRequest(http.MethodPost, path, []byte(`{}`), dcAdmin)),
		img.ID); rec.Code != http.StatusBadRequest {
		t.Errorf("reload without a target: %d %s, want 400", rec.Code, rec.Body.String())
	}

	rec := callReloadImage(t, s, withClaim(newAuthedRequest(http.MethodPost, path, body, dcAdmin)), img.ID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reload: %d %s", rec.Code, rec.Body.String())
	}
	jobID := decodeJob(t, rec.Body).ID
	waitFor(t, "the device takes the reload", func() bool {
		device.mu.Lock()
		defer device.mu.Unlock()
		return slices.Contains(device.steps, "reload")
	})
	if err := s.jobsRunner().Cancel(jobID); err != nil {
		t.Fatalf("cancel the reload job: %v", err)
	}
	waitFor(t, "the reload job ends", func() bool {
		job, _ := s.jobsRunner().Get(jobID)
		return job.State.Terminal()
	})

	device.mu.Lock()
	requests := slices.Clone(device.requests)
	device.mu.Unlock()
	if last := requests[len(requests)-1]; last.Password != "wr1te" {
		t.Errorf("the reload logged in with %q, want the push credential", last.Password)
	}
	requireImageAudit(t, db, img.ID, "device_config.image.uploaded", "device_config.image.stage_started",
		"device_config.image.staged", "device_config.image.verified", "device_config.image.boot_set",
		"device_config.image.reload_requested", "device_config.image.reload_sent",
		"device_config.image.reload_failed")
}
