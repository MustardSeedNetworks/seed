package api_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// apiDatabaseImporters is how many production files in internal/api still
// import internal/database directly (seed#2750, ADR-0020). Persistence belongs
// behind an internal/app use-case; each feature moved there lowers this number,
// and it only goes down. At zero, internal/app is the database's only importer
// outside cmd/seed.
const apiDatabaseImporters = 9

const databaseImportPath = "github.com/MustardSeedNetworks/seed/internal/database"

func TestAPIDatabaseImportsOnlyRatchetDown(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var importers []string
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, parseErr := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		require.NoError(t, parseErr)
		if slices.ContainsFunc(f.Imports, func(spec *ast.ImportSpec) bool {
			path, _ := strconv.Unquote(spec.Path.Value)
			return path == databaseImportPath
		}) {
			importers = append(importers, name)
		}
	}

	require.LessOrEqual(t, len(importers), apiDatabaseImporters,
		"a new internal/api file imports internal/database; put the query behind an internal/app use-case instead: %v",
		importers)
	require.Len(t, importers, apiDatabaseImporters,
		"fewer internal/api files import internal/database than the ratchet allows; lower apiDatabaseImporters to %d",
		len(importers))
}
