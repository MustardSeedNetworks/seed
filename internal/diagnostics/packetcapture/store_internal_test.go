package packetcapture

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// finish creates a capture and releases it, dating its file mod so pruning
// order does not depend on the filesystem's timestamp resolution.
func finish(t *testing.T, s *Store, mod time.Time) string {
	t.Helper()
	id, f, err := s.create()
	require.NoError(t, err)
	require.NoError(t, f.Close())
	s.release(id)
	require.NoError(t, os.Chtimes(s.path(id), mod, mod))
	return id
}

func TestStoreKeepsTheNewestFinishedCaptures(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "captures"))
	base := time.Unix(1_700_000_000, 0)
	var ids []string
	for i := range keepFinished + 2 {
		ids = append(ids, finish(t, s, base.Add(time.Duration(i)*time.Minute)))
	}
	// A running capture is never pruned, however old its file.
	running, f, err := s.create()
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chtimes(s.path(running), base.Add(-time.Hour), base.Add(-time.Hour)))
	// Creating the next capture prunes to keepFinished-1, leaving room for it.
	next, f, err := s.create()
	require.NoError(t, err)
	require.NoError(t, f.Close())

	kept := ids[len(ids)-(keepFinished-1):]
	for _, id := range ids {
		_, statErr := os.Stat(s.path(id))
		if slices.Contains(kept, id) {
			assert.NoError(t, statErr, "newest finished capture %s must be kept", id)
		} else {
			assert.ErrorIs(t, statErr, os.ErrNotExist, "old capture %s must be pruned", id)
		}
	}
	assert.FileExists(t, s.path(running))
	assert.FileExists(t, s.path(next))
}

func TestStorePruneLeavesOtherFilesAlone(t *testing.T) {
	s := NewStore(t.TempDir())
	stranger := filepath.Join(s.dir, "notes.pcap")
	require.NoError(t, os.WriteFile(stranger, nil, 0o600))
	base := time.Unix(1_700_000_000, 0)
	// Older than every capture, so it would be the first pruned if the store
	// counted it as one.
	require.NoError(t, os.Chtimes(stranger, base.Add(-time.Hour), base.Add(-time.Hour)))
	for i := range keepFinished + 1 {
		finish(t, s, base.Add(time.Duration(i)*time.Minute))
	}
	assert.FileExists(t, stranger)
}

func TestStoreOpen(t *testing.T) {
	s := NewStore(t.TempDir())
	done := finish(t, s, time.Now())
	running, f, err := s.create()
	require.NoError(t, err)
	require.NoError(t, f.Close())
	outside := filepath.Join(filepath.Dir(s.dir), "secret.pcap")
	require.NoError(t, os.WriteFile(outside, nil, 0o600))

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "finished", id: done},
		{name: "still being written", id: running, wantErr: ErrInProgress},
		{name: "well-formed but unknown", id: "AAAAAAAAAAAAAAAAAAAAAAAAAA", wantErr: ErrNotFound},
		{name: "path traversal", id: "../secret", wantErr: ErrNotFound},
		{name: "lowercase", id: "aaaaaaaaaaaaaaaaaaaaaaaaaa", wantErr: ErrNotFound},
		{name: "empty", id: "", wantErr: ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opened, openErr := s.Open(tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, openErr, tt.wantErr)
				return
			}
			require.NoError(t, openErr)
			require.NoError(t, opened.Close())
		})
	}
}

func TestStoreFilesAreOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Go on Windows ignores the mode bits; the data directory's ACL applies")
	}
	s := NewStore(filepath.Join(t.TempDir(), "captures"))
	id := finish(t, s, time.Now())

	dir, err := os.Stat(s.dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dir.Mode().Perm())
	file, err := os.Stat(s.path(id))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), file.Mode().Perm())
}
