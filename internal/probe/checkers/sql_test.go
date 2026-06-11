package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// newSQLDialer returns a modbusDialer (defined in modbus_test.go, same
// package) wired to succeed or fail — the SQL TCP path only calls Dial
// and Close, so the heavier modbusFakeConn is a fine substitute.
func newSQLDialer(err error) *modbusDialer {
	if err != nil {
		return &modbusDialer{err: err}
	}
	// Success path: provide a conn that Close() never errors.
	return &modbusDialer{conn: &modbusFakeConn{}}
}

// ---- Kind -----------------------------------------------------------------

func TestSQLChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewSQLChecker().Kind() != "sql" {
		t.Errorf("Kind() = %q; want %q", checkers.NewSQLChecker().Kind(), "sql")
	}
}

// ---- TCP path — postgres ---------------------------------------------------

func TestSQLChecker_Run_Postgres_Success(t *testing.T) {
	t.Parallel()

	params, _ := json.Marshal(checkers.SQLParams{Driver: "postgres"})
	c := checkers.NewSQLChecker().WithSQLDialer(newSQLDialer(nil))
	r := c.Run(context.Background(), probe.Probe{
		ID:     "p1",
		Kind:   probe.KindSQL,
		Target: "db.example.com",
		Params: params,
	})

	if !r.Success {
		t.Fatalf("Success = false; want true: %s", r.Error)
	}
	if r.Kind != probe.KindSQL {
		t.Errorf("Kind = %q; want %q", r.Kind, probe.KindSQL)
	}

	// Metadata must contain driver and addr.
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("Metadata not valid JSON: %v", err)
	}
	if meta["driver"] != "postgres" {
		t.Errorf("metadata.driver = %v; want %q", meta["driver"], "postgres")
	}
	if addr, ok := meta["addr"].(string); !ok || addr == "" {
		t.Errorf("metadata.addr missing or empty")
	}
}

func TestSQLChecker_Run_Postgres_DialError(t *testing.T) {
	t.Parallel()

	params, _ := json.Marshal(checkers.SQLParams{Driver: "postgres"})
	c := checkers.NewSQLChecker().WithSQLDialer(newSQLDialer(errors.New("connection refused")))
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: "down.example.com",
		Params: params,
	})

	if r.Success {
		t.Error("Success = true on dial error; want false")
	}
	if r.Error == "" {
		t.Error("Error field empty on dial failure")
	}
}

// ---- TCP path — MySQL default port ----------------------------------------

func TestSQLChecker_Run_MySQL_DefaultPort(t *testing.T) {
	t.Parallel()

	params, _ := json.Marshal(checkers.SQLParams{Driver: "mysql"})
	c := checkers.NewSQLChecker().WithSQLDialer(newSQLDialer(nil))
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: "mysql.example.com",
		Params: params,
	})

	if !r.Success {
		t.Fatalf("Success = false; want true: %s", r.Error)
	}
	var meta map[string]any
	_ = json.Unmarshal(r.Metadata, &meta)
	// addr should include the default MySQL port 3306.
	if addr, _ := meta["addr"].(string); addr == "" {
		t.Error("metadata.addr missing")
	}
}

// ---- Unknown / empty driver -----------------------------------------------

func TestSQLChecker_Run_UnknownDriver(t *testing.T) {
	t.Parallel()

	params, _ := json.Marshal(checkers.SQLParams{Driver: "cassandra"})
	c := checkers.NewSQLChecker().WithSQLDialer(newSQLDialer(nil))
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: "host.example.com",
		Params: params,
	})

	if r.Success {
		t.Error("Success = true for unknown driver; want false")
	}
	if r.Error == "" {
		t.Error("Error field empty for unknown driver")
	}
}

func TestSQLChecker_Run_EmptyDriver(t *testing.T) {
	t.Parallel()

	// No params at all — driver defaults to empty string.
	c := checkers.NewSQLChecker().WithSQLDialer(newSQLDialer(nil))
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: "host.example.com",
	})

	if r.Success {
		t.Error("Success = true with no driver specified; want false")
	}
	if r.Error == "" {
		t.Error("Error field empty for missing driver")
	}
}

// ---- SQLite path ----------------------------------------------------------

func TestSQLChecker_Run_SQLite_FileExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbFile := filepath.Join(dir, "test.db")
	if err := os.WriteFile(dbFile, []byte("SQLite format 3"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	params, _ := json.Marshal(checkers.SQLParams{Driver: "sqlite", Database: dbFile})
	// SQLite path does not use the dialer; pass nil-conn dialer to confirm it
	// is never invoked (a dial attempt would return an error and fail the test).
	c := checkers.NewSQLChecker()
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: dbFile,
		Params: params,
	})

	if !r.Success {
		t.Fatalf("Success = false; want true: %s", r.Error)
	}

	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("Metadata not valid JSON: %v", err)
	}
	if meta["server_version"] != "SQLite" {
		t.Errorf("metadata.server_version = %v; want %q", meta["server_version"], "SQLite")
	}
	if meta["path"] != dbFile {
		t.Errorf("metadata.path = %v; want %q", meta["path"], dbFile)
	}
}

func TestSQLChecker_Run_SQLite_FileNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missing := filepath.Join(dir, "nonexistent.db")

	params, _ := json.Marshal(checkers.SQLParams{Driver: "sqlite", Database: missing})
	c := checkers.NewSQLChecker()
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: missing,
		Params: params,
	})

	if r.Success {
		t.Error("Success = true for missing SQLite file; want false")
	}
	if r.Error == "" {
		t.Error("Error field empty for missing file")
	}
}

func TestSQLChecker_Run_SQLite_TargetFallback(t *testing.T) {
	t.Parallel()

	// When Database is empty, checker falls back to Probe.Target.
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "fallback.db")
	if err := os.WriteFile(dbFile, []byte("stub"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	params, _ := json.Marshal(checkers.SQLParams{Driver: "sqlite"}) // no Database field
	c := checkers.NewSQLChecker()
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: dbFile, // used as fallback path
		Params: params,
	})

	if !r.Success {
		t.Fatalf("Success = false on Target fallback; want true: %s", r.Error)
	}
}

func TestSQLChecker_Run_SQLite_DirectoryIsRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	params, _ := json.Marshal(checkers.SQLParams{Driver: "sqlite", Database: dir})
	c := checkers.NewSQLChecker()
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindSQL,
		Target: dir,
		Params: params,
	})

	if r.Success {
		t.Error("Success = true when path is a directory; want false")
	}
}
