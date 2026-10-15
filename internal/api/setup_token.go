package api

// The first-run setup token proves the caller can read the host's data
// directory, the same possession proof password recovery uses. It is never
// sent over HTTP: setup/status answers any network peer, so a token served
// there would let whoever reached the port first claim admin. The CSRF
// protection the token was first added for (#724, #758) holds as before,
// since a cross-site form cannot read the file either.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	// setupTokenLength is the length of setup tokens in bytes before encoding.
	setupTokenLength = 32

	// setupTokenFile is the data-directory file the token is written to.
	setupTokenFile = ".setup-token"
)

// SetupTokenManager issues the first-run setup token to a 0600 file and
// checks the token the operator copies from it. The token lives until setup
// completes; the setup-mode timeout (#891) bounds the window it is accepted in.
type SetupTokenManager struct {
	mu      sync.Mutex
	token   string
	dataDir string
}

// NewSetupTokenManager creates a manager that writes its token under dataDir.
func NewSetupTokenManager(dataDir string) *SetupTokenManager {
	return &SetupTokenManager{dataDir: dataDir}
}

// FilePath returns the path the token is written to.
func (m *SetupTokenManager) FilePath() string {
	return filepath.Join(m.dataDir, setupTokenFile)
}

// Issue generates a new token and writes it to FilePath, readable only by the
// daemon's user, replacing any earlier token.
func (m *SetupTokenManager) Issue() error {
	tokenBytes := make([]byte, setupTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.dataDir, 0o700); err != nil {
		return err
	}
	// Remove first so the 0600 mode applies even when an earlier file was
	// created with a looser one.
	path := m.FilePath()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	m.token = token
	return nil
}

// Validate reports whether provided matches the issued token, in constant
// time. It does not consume the token, so a password the policy rejects can
// be retried without fetching a new one.
func (m *SetupTokenManager) Validate(provided string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.token == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(m.token)) == 1
}

// Consume clears the token and deletes its file once setup has completed.
func (m *SetupTokenManager) Consume() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.token = ""
	if err := os.Remove(m.FilePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// issueSetupToken writes the token for this setup window and logs where the
// operator finds it. Without the file setup cannot complete, so a failure is
// logged as an error rather than ending startup: the rest of the daemon runs.
func (s *Server) issueSetupToken() {
	if err := s.setupToken.Issue(); err != nil {
		logging.GetLogger().Error("Failed to write the setup token; setup cannot complete",
			"event", "auth.setup.token_failed", "path", s.setupToken.FilePath(), "error", err)
		return
	}
	logging.GetLogger().Info("Setup pending: enter the setup token from this file in the setup wizard",
		"event", "auth.setup.token_issued", "path", s.setupToken.FilePath(),
		"expires_in_minutes", setupModeTimeoutMin)
}
