package config

// migrate.go moves an on-disk config forward when a persisted key is renamed.
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

import (
	"encoding/json"
	"fmt"
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

// migrateJSON applies every rename the config is behind, returning the rewritten
// document and whether anything changed.
//
// It rewrites the operator's own JSON rather than re-marshalling a Config, so a
// five-line config stays five lines instead of gaining every default.
func migrateJSON(data []byte) ([]byte, bool, error) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		// Not an object, or not JSON at all. The strict decode that follows
		// will report it far better than this can.
		return data, false, nil //nolint:nilerr // the decoder owns this error
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
	if !changed {
		return data, false, nil
	}

	document["version"] = ConfigVersion
	migrated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("re-encode migrated config: %w", err)
	}

	return migrated, true, nil
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
