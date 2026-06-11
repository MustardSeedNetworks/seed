package checkers

// SQLChecker implements probe.Checker for Kind="sql".
//
// V1.0 scope — reachability only: for network-backed databases (MySQL,
// PostgreSQL, SQL Server, Oracle) the checker dials the database TCP port
// and reports Success on connect. Driver-level authentication, session
// setup, and query execution are out of scope for V1.0 because no SQL
// driver is imported in this module; adding one would require a dependency
// that is unjustified until a specific customer need arises.
//
// For SQLite (file-backed, no network) the checker verifies that the
// database file exists and is a regular file via os.Stat. No read or write
// is attempted — file presence is sufficient evidence of reachability.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultSQLTimeout is the per-attempt SQL probe timeout.
const defaultSQLTimeout = 10 * time.Second

// Default TCP ports by driver (IANA / vendor assignments).
const (
	defaultMySQLPort     = 3306
	defaultPostgresPort  = 5432
	defaultSQLServerPort = 1433
	defaultOraclePort    = 1521
)

// Recognised driver identifiers accepted in SQLParams.Driver.
const (
	sqlDriverMySQL     = "mysql"
	sqlDriverPostgres  = "postgres"
	sqlDriverSQLServer = "sqlserver"
	sqlDriverOracle    = "oracle"
	sqlDriverSQLite    = "sqlite"
)

// SQLParams is the kind-specific params shape. Driver is required.
// Port, Database, and TimeoutMs are optional and default as described.
type SQLParams struct {
	// Driver selects the database engine: "mysql", "postgres",
	// "sqlserver", "oracle", or "sqlite".
	Driver string `json:"driver,omitempty"`

	// Port overrides the well-known TCP port for the driver. When 0
	// the checker uses the driver default (mysql=3306, postgres=5432,
	// sqlserver=1433, oracle=1521). Unused for sqlite.
	Port int `json:"port,omitempty"`

	// Database names the database / file path. For sqlite the value is
	// used as the file path when non-empty, falling back to Probe.Target.
	Database string `json:"database,omitempty"`

	// TimeoutMs overrides the default 10 000 ms dial/stat timeout.
	TimeoutMs int `json:"timeout_ms,omitempty"`
}

// SQLChecker implements probe.Checker for Kind="sql" — TCP reachability
// for network databases; file-stat reachability for SQLite.
type SQLChecker struct {
	dialer PingDialer
}

// NewSQLChecker returns an SQLChecker wired to a real [net.Dialer].
func NewSQLChecker() *SQLChecker {
	return &SQLChecker{dialer: realPingDialer{}}
}

// WithSQLDialer swaps the dialer; used by tests.
func (c *SQLChecker) WithSQLDialer(d PingDialer) *SQLChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindSQL.
func (c *SQLChecker) Kind() string { return probe.KindSQL }

// RequiredCapabilities returns nil; TCP dial and file-stat need no
// special hardware capability.
func (c *SQLChecker) RequiredCapabilities() []string { return nil }

// Run probes the SQL endpoint. For SQLite it stat-checks the database
// file; for all other drivers it dials Target:Port over TCP.
func (c *SQLChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseSQLParams(p.Params)

	timeout := defaultSQLTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}

	driver := strings.ToLower(strings.TrimSpace(params.Driver))

	if driver == sqlDriverSQLite {
		return c.runSQLite(p, params, timeout)
	}

	return c.runNetwork(ctx, p, params, driver, timeout)
}

// runSQLite stat-checks the SQLite database file. It reports Success
// when the file exists and is a regular file (not a directory).
func (c *SQLChecker) runSQLite(p probe.Probe, params SQLParams, _ time.Duration) probe.Result {
	path := params.Database
	if path == "" {
		path = p.Target
	}

	start := time.Now()
	fi, err := os.Stat(path)
	latencyMs := float64(time.Since(start).Milliseconds())

	if err != nil {
		return sqlFailure(p, latencyMs, fmt.Sprintf("sqlite: cannot stat %q: %s", path, err.Error()))
	}
	if fi.IsDir() {
		return sqlFailure(p, latencyMs, fmt.Sprintf("sqlite: path %q is a directory, not a file", path))
	}

	meta, _ := json.Marshal(map[string]any{
		"driver":         sqlDriverSQLite,
		"path":           path,
		"size":           fi.Size(),
		"modified":       fi.ModTime().UTC().Format(time.RFC3339),
		"server_version": "SQLite",
	})
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// runNetwork dials the database TCP port and reports reachability.
func (c *SQLChecker) runNetwork(
	ctx context.Context,
	p probe.Probe,
	params SQLParams,
	driver string,
	timeout time.Duration,
) probe.Result {
	port, err := resolvePort(driver, params.Port)
	if err != nil {
		return sqlFailure(p, 0, err.Error())
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(p.Target, strconv.Itoa(port))
	start := time.Now()
	conn, dialErr := c.dialer.Dial(dialCtx, "tcp", addr)
	latencyMs := float64(time.Since(start).Milliseconds())

	if dialErr != nil {
		return sqlFailure(p, latencyMs, dialErr.Error())
	}
	_ = conn.Close()

	meta, _ := json.Marshal(map[string]any{
		"driver":    driver,
		metaKeyAddr: addr,
		"database":  params.Database,
	})
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// resolvePort returns the effective TCP port for the given driver,
// applying the well-known default when port is 0. It returns an error
// for unknown or empty driver names.
func resolvePort(driver string, port int) (int, error) {
	if port > 0 {
		return port, nil
	}
	switch driver {
	case sqlDriverMySQL:
		return defaultMySQLPort, nil
	case sqlDriverPostgres:
		return defaultPostgresPort, nil
	case sqlDriverSQLServer:
		return defaultSQLServerPort, nil
	case sqlDriverOracle:
		return defaultOraclePort, nil
	case "":
		return 0, errors.New("sql probe requires params.driver (mysql|postgres|sqlserver|oracle|sqlite)")
	default:
		return 0, fmt.Errorf("sql probe: unknown driver %q; want mysql|postgres|sqlserver|oracle|sqlite", driver)
	}
}

// sqlFailure builds a failed Result with the measured latency so far.
func sqlFailure(p probe.Probe, latencyMs float64, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   false,
		LatencyMs: latencyMs,
		Error:     msg,
	}
}

// parseSQLParams decodes the params JSON; returns zero on empty or
// unparseable input.
func parseSQLParams(raw json.RawMessage) SQLParams {
	if len(raw) == 0 {
		return SQLParams{}
	}
	var p SQLParams
	_ = json.Unmarshal(raw, &p)
	return p
}
