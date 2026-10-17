package database

import (
	"context"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

// InsertImage stores an image record, generating its id. A name the client
// already has is deviceconfig.ErrImageExists.
func (r *DeviceConfigRepository) InsertImage(ctx context.Context, img *deviceconfig.Image) error {
	img.ID = "dim-" + randomID()
	_, err := r.db.Exec(ctx, `INSERT INTO device_images (id, client_id, name, platform, size, sha256, sha512,
			uploaded_by, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		img.ID, img.ClientID, img.Name, string(img.Platform), img.Size, img.SHA256, img.SHA512,
		img.UploadedBy, img.UploadedAt.UTC().Format(time.RFC3339))
	if isUniqueConstraintError(err) {
		return deviceconfig.ErrImageExists
	}
	if err != nil {
		return fmt.Errorf("insert device_images: %w", err)
	}
	return nil
}

// ListImages returns the client's images, newest first.
func (r *DeviceConfigRepository) ListImages(ctx context.Context, clientID string) ([]deviceconfig.Image, error) {
	rows, err := r.db.Query(ctx, `SELECT id, client_id, name, platform, size, sha256, sha512, uploaded_by,
			uploaded_at
		FROM device_images WHERE client_id = ? ORDER BY uploaded_at DESC, id`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list device_images: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []deviceconfig.Image
	for rows.Next() {
		var (
			img                  deviceconfig.Image
			platform, uploadedAt string
		)
		if err = rows.Scan(&img.ID, &img.ClientID, &img.Name, &platform, &img.Size, &img.SHA256, &img.SHA512,
			&img.UploadedBy, &uploadedAt); err != nil {
			return nil, fmt.Errorf("scan device_images: %w", err)
		}
		img.Platform = deviceconfig.Platform(platform)
		img.UploadedAt = parseBackupTime(uploadedAt)
		out = append(out, img)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device_images: %w", err)
	}
	return out, nil
}

// DeleteImage removes an image record.
func (r *DeviceConfigRepository) DeleteImage(ctx context.Context, clientID, id string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM device_images WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return fmt.Errorf("delete device_images: %w", err)
	}
	return requireOneRow(res)
}
