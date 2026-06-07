package config

// keyring.go implements the credential Data-Encryption-Key (DEK) keyring
// described in ADR-0015. It separates SNMP-credential encryption from
// Auth.JWTSecret: the DEK is random key material persisted in a dedicated
// key file (or supplied via SEED_CREDENTIAL_KEY), per-version AES-256 keys are
// derived with HKDF-SHA256, and ciphertext carries an explicit version tag
// ("enc:v<N>:...") so the key that produced it is always known and rotation is
// possible. The legacy unversioned "enc:..." (JWT-derived) format remains
// decryptable, read-only, for transparent migration.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// dekEnvVar supplies a base64-encoded 32-byte DEK master, taking
	// precedence over the on-disk key file. It is never written to disk.
	dekEnvVar = "SEED_CREDENTIAL_KEY"
	// dekFileName is the dedicated key-file name under the data dir.
	dekFileName = "credential.key"
	// dekInfoFmt is the HKDF domain-separation label per version.
	dekInfoFmt = "seed:snmp-credential-encryption:v%d"
	// dekKDFName records the KDF used for a keyring version.
	dekKDFName = "hkdf-sha256"

	dekMasterLen = 32 // AES-256 master key material
	dekSaltLen   = 16 // per-version HKDF salt
	dekKeyLen    = 32 // derived AES-256 key

	versionTag = "v" // the "v<N>" marker that follows encryptedPrefix
)

// ErrKeyringUnavailable is returned when credential encryption is attempted
// without an initialised keyring and no ephemeral fallback could be created.
var ErrKeyringUnavailable = errors.New("credential keyring unavailable")

// Keyring derives versioned AES-256 keys for credential encryption (ADR-0015).
// It is immutable after construction; EncryptValue/DecryptValue are safe for
// concurrent use and never touch Auth.JWTSecret.
type Keyring struct {
	master   []byte
	active   int
	salts    map[int][]byte
	path     string // "" => not persisted (env-only or ephemeral)
	envBound bool   // master came from SEED_CREDENTIAL_KEY (do not persist it)
}

// keyringVersion is one persisted keyring entry.
type keyringVersion struct {
	Version int    `json:"version"`
	Salt    string `json:"salt"` // base64 std
	KDF     string `json:"kdf"`
}

// keyringFile is the on-disk JSON layout of credential.key.
type keyringFile struct {
	Active   int              `json:"active"`
	Master   string           `json:"master,omitempty"` // omitted when env-supplied
	Versions []keyringVersion `json:"versions"`
}

// LoadOrCreateKeyring resolves the DEK master (env override first, then the
// key file under dir) and loads or creates the keyring, persisting a freshly
// generated key file with 0600 permissions when none exists.
func LoadOrCreateKeyring(dir string) (*Keyring, error) {
	envMaster, envSet, err := masterFromEnv()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, dekFileName)
	raw, readErr := os.ReadFile(path)
	switch {
	case readErr == nil:
		return loadKeyring(raw, path, envMaster, envSet)
	case errors.Is(readErr, os.ErrNotExist):
		return createKeyring(dir, path, envMaster, envSet)
	default:
		return nil, fmt.Errorf("read credential key file: %w", readErr)
	}
}

// newEphemeralKeyring builds an in-memory, non-persisted keyring with a random
// master and a single active version. It is the safety-net used when no keyring
// was initialised (e.g. tests); ciphertext it produces does not survive restart.
func newEphemeralKeyring() (*Keyring, error) {
	master := make([]byte, dekMasterLen)
	if _, err := io.ReadFull(rand.Reader, master); err != nil {
		return nil, fmt.Errorf("generate ephemeral DEK: %w", err)
	}
	salt := make([]byte, dekSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate ephemeral salt: %w", err)
	}
	return &Keyring{
		master: master,
		active: 1,
		salts:  map[int][]byte{1: salt},
	}, nil
}

func masterFromEnv() ([]byte, bool, error) {
	v := strings.TrimSpace(os.Getenv(dekEnvVar))
	if v == "" {
		return nil, false, nil
	}
	decoded, decErr := base64.StdEncoding.DecodeString(v)
	if decErr != nil {
		return nil, false, fmt.Errorf("%s: invalid base64: %w", dekEnvVar, decErr)
	}
	if len(decoded) != dekMasterLen {
		return nil, false, fmt.Errorf(
			"%s: must decode to %d bytes, got %d", dekEnvVar, dekMasterLen, len(decoded),
		)
	}
	return decoded, true, nil
}

func loadKeyring(raw []byte, path string, envMaster []byte, envSet bool) (*Keyring, error) {
	var file keyringFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse credential key file: %w", err)
	}
	if len(file.Versions) == 0 || file.Active < 1 {
		return nil, errors.New("credential key file is missing version metadata")
	}

	salts := make(map[int][]byte, len(file.Versions))
	for _, ver := range file.Versions {
		salt, err := base64.StdEncoding.DecodeString(ver.Salt)
		if err != nil {
			return nil, fmt.Errorf("decode salt for v%d: %w", ver.Version, err)
		}
		salts[ver.Version] = salt
	}
	if _, ok := salts[file.Active]; !ok {
		return nil, fmt.Errorf("active version v%d has no salt", file.Active)
	}

	master, err := resolveMaster(file.Master, envMaster, envSet)
	if err != nil {
		return nil, err
	}

	return &Keyring{
		master:   master,
		active:   file.Active,
		salts:    salts,
		path:     path,
		envBound: envSet,
	}, nil
}

func resolveMaster(fileMaster string, envMaster []byte, envSet bool) ([]byte, error) {
	if envSet {
		return envMaster, nil
	}
	if fileMaster == "" {
		return nil, errors.New(
			"credential key file has no master and " + dekEnvVar + " is not set",
		)
	}
	master, err := base64.StdEncoding.DecodeString(fileMaster)
	if err != nil {
		return nil, fmt.Errorf("decode credential master: %w", err)
	}
	if len(master) != dekMasterLen {
		return nil, fmt.Errorf("credential master must be %d bytes, got %d", dekMasterLen, len(master))
	}
	return master, nil
}

func createKeyring(dir, path string, envMaster []byte, envSet bool) (*Keyring, error) {
	master := envMaster
	if !envSet {
		master = make([]byte, dekMasterLen)
		if _, err := io.ReadFull(rand.Reader, master); err != nil {
			return nil, fmt.Errorf("generate DEK master: %w", err)
		}
	}

	salt := make([]byte, dekSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate DEK salt: %w", err)
	}

	kr := &Keyring{
		master:   master,
		active:   1,
		salts:    map[int][]byte{1: salt},
		path:     path,
		envBound: envSet,
	}
	if err := kr.persist(dir); err != nil {
		return nil, err
	}
	return kr, nil
}

// persist writes the keyring to its key file at 0600. The master is omitted
// when it is env-supplied so secret material is never written to disk.
func (kr *Keyring) persist(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	versions := make([]keyringVersion, 0, len(kr.salts))
	for ver, salt := range kr.salts {
		versions = append(versions, keyringVersion{
			Version: ver,
			Salt:    base64.StdEncoding.EncodeToString(salt),
			KDF:     dekKDFName,
		})
	}

	file := keyringFile{Active: kr.active, Versions: versions}
	if !kr.envBound {
		file.Master = base64.StdEncoding.EncodeToString(kr.master)
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credential keyring: %w", err)
	}
	if writeErr := os.WriteFile(kr.path, data, 0o600); writeErr != nil {
		return fmt.Errorf("write credential key file: %w", writeErr)
	}
	return nil
}

// deriveKey derives the AES-256 key for the given keyring version via
// HKDF-SHA256 over the random master, the per-version salt, and a versioned
// domain-separation label.
func (kr *Keyring) deriveKey(version int) ([]byte, error) {
	salt, ok := kr.salts[version]
	if !ok {
		return nil, fmt.Errorf("%w: unknown key version v%d", ErrInvalidCiphertext, version)
	}
	info := fmt.Sprintf(dekInfoFmt, version)
	key, err := hkdf.Key(sha256.New, kr.master, salt, info, dekKeyLen)
	if err != nil {
		return nil, fmt.Errorf("derive credential key: %w", err)
	}
	return key, nil
}

// EncryptValue encrypts plaintext with the active key version, producing
// "enc:v<active>:base64(nonce‖ciphertext‖tag)". Empty and already-encrypted
// inputs are returned unchanged (idempotent).
func (kr *Keyring) EncryptValue(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if IsEncrypted(plaintext) {
		return plaintext, nil
	}

	key, err := kr.deriveKey(kr.active)
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, nonceErr := io.ReadFull(rand.Reader, nonce); nonceErr != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", nonceErr)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return fmt.Sprintf("%s%s%d:%s", encryptedPrefix, versionTag, kr.active, encoded), nil
}

// DecryptValue decrypts a versioned ("enc:v<N>:...") credential.
func (kr *Keyring) DecryptValue(encrypted string) (string, error) {
	version, encoded, ok := parseVersioned(encrypted)
	if !ok {
		return "", fmt.Errorf("%w: not a versioned ciphertext", ErrInvalidCiphertext)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64: %w", ErrInvalidCiphertext, err)
	}

	key, err := kr.deriveKey(version)
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}
	nonce, body := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("%w: authentication failed: %w", ErrInvalidCiphertext, err)
	}
	return string(plaintext), nil
}

// newGCM builds an AES-256-GCM AEAD from a 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return gcm, nil
}

// parseVersioned splits "enc:v<N>:<base64>" into its version and payload. The
// final bool is false for the legacy unversioned "enc:..." format.
func parseVersioned(s string) (int, string, bool) {
	if !strings.HasPrefix(s, encryptedPrefix) {
		return 0, "", false
	}
	rest := s[len(encryptedPrefix):]
	if !strings.HasPrefix(rest, versionTag) {
		return 0, "", false
	}
	idx := strings.IndexByte(rest, ':')
	if idx <= len(versionTag) {
		return 0, "", false
	}
	n, err := strconv.Atoi(rest[len(versionTag):idx])
	if err != nil || n < 1 {
		return 0, "", false
	}
	return n, rest[idx+1:], true
}

// isVersionedCiphertext reports whether value uses the versioned DEK format.
func isVersionedCiphertext(value string) bool {
	_, _, ok := parseVersioned(value)
	return ok
}

// IsLegacyEncrypted reports whether value is an encrypted credential in the
// legacy unversioned ("enc:...", JWT-derived) format that must be migrated.
func IsLegacyEncrypted(value string) bool {
	return IsEncrypted(value) && !isVersionedCiphertext(value)
}
