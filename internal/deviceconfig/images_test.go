package deviceconfig_test

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

type memImages struct {
	mu     sync.Mutex
	images []deviceconfig.Image
	nextID int
}

func (m *memImages) InsertImage(_ context.Context, img *deviceconfig.Image) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, have := range m.images {
		if have.ClientID == img.ClientID && have.Name == img.Name {
			return deviceconfig.ErrImageExists
		}
	}
	m.nextID++
	img.ID = fmt.Sprintf("dim-%012d", m.nextID)
	m.images = append(m.images, *img)
	return nil
}

func (m *memImages) ListImages(_ context.Context, clientID string) ([]deviceconfig.Image, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []deviceconfig.Image
	for _, img := range slices.Backward(m.images) {
		if img.ClientID == clientID {
			out = append(out, img)
		}
	}
	return out, nil
}

func (m *memImages) GetImage(_ context.Context, clientID, id string) (deviceconfig.Image, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, img := range m.images {
		if img.ID == id && img.ClientID == clientID {
			return img, nil
		}
	}
	return deviceconfig.Image{}, deviceconfig.ErrNotFound
}

func (m *memImages) DeleteImage(_ context.Context, clientID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, img := range m.images {
		if img.ID == id && img.ClientID == clientID {
			m.images = slices.Delete(m.images, i, i+1)
			return nil
		}
	}
	return deviceconfig.ErrNotFound
}

const imageBytes = "IOS-XE image bytes"

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func sha512Hex(s string) string {
	sum := sha512.Sum512([]byte(s))
	return hex.EncodeToString(sum[:])
}

func newImages(t *testing.T) (*deviceconfig.Images, *memImages, *memAudit, string) {
	t.Helper()
	dir := t.TempDir()
	store, audit := &memImages{}, &memAudit{}
	images, err := deviceconfig.NewImages(store, func() (*os.Root, error) { return os.OpenRoot(dir) }, audit.record)
	require.NoError(t, err)
	return images, store, audit, dir
}

func upload(name, platform, hash, body string) deviceconfig.ImageUpload {
	return deviceconfig.ImageUpload{
		ClientID: "default", Name: name, Platform: deviceconfig.Platform(platform), VendorHash: hash,
		Body: strings.NewReader(body), User: "admin1", Remote: "10.0.0.9",
	}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestUploadKeepsAnImageThatMatchesItsVendorHash(t *testing.T) {
	for _, tc := range []struct{ name, hash string }{
		{"SHA-512 as Cisco and Arista publish", strings.ToUpper(sha512Hex(imageBytes))},
		{"SHA-256 as Juniper publishes", sha256Hex(imageBytes)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			images, _, audit, dir := newImages(t)
			img, err := images.Upload(t.Context(), upload("cat9k.bin", "cisco-ios", tc.hash, imageBytes))
			require.NoError(t, err)
			require.Equal(t, int64(len(imageBytes)), img.Size)
			require.Equal(t, sha256Hex(imageBytes), img.SHA256)
			require.Equal(t, sha512Hex(imageBytes), img.SHA512)
			require.Equal(t, "admin1", img.UploadedBy)

			require.Equal(t, []string{img.ID}, dirNames(t, dir), "the file is named by id, nothing left over")
			got, err := os.ReadFile(filepath.Join(dir, img.ID))
			require.NoError(t, err)
			require.Equal(t, imageBytes, string(got))

			list, err := images.ListImages(t.Context(), "default")
			require.NoError(t, err)
			require.Equal(t, []deviceconfig.Image{img}, list)

			require.Len(t, audit.events, 1)
			e := audit.events[0]
			require.Equal(t, "device_config.image.uploaded", e.Action)
			require.Equal(t, deviceconfig.ImageAuditResource, e.ResourceType)
			require.Equal(t, img.ID, e.ResourceID)
			require.Equal(t, "admin1", e.User)
			require.Equal(t, img.SHA512, e.Detail["sha512"])
		})
	}
}

func TestUploadRefusesAnImageThatDoesNotMatch(t *testing.T) {
	images, store, audit, dir := newImages(t)
	_, err := images.Upload(t.Context(), upload("cat9k.bin", "cisco-ios", sha512Hex("another image"), imageBytes))

	var mismatch deviceconfig.ImageHashMismatchError
	require.ErrorAs(t, err, &mismatch)
	require.Equal(t, "SHA-512", mismatch.Digest)
	require.Empty(t, dirNames(t, dir), "the bytes are deleted")
	require.Empty(t, store.images)
	require.Len(t, audit.events, 1)
	require.Equal(t, "device_config.image.rejected", audit.events[0].Action)
	require.Equal(t, "cat9k.bin", audit.events[0].Detail["name"])
}

func TestUploadRefusesBeforeReadingTheImage(t *testing.T) {
	good := sha512Hex(imageBytes)
	for _, tc := range []struct {
		name, image, platform, hash, want string
	}{
		{"NX-OS is read-only", "n9k.bin", "cisco-nxos", good, "cannot be staged on cisco-nxos"},
		{"NETCONF has no staging", "x.bin", "netconf", good, "cannot be staged on netconf"},
		{"unknown platform", "x.bin", "hp-procurve", good, "platform must be"},
		{"a path", "../x.bin", "cisco-ios", good, "name must be a file name"},
		{"a device path", "flash:x.bin", "cisco-ios", good, "name must be a file name"},
		{"a hidden file", ".x.bin", "cisco-ios", good, "name must be a file name"},
		{"a space", "x y.bin", "cisco-ios", good, "name must be a file name"},
		{"no name", "", "cisco-ios", good, "name must be a file name"},
		{"MD5 is not enough", "x.bin", "cisco-ios", strings.Repeat("a", 32), "hash must be"},
		{"not hex", "x.bin", "cisco-ios", strings.Repeat("z", 128), "hash must be"},
		{"no hash", "x.bin", "cisco-ios", "", "hash must be"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			images, _, audit, dir := newImages(t)
			_, err := images.Upload(t.Context(), upload(tc.image, tc.platform, tc.hash, imageBytes))
			var ve deviceconfig.ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, ve.Msg, tc.want)
			require.Empty(t, dirNames(t, dir))
			require.Empty(t, audit.events)
		})
	}
}

func TestUploadRefusesANameTakenAndAnEmptyImage(t *testing.T) {
	images, _, _, dir := newImages(t)
	_, err := images.Upload(t.Context(), upload("eos.swi", "arista-eos", sha512Hex(imageBytes), imageBytes))
	require.NoError(t, err)

	_, err = images.Upload(t.Context(), upload("eos.swi", "arista-eos", sha512Hex("v2"), "v2"))
	require.ErrorIs(t, err, deviceconfig.ErrImageExists)

	_, err = images.Upload(t.Context(), upload("empty.swi", "arista-eos", sha512Hex(""), ""))
	var ve deviceconfig.ValidationError
	require.ErrorAs(t, err, &ve)
	require.Contains(t, ve.Msg, "empty")
	require.Len(t, dirNames(t, dir), 1)
}

func TestUploadRemovesAnUploadACrashLeftBehind(t *testing.T) {
	images, _, _, dir := newImages(t)
	stale, fresh := filepath.Join(dir, "STALE.part"), filepath.Join(dir, "INFLIGHT.part")
	for _, p := range []string{stale, fresh} {
		require.NoError(t, os.WriteFile(p, []byte("half"), 0o600))
	}
	old := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(stale, old, old))

	img, err := images.Upload(t.Context(), upload("junos.tgz", "juniper-junos", sha256Hex(imageBytes), imageBytes))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"INFLIGHT.part", img.ID}, dirNames(t, dir),
		"a day-old upload goes, one that may still be running stays")
}

func TestDeleteImageRemovesTheFileAndIsScopedToTheClient(t *testing.T) {
	images, store, audit, dir := newImages(t)
	img, err := images.Upload(t.Context(), upload("cat9k.bin", "cisco-ios", sha512Hex(imageBytes), imageBytes))
	require.NoError(t, err)

	require.ErrorIs(t, images.DeleteImage(t.Context(), "tenant-x", img.ID, "admin1", "", ""), deviceconfig.ErrNotFound)
	require.Len(t, dirNames(t, dir), 1, "another client's delete leaves the file")

	require.NoError(t, images.DeleteImage(t.Context(), "default", img.ID, "admin1", "10.0.0.9", "curl"))
	require.Empty(t, dirNames(t, dir))
	require.Empty(t, store.images)
	last := audit.events[len(audit.events)-1]
	require.Equal(t, "device_config.image.deleted", last.Action)
	require.Equal(t, img.ID, last.ResourceID)
}
