package config

// migrate.go moves an on-disk config forward when a persisted key is renamed or
// removed.
//
// Config.UnmarshalJSON rejects unknown fields on purpose: pre-v1, a stale or
// misspelled setting is an error rather than something silently ignored. That
// is right, and it has a consequence nobody had paid for — seed.service is
// Restart=on-failure, so renaming a key turns every existing install into a
// crash-loop the operator cannot fix from the UI, because the daemon never
// starts. Observed on the lab probe when #377's additional_subnets ->
// target_networks rename met a 0.212.11 config.
//
// This is a migration, not a compatibility alias. The old spelling is rewritten
// once and is not accepted afterwards, which is what "migrate every caller in
// the same change" asks for when the caller is a file on disk.
//
// Removal has the same consequence and needed the same treatment. A config
// written by v0.200.0 — a real released version — still carried `pipeline` and
// `server.https`, and the strict decode killed the daemon on both. A removed key
// is stripped and reported, never silently ignored: the operator is told the
// setting is gone and where its replacement lives. An unknown key that is NOT on
// the table is still fatal, so this does not weaken the typo-catching the strict
// decode exists for.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// renamedKey is one persisted key that changed spelling.
type renamedKey struct {
	// introducedIn is the ConfigVersion that first requires the new spelling;
	// a config already at or above it is left alone.
	introducedIn int
	// path is the chain of objects containing the key, outermost first.
	path []string
	from string
	to   string
}

// versionTargetNetworks is the ConfigVersion that renamed
// networkDiscovery.additional_subnets to target_networks (#377).
const versionTargetNetworks = 2

// removedKey is one persisted key the schema no longer has.
//
// Unlike a rename, this is not gated on ConfigVersion. A key the schema does not
// have is never valid at any version, and the version was not always bumped when
// one was dropped — gating on it would leave exactly the configs that need the
// migration unmigrated.
type removedKey struct {
	// path is the chain of objects containing the key, outermost first.
	path []string
	key  string
	// replacement tells the operator where the setting went, or that it is gone.
	// It is the whole value of reporting rather than silently dropping.
	replacement string
	// stripWhen decides, from the value on disk, whether this key can be
	// dropped. nil means always.
	//
	// The distinction it draws is between a key that is merely obsolete and one
	// that carries an instruction the product can no longer honour.
	// `server.https: true` is the former: HTTPS is what Seed does, so the
	// setting is satisfied and dropping it changes nothing. `server.https:
	// false` is the latter — the operator asked for plaintext, and quietly
	// serving TLS while pretending the key was noise would be worse than
	// refusing to start. That one stays fatal, and
	// TestLoadRejectsRemovedTLSServerSettings pins it.
	stripWhen func(value any) bool
}

// String renders the key the way it appears in the file.
func (r removedKey) String() string {
	return strings.Join(append(append([]string{}, r.path...), r.key), ".")
}

// removedKeys is every key the loader strips rather than choking on.
func removedKeys() []removedKey {
	return []removedKey{
		{
			key: "pipeline",
			replacement: "discovery pipeline settings now live under networkDiscovery " +
				"(options, timing, profiler) and snmp; re-apply them there",
		},
		{
			path:        []string{"server"},
			key:         "https",
			replacement: "Seed always serves HTTPS; there is nothing to switch on",
			stripWhen: func(value any) bool {
				enabled, ok := value.(bool)
				return ok && enabled
			},
		},
	}
}

// renamedKeys is every rename the loader knows how to apply, oldest first.
func renamedKeys() []renamedKey {
	return []renamedKey{
		{
			introducedIn: versionTargetNetworks,
			path:         []string{"networkDiscovery"},
			from:         "additional_subnets",
			to:           "target_networks",
		},
	}
}

// migrateJSON applies every rename and removal the config is behind, returning
// the rewritten document, the removed keys it stripped, and whether anything
// changed.
//
// It rewrites the operator's own JSON rather than re-marshalling a Config, so a
// five-line config stays five lines instead of gaining every default.
//
// The stripped keys come back rather than being logged here so the migration
// stays a pure function of the document; Load reports them.
func migrateJSON(data []byte) ([]byte, []removedKey, bool, error) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		// Not an object, or not JSON at all. The strict decode that follows
		// will report it far better than this can.
		return data, nil, false, nil //nolint:nilerr // the decoder owns this error
	}

	version := versionOf(document)
	changed := false
	for _, rename := range renamedKeys() {
		if version >= rename.introducedIn {
			continue
		}
		if applyRename(document, rename) {
			changed = true
		}
	}

	var stripped []removedKey
	for _, removed := range removedKeys() {
		if applyRemoval(document, removed) {
			stripped = append(stripped, removed)
			changed = true
		}
	}

	if !changed {
		return data, nil, false, nil
	}

	document["version"] = ConfigVersion
	migrated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, nil, false, fmt.Errorf("re-encode migrated config: %w", err)
	}

	return migrated, stripped, true, nil
}

// applyRemoval deletes one key from its containing object, reporting whether it
// was there to delete.
func applyRemoval(document map[string]any, removed removedKey) bool {
	container := document
	for _, segment := range removed.path {
		next, ok := container[segment].(map[string]any)
		if !ok {
			return false
		}
		container = next
	}

	value, present := container[removed.key]
	if !present {
		return false
	}
	if removed.stripWhen != nil && !removed.stripWhen(value) {
		// Left in place on purpose: the strict decode refuses it, and that
		// refusal is the point.
		return false
	}
	delete(container, removed.key)

	return true
}

// versionOf reads the document's version, treating absent or malformed as 0 —
// the same "unversioned" the loader already recognises.
func versionOf(document map[string]any) int {
	raw, ok := document["version"].(float64)
	if !ok {
		return 0
	}

	return int(raw)
}

// applyRename moves one key within its containing object, reporting whether it
// was there to move.
func applyRename(document map[string]any, rename renamedKey) bool {
	container := document
	for _, segment := range rename.path {
		next, ok := container[segment].(map[string]any)
		if !ok {
			return false
		}
		container = next
	}

	value, present := container[rename.from]
	if !present {
		return false
	}
	// A config carrying both spellings keeps the new one; the old is dropped
	// rather than allowed to win by ordering.
	if _, alreadyMigrated := container[rename.to]; !alreadyMigrated {
		container[rename.to] = value
	}
	delete(container, rename.from)

	return true
}
