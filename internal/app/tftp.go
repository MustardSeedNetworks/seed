package app

// tftp.go wires the TFTP session audit trail (P-D4) to the database audit log,
// so the API handler records starts, stops and transfers without importing
// internal/database (ADR-0020).

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/tftp"
)

// NewTFTPAudit returns the writer for TFTP audit events. The database is
// resolved on each call; with none wired the event is dropped.
func NewTFTPAudit(db func() *database.DB) func(context.Context, tftp.AuditEvent) error {
	return func(ctx context.Context, e tftp.AuditEvent) error {
		d := db()
		if d == nil {
			return nil
		}
		detail, err := json.Marshal(e.Detail)
		if err != nil {
			return fmt.Errorf("encode tftp audit detail: %w", err)
		}
		return d.RecordAuditLog(ctx, &database.AuditLogEntry{
			Action:       e.Action,
			User:         e.User,
			ResourceType: tftp.AuditResource,
			ResourceID:   e.Resource,
			NewValueJSON: string(detail),
			IPAddress:    e.Remote,
			UserAgent:    e.UserAgent,
		})
	}
}
