// SPDX-License-Identifier: BUSL-1.1

// Package licensetest writes sealed activation state the way the foundation
// manager does, so tests can stand up the expired and forged licence files
// that no manager call produces. The file is sealed with the device
// fingerprint and the policy salt, both readable by any local process, which
// is why a forged file is a real input and worth testing against.
package licensetest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

	"github.com/MustardSeedNetworks/seed/internal/license"
)

// Path is where a manager rooted at dir reads its state.
func Path(dir string) string {
	return filepath.Join(dir, license.Policy().LicenseFileName)
}

// WriteState seals st into dir for the next manager rooted there to load.
func WriteState(t testing.TB, dir string, st license.ActivationState) {
	t.Helper()

	fp, fpErr := fnd.GenerateFingerprint()
	if fpErr != nil {
		t.Fatalf("fingerprint: %v", fpErr)
	}
	plain, marshalErr := json.Marshal(st)
	if marshalErr != nil {
		t.Fatalf("marshal state: %v", marshalErr)
	}

	key := sha256.Sum256([]byte(fp.Hash() + license.Policy().EncryptionSalt))
	block, blockErr := aes.NewCipher(key[:])
	if blockErr != nil {
		t.Fatalf("cipher: %v", blockErr)
	}
	gcm, gcmErr := cipher.NewGCM(block)
	if gcmErr != nil {
		t.Fatalf("gcm: %v", gcmErr)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, randErr := rand.Read(nonce); randErr != nil {
		t.Fatalf("nonce: %v", randErr)
	}
	sealed := base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, plain, nil))

	if writeErr := os.WriteFile(Path(dir), []byte(sealed), 0o600); writeErr != nil {
		t.Fatalf("write state: %v", writeErr)
	}
}

// PaidManager loads a manager over a paid licence for tier, sealed in a fresh
// directory and signed with an ephemeral key the manager is told to trust. The
// production key cannot sign in a test, and only a signed licence reaches the
// expiry and device checks; exp zero is perpetual.
func PaidManager(t testing.TB, tier license.Tier, exp time.Time, deviceHash string) *license.Manager {
	t.Helper()

	pub, priv, keyErr := ed25519.GenerateKey(rand.Reader)
	if keyErr != nil {
		t.Fatalf("generate key: %v", keyErr)
	}
	_, code, ok := license.Policy().FeaturesForTier(int(tier))
	if !ok {
		t.Fatalf("tier %s has no product code", tier)
	}
	payload := fnd.Payload{
		Version:  1,
		Product:  license.Policy().ProductName,
		Code:     code,
		Serial:   "TEST-" + t.Name(),
		Tier:     int(tier),
		IssuedAt: time.Now().Add(-time.Hour).Unix(),
	}
	if !exp.IsZero() {
		payload.ExpiresAt = exp.Unix()
	}
	raw, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		t.Fatalf("marshal payload: %v", marshalErr)
	}
	token := "MSN1." + base64.RawURLEncoding.EncodeToString(raw) + "." +
		base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, raw))

	dir := t.TempDir()
	WriteState(t, dir, license.ActivationState{
		LicenseKey:  token,
		DeviceHash:  deviceHash,
		ActivatedAt: time.Now().Add(-time.Hour),
	})
	mgr, mgrErr := fnd.NewManagerWithDir(fnd.NewVerifier(pub, license.Policy()), license.Policy(), dir)
	if mgrErr != nil {
		t.Fatalf("license manager: %v", mgrErr)
	}
	if status := mgr.LoadStatus(); status != fnd.StatusLoaded {
		t.Fatalf("LoadStatus = %s, want loaded (%v)", status, mgr.LoadError())
	}
	return mgr
}
