package packetcapture

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// fileSuffix names every capture file the store owns; pruning never
	// touches anything else in the directory.
	fileSuffix = ".pcap"
	// idLen is the length of a [rand.Text] token, which names a capture.
	idLen = 26
	// keepFinished is how many finished captures stay on disk. Starting a
	// capture removes the oldest beyond it, so the directory is bounded by
	// keepFinished × MaxFileBytes plus whatever is being written.
	keepFinished = 4
)

var (
	// ErrNotFound is returned by Open for an unknown or malformed ID.
	ErrNotFound = errors.New("capture not found")
	// ErrInProgress is returned by Open while the capture is still being
	// written: the file is incomplete until the job ends.
	ErrInProgress = errors.New("capture still in progress")
)

// Store owns the capture files in one directory. It creates the directory on
// the first capture, not at construction, so a daemon that never captures
// never writes to disk.
type Store struct {
	dir    string
	mu     sync.Mutex
	active map[string]struct{}
}

// NewStore returns a store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{dir: dir, active: make(map[string]struct{})}
}

// create reserves a new ID, removes finished captures beyond keepFinished and
// opens the new file for writing. The caller must call release when writing
// ends, and remove as well if the capture failed.
func (s *Store) create() (string, *os.File, error) {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", nil, fmt.Errorf("create capture directory: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.prune(); err != nil {
		return "", nil, err
	}
	id := rand.Text()
	f, err := os.OpenFile(s.path(id), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", nil, fmt.Errorf("create capture file: %w", err)
	}
	s.active[id] = struct{}{}
	return id, f, nil
}

// release marks a capture finished, making it readable and prunable.
func (s *Store) release(id string) {
	s.mu.Lock()
	delete(s.active, id)
	s.mu.Unlock()
}

// remove deletes a capture that failed, so no job result points at it and no
// partial file outlives it.
func (s *Store) remove(id string) {
	s.release(id)
	_ = os.Remove(s.path(id))
}

// Open returns a finished capture for reading.
func (s *Store) Open(id string) (*os.File, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	s.mu.Lock()
	_, running := s.active[id]
	s.mu.Unlock()
	if running {
		return nil, ErrInProgress
	}
	// The ID arrives over HTTP. validID already rules out a separator, and
	// opening through an os.Root keeps the read inside the directory even if
	// that check ever loosens.
	root, err := os.OpenRoot(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open capture directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open(id + fileSuffix)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open capture: %w", err)
	}
	return f, nil
}

// prune removes the oldest finished captures so that, with the one about to
// be created, at most keepFinished finished captures remain. Called with s.mu
// held.
func (s *Store) prune() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("list captures: %w", err)
	}
	type finished struct {
		id  string
		mod time.Time
	}
	var done []finished
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), fileSuffix)
		if !ok || !validID(id) || !e.Type().IsRegular() {
			continue
		}
		if _, running := s.active[id]; running {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		done = append(done, finished{id: id, mod: info.ModTime()})
	}
	if len(done) < keepFinished {
		return nil
	}
	slices.SortFunc(done, func(a, b finished) int { return b.mod.Compare(a.mod) })
	for _, f := range done[keepFinished-1:] {
		if rmErr := os.Remove(s.path(f.id)); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return fmt.Errorf("remove old capture: %w", rmErr)
		}
	}
	return nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+fileSuffix)
}

// validID reports whether id has the shape [rand.Text] produces: idLen
// characters of the RFC 4648 base32 alphabet. Anything else — a path
// separator, a dot — never reaches the filesystem.
func validID(id string) bool {
	if len(id) != idLen {
		return false
	}
	for _, c := range id {
		if (c < 'A' || c > 'Z') && (c < '2' || c > '7') {
			return false
		}
	}
	return true
}
