package database_test

// table_writers_test.go: every table the migrations leave behind has a writer
// in production Go code (P-A6, #3029). An empty table implies a capability
// Seed does not have, and every reader of it serves nothing (#2327, #2628).
// A feature adds its table in the same change as the code that writes it.

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestEveryTableHasAWriter(t *testing.T) {
	t.Parallel()

	db, cleanup := testDB(t)
	defer cleanup()

	rows, err := db.Query(context.Background(),
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var tables []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			t.Fatalf("scan table name: %v", scanErr)
		}
		if !schemaBookkeepingTables()[name] {
			tables = append(tables, name)
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("list tables: %v", rowsErr)
	}
	if len(tables) == 0 {
		t.Fatal("migrated schema has no tables")
	}

	source := productionGoSource(t, os.DirFS(filepath.Join("..", "..")))
	var missing []string
	for _, table := range tables {
		write := regexp.MustCompile(`(?i)\b(INSERT\s+(OR\s+\w+\s+)?INTO|REPLACE\s+INTO|UPDATE)\s+` +
			regexp.QuoteMeta(table) + `\b`)
		if !write.MatchString(source) {
			missing = append(missing, table)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("tables with no INSERT/REPLACE/UPDATE in production code: %s\n"+
			"Ship the producer with the table, or drop the table in a migration.",
			strings.Join(missing, ", "))
	}
}

// productionGoSource concatenates every non-test Go file in the module.
func productionGoSource(t *testing.T, module fs.FS) string {
	t.Helper()
	var b strings.Builder
	err := fs.WalkDir(module, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			// Hidden directories hold no module source, and CI points
			// GOMODCACHE at <repo>/.cache: walking it scanned every
			// dependency once per table and timed the package out.
			if path != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			switch d.Name() {
			case "node_modules", "testdata":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := fs.ReadFile(module, path)
		if readErr != nil {
			return readErr
		}
		b.Write(data)
		b.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatalf("walk module source: %v", err)
	}
	return b.String()
}
