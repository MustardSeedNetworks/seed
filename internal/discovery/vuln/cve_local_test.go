package vuln_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
)

// TestLocalProviderSearchByProductVersions pins seed#3061: a CVE that lists
// versions matches only those versions, and one that lists none matches every
// version of its product, alongside any exact match.
func TestLocalProviderSearchByProductVersions(t *testing.T) {
	const db = `{
  "version": "1",
  "cves": [
    {"id": "CVE-2099-0001", "vendor": "linux", "product": "linux-kernel", "versions": ["5.4"]},
    {"id": "CVE-2099-0002", "vendor": "linux", "product": "linux-kernel", "versions": ["5.4", "5.10"]},
    {"id": "CVE-2099-0003", "vendor": "linux", "product": "linux-kernel"},
    {"id": "CVE-2099-0004", "vendor": "openbsd", "product": "openssh", "versions": ["9.6"]}
  ]
}`
	path := filepath.Join(t.TempDir(), "cves.json")
	require.NoError(t, os.WriteFile(path, []byte(db), 0o600))
	provider, err := vuln.NewLocalProvider(path)
	require.NoError(t, err)

	tests := []struct {
		name                     string
		vendor, product, version string
		want                     []string
	}{
		{"listed version", "Linux", "Linux-Kernel", "5.4", []string{"CVE-2099-0001", "CVE-2099-0002", "CVE-2099-0003"}},
		{"second listed version", "linux", "linux-kernel", "5.10", []string{"CVE-2099-0002", "CVE-2099-0003"}},
		{"unlisted version", "linux", "linux-kernel", "6.1", []string{"CVE-2099-0003"}},
		{"product with only versioned CVEs", "openbsd", "openssh", "9.7", nil},
		{"unknown product", "cisco", "ios", "15.2", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns, searchErr := provider.SearchByProduct(t.Context(), tt.vendor, tt.product, tt.version)
			require.NoError(t, searchErr)
			var got []string
			for _, v := range vulns {
				got = append(got, v.CVEID)
			}
			require.ElementsMatch(t, tt.want, got)
		})
	}
}
