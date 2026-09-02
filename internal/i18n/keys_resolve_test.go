package i18n_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
)

// keyCall matches localizer.T("...") and TWithData("...", ...) with a literal
// key. Keys built at run time are out of reach of a static check and are not
// what this is for.
func keyCall() *regexp.Regexp {
	return regexp.MustCompile(`\.T(?:WithData)?\("([a-zA-Z0-9_.]+)"`)
}

// knownUnresolved is the debt this test found and does not fix.
//
// 57 of the API's 106 distinct literal keys have no message in either locale.
// Writing the copy is its own piece of work in two languages, tracked
// separately. The list is exact rather than a count so that fixing one is
// noticed here: an entry that starts resolving fails this test and has to be
// removed, which is what stops the list rotting into a permanent excuse.
func knownUnresolvedKeys() []string {
	return []string{
		"errors.api.validationFailed",
		"errors.config.backupNameRequired",
		"errors.config.failedToCreateBackup",
		"errors.config.failedToDeleteBackup",
		"errors.config.invalidBackupName",
		"errors.config.nameParamRequired",
		"errors.discovery.failedToApplyOptions",
		"errors.discovery.managerUnavailable",
		"errors.health.deviceDiscoveryNotAvailable",
		"errors.health.dnsNotAvailable",
		"errors.health.dnsSecurityNotAvailable",
		"errors.health.iperfInvalidAction",
		"errors.health.iperfServerStartFailed",
		"errors.health.iperfServerStopFailed",
		"errors.health.iperfValidationFailed",
		"errors.health.noServersToScan",
		"errors.health.scanFailed",
		"errors.health.scanInProgress",
		"errors.health.speedtestInProgress",
		"errors.health.speedtestNotAvailable",
		"errors.logs.notInitialized",
		"errors.methodNotAllowed",
		"errors.profile.activeNotFound",
		"errors.profile.cannotDeleteActive",
		"errors.profile.cannotDeleteDefault",
		"errors.profile.createFailed",
		"errors.profile.dbNotAvailable",
		"errors.profile.deleteFailed",
		"errors.profile.duplicateFailed",
		"errors.profile.getActiveFailed",
		"errors.profile.getDefaultFailed",
		"errors.profile.getFailed",
		"errors.profile.idRequired",
		"errors.profile.listFailed",
		"errors.profile.nameExists",
		"errors.profile.nameRequired",
		"errors.profile.noActiveOrDefault",
		"errors.profile.notFound",
		"errors.profile.setActiveFailed",
		"errors.profile.updateFailed",
		"errors.security.failedToEncryptAuth",
		"errors.security.failedToEncryptPriv",
		"errors.security.gatewayTesterUnavailable",
		"errors.security.invalidAction",
		"errors.security.nullOriginForbidden",
		"errors.security.panicRecovered",
		"errors.settings.loadFailed",
		"errors.settings.saveFailed",
		"errors.tools.failedToCreateProber",
		"errors.tools.failedToCreateScanner",
		"errors.tools.invalidTarget",
		"errors.tools.ipRequired",
		"errors.tools.portRequired",
		"errors.vuln.invalidIp",
		"errors.vuln.missingIpParam",
		"errors.vuln.scannerNotEnabled",
		"validation.port.invalidRange",
	}
}

// TestEveryLiteralKeyResolves is the backend counterpart of the UI's
// missing-key detector.
//
// Localizer.T returns the key itself when lookup fails, so an unresolvable key
// is not an error — it is the raw string "errors.netif.invalidMode" arriving at
// the client as the user-facing message. Every call site in handlers_network.go
// did exactly that: the namespace is `errors.network.`, not `errors.netif.`,
// and nothing failed (#50).
func TestEveryLiteralKeyResolves(t *testing.T) {
	t.Parallel()

	localizer := i18n.NewLocalizer("en")
	seen := literalKeys(t)
	if len(seen) == 0 {
		t.Fatal("found no translation calls at all; the matcher is broken")
	}

	var stillBroken []string
	for key, path := range seen {
		if localizer.T(key) != key {
			continue
		}
		if slices.Contains(knownUnresolvedKeys(), key) {
			continue
		}
		stillBroken = append(stillBroken, path+": "+key)
	}
	slices.Sort(stillBroken)
	for _, entry := range stillBroken {
		t.Errorf("%s does not resolve — it would reach the client verbatim", entry)
	}

	// An entry that now resolves has been fixed and must leave the list, or
	// the list stops describing anything.
	for _, key := range knownUnresolvedKeys() {
		if _, used := seen[key]; !used {
			t.Errorf("%q is no longer called anywhere; drop it from knownUnresolved", key)

			continue
		}
		if localizer.T(key) != key {
			t.Errorf("%q resolves now; drop it from knownUnresolved", key)
		}
	}
}

// literalKeys collects every literal translation key in the tree, mapped to the
// first file it was seen in.
func literalKeys(t *testing.T) map[string]string {
	t.Helper()

	found := map[string]string{}
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range keyCall().FindAllStringSubmatch(string(source), -1) {
			if _, ok := found[match[1]]; !ok {
				found[match[1]] = path
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return found
}
