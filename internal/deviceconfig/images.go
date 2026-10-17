package deviceconfig

// Operating system images for staging (P-D3, #3499, ADR-0035 §3). An admin
// uploads an image with the hash its vendor publishes; Seed hashes the bytes
// as they arrive and keeps the image only when the two agree. Both SHA-256
// and SHA-512 are kept, because staging asks the device to verify its copy
// with the one its platform computes. Staging is in stage.go.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	// MaxImageBytes caps an image. The largest switch and router images Seed
	// stages (IOS-XE and Junos bundles) are under 2 GiB.
	MaxImageBytes int64 = 4 << 30
	// ImageAuditResource is the audit log resource type of image events.
	ImageAuditResource = "device_image"
	// partSuffix marks an upload still being written. An upload left behind
	// by a crash is removed once it is older than staleUpload.
	partSuffix  = ".part"
	staleUpload = 24 * time.Hour
	// imageFileMode keeps an image readable by Seed alone.
	imageFileMode = 0o600
)

// imageNamePattern is a file name every staging platform accepts on its flash
// file system: no path, no spaces, no leading dot.
var imageNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Image is an uploaded operating system image. Its file is named by ID in
// the image directory.
type Image struct {
	ID         string    `json:"id"`
	ClientID   string    `json:"clientId"`
	Name       string    `json:"name"`
	Platform   Platform  `json:"platform"`
	Size       int64     `json:"size"`
	SHA256     string    `json:"sha256"`
	SHA512     string    `json:"sha512"`
	UploadedBy string    `json:"uploadedBy"`
	UploadedAt time.Time `json:"uploadedAt"`
}

// ImageStore keeps image records.
type ImageStore interface {
	// InsertImage stores an image record, generating its id. A second image
	// of the same name for the client is ErrImageExists.
	InsertImage(ctx context.Context, img *Image) error
	// ListImages returns the client's images, newest first.
	ListImages(ctx context.Context, clientID string) ([]Image, error)
	// GetImage returns one image record, or answers ErrNotFound.
	GetImage(ctx context.Context, clientID, id string) (Image, error)
	// DeleteImage removes an image record, or answers ErrNotFound.
	DeleteImage(ctx context.Context, clientID, id string) error
}

// ErrImageExists is an upload whose name another image of the client has.
var ErrImageExists = errors.New("deviceconfig: an image with this name already exists")

// ImageUpload is an admin's upload. VendorHash is the hex SHA-256 or SHA-512
// the vendor publishes for the image; its length says which.
type ImageUpload struct {
	ClientID   string
	Name       string
	Platform   Platform
	VendorHash string
	Body       io.Reader
	User       string
	Remote     string
	UserAgent  string
}

// Images is the image use-case.
type Images struct {
	store ImageStore
	dir   func() (*os.Root, error)
	audit func(context.Context, PushAuditEvent) error
	now   func() time.Time
}

// NewImages builds the use-case over the image records, the directory the
// image files are kept in, opened per call, and the push audit log.
func NewImages(
	store ImageStore, dir func() (*os.Root, error), audit func(context.Context, PushAuditEvent) error,
) (*Images, error) {
	if store == nil || dir == nil || audit == nil {
		return nil, ErrUnavailable
	}
	return &Images{store: store, dir: dir, audit: audit, now: time.Now}, nil
}

// CheckUpload refuses an upload that can be refused before its bytes are
// read.
func (s *Images) CheckUpload(ctx context.Context, up ImageUpload) error {
	switch up.Platform {
	case PlatformCiscoIOS, PlatformAristaEOS, PlatformJuniperJunos:
	case PlatformCiscoNXOS, PlatformNETCONF:
		return ValidationError{Msg: fmt.Sprintf("images cannot be staged on %s targets", up.Platform)}
	default:
		return ValidationError{Msg: "platform must be cisco-ios, arista-eos or juniper-junos"}
	}
	if !imageNamePattern.MatchString(up.Name) {
		return ValidationError{Msg: "name must be a file name of letters, digits, dots, dashes and underscores"}
	}
	if _, err := vendorDigest(up.VendorHash); err != nil {
		return err
	}
	images, err := s.store.ListImages(ctx, up.ClientID)
	if err != nil {
		return err
	}
	for _, img := range images {
		if img.Name == up.Name {
			return ErrImageExists
		}
	}
	return nil
}

// vendorDigest returns the hash a vendor hash is checked with, by its length.
func vendorDigest(vendorHash string) (string, error) {
	if _, err := hex.DecodeString(vendorHash); err == nil {
		switch len(vendorHash) {
		case hex.EncodedLen(sha256.Size):
			return "SHA-256", nil
		case hex.EncodedLen(sha512.Size):
			return "SHA-512", nil
		}
	}
	return "", ValidationError{Msg: "hash must be the vendor's hex SHA-256 or SHA-512 of the image"}
}

// Upload stores an image whose bytes match its vendor hash. An image that
// does not match is deleted and refused, and the refusal is audited.
func (s *Images) Upload(ctx context.Context, up ImageUpload) (Image, error) {
	up.VendorHash = strings.ToLower(up.VendorHash)
	if err := s.CheckUpload(ctx, up); err != nil {
		return Image{}, err
	}
	root, err := s.dir()
	if err != nil {
		return Image{}, err
	}
	defer func() { _ = root.Close() }()
	s.removeStaleUploads(root)

	part := rand.Text() + partSuffix
	img, err := s.write(root, part, up)
	if err != nil {
		_ = root.Remove(part)
		if mismatch, ok := errors.AsType[ImageHashMismatchError](err); ok {
			return Image{}, errors.Join(err, s.audit(ctx, PushAuditEvent{
				Action:       "device_config.image.rejected",
				User:         up.User,
				ResourceType: ImageAuditResource,
				Remote:       up.Remote,
				UserAgent:    up.UserAgent,
				Detail: map[string]any{
					"name":     up.Name,
					"platform": up.Platform,
					"size":     img.Size,
					"hash":     mismatch.Digest,
				},
			}))
		}
		return Image{}, err
	}
	if err = s.store.InsertImage(ctx, &img); err != nil {
		_ = root.Remove(part)
		return Image{}, err
	}
	if err = root.Rename(part, img.ID); err != nil {
		_ = root.Remove(part)
		return Image{}, errors.Join(fmt.Errorf("keep image %s: %w", img.ID, err),
			s.store.DeleteImage(ctx, img.ClientID, img.ID))
	}
	if err = s.audit(ctx, PushAuditEvent{
		Action: "device_config.image.uploaded", User: up.User, ResourceType: ImageAuditResource, ResourceID: img.ID,
		Remote: up.Remote, UserAgent: up.UserAgent,
		Detail: map[string]any{"name": img.Name, "platform": img.Platform, "size": img.Size, "sha512": img.SHA512},
	}); err != nil {
		return Image{}, fmt.Errorf("audit image upload: %w", err)
	}
	return img, nil
}

// ImageHashMismatchError is an upload whose bytes are not the image its
// vendor hash names: a corrupt or truncated download, or another file.
type ImageHashMismatchError struct{ Digest string }

func (e ImageHashMismatchError) Error() string {
	return fmt.Sprintf("the image's %s does not match the vendor hash; download it again", e.Digest)
}

// write copies the upload to part, hashing it on the way, and returns the
// image record it makes. The file is synced before the record is returned.
func (s *Images) write(root *os.Root, part string, up ImageUpload) (Image, error) {
	f, err := root.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_EXCL, imageFileMode)
	if err != nil {
		return Image{}, fmt.Errorf("create image file: %w", err)
	}
	defer func() { _ = f.Close() }()

	h256, h512 := sha256.New(), sha512.New()
	n, err := io.Copy(io.MultiWriter(f, h256, h512), io.LimitReader(up.Body, MaxImageBytes+1))
	if err != nil {
		return Image{}, fmt.Errorf("receive image: %w", err)
	}
	switch {
	case n == 0:
		return Image{}, ValidationError{Msg: "the image is empty"}
	case n > MaxImageBytes:
		return Image{}, ValidationError{Msg: fmt.Sprintf("image exceeds %d bytes", MaxImageBytes)}
	}
	img := Image{
		ClientID:   up.ClientID,
		Name:       up.Name,
		Platform:   up.Platform,
		Size:       n,
		SHA256:     hexSum(h256),
		SHA512:     hexSum(h512),
		UploadedBy: up.User,
		UploadedAt: s.now().UTC().Truncate(time.Second),
	}
	digest, _ := vendorDigest(up.VendorHash)
	if got := map[string]string{"SHA-256": img.SHA256, "SHA-512": img.SHA512}[digest]; got != up.VendorHash {
		return img, ImageHashMismatchError{Digest: digest}
	}
	if err = f.Sync(); err != nil {
		return Image{}, fmt.Errorf("write image file: %w", err)
	}
	return img, nil
}

// removeStaleUploads deletes uploads a crash left half written. An upload in
// progress is younger than staleUpload, which no upload takes.
func (s *Images) removeStaleUploads(root *os.Root) {
	d, err := root.Open(".")
	if err != nil {
		return
	}
	defer func() { _ = d.Close() }()
	entries, err := d.ReadDir(-1)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), partSuffix) {
			continue
		}
		if info, infoErr := e.Info(); infoErr == nil && s.now().Sub(info.ModTime()) > staleUpload {
			_ = root.Remove(e.Name())
		}
	}
}

// ListImages returns the client's images, newest first.
func (s *Images) ListImages(ctx context.Context, clientID string) ([]Image, error) {
	return s.store.ListImages(ctx, clientID)
}

// open returns an image record and its file, for staging.
func (s *Images) open(ctx context.Context, clientID, id string) (Image, *os.File, error) {
	img, err := s.store.GetImage(ctx, clientID, id)
	if err != nil {
		return Image{}, nil, err
	}
	root, err := s.dir()
	if err != nil {
		return Image{}, nil, err
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open(img.ID)
	if err != nil {
		return Image{}, nil, fmt.Errorf("open image %s: %w", img.ID, err)
	}
	return img, f, nil
}

// DeleteImage removes an image and its file.
func (s *Images) DeleteImage(ctx context.Context, clientID, id, user, remote, userAgent string) error {
	root, err := s.dir()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if err = s.store.DeleteImage(ctx, clientID, id); err != nil {
		return err
	}
	if err = root.Remove(id); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove image file %s: %w", id, err)
	}
	if err = s.audit(ctx, PushAuditEvent{
		Action: "device_config.image.deleted", User: user, ResourceType: ImageAuditResource, ResourceID: id,
		Remote: remote, UserAgent: userAgent,
	}); err != nil {
		return fmt.Errorf("audit image delete: %w", err)
	}
	return nil
}

func hexSum(h hash.Hash) string { return hex.EncodeToString(h.Sum(nil)) }
