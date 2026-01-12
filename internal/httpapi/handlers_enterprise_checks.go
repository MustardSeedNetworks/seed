package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

// SQL driver constants.
const (
	DriverMySQL     = "mysql"
	DriverPostgres  = "postgres"
	DriverSQLServer = "sqlserver"
	DriverOracle    = "oracle"
	DriverSQLite    = "sqlite"
)

// Default ports for SQL databases.
const (
	DefaultMySQLPort     = 3306
	DefaultPostgresPort  = 5432
	DefaultSQLServerPort = 1433
	DefaultOraclePort    = 1521
)

// LDAP protocol constants.
const (
	DefaultLDAPPort  = 389
	DefaultLDAPSPort = 636
	LDAPTimeout      = 10 * time.Second
)

// File share protocol constants.
const (
	ProtocolSMB = "smb"
	ProtocolNFS = "nfs"

	DefaultSMBPort = 445
	DefaultNFSPort = 2049

	// DefaultTestFileSizeMB is the default size for performance test files.
	DefaultTestFileSizeMB = 10

	// BytesPerMB is the number of bytes in a megabyte.
	BytesPerMB = 1024 * 1024

	// fileShareTimeout is the timeout for file share connections.
	fileShareTimeout = 10 * time.Second
)

// SQL testing constants.
const (
	// sqlConnectTimeout is the timeout for SQL TCP connections.
	sqlConnectTimeout = 10 * time.Second

	// sqlMaxVersionLen is the maximum length for SQL version strings.
	sqlMaxVersionLen = 100

	// defaultSQLTestQuery is the default query for testing SQL connectivity.
	defaultSQLTestQuery = "SELECT 1"
)

// SQLTestResult contains the result of a SQL database health check.
type SQLTestResult struct {
	Name          string  `json:"name"`
	Driver        string  `json:"driver"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	Database      string  `json:"database"`
	Success       bool    `json:"success"`
	ConnectTimeMs float64 `json:"connectTimeMs"`
	QueryTimeMs   float64 `json:"queryTimeMs,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	ServerVersion string  `json:"serverVersion,omitempty"`
	Error         string  `json:"error,omitempty"`
	Timestamp     string  `json:"timestamp"`
}

// FileShareTestResult contains the result of a file share health check.
type FileShareTestResult struct {
	Name           string  `json:"name"`
	Protocol       string  `json:"protocol"`
	Host           string  `json:"host"`
	Share          string  `json:"share"`
	Success        bool    `json:"success"`
	ConnectTimeMs  float64 `json:"connectTimeMs"`
	ReadSpeedMBps  float64 `json:"readSpeedMBps,omitempty"`
	WriteSpeedMBps float64 `json:"writeSpeedMBps,omitempty"`
	ReadLatencyMs  float64 `json:"readLatencyMs,omitempty"`
	WriteLatencyMs float64 `json:"writeLatencyMs,omitempty"`
	TestFileSizeMB int     `json:"testFileSizeMB,omitempty"`
	TotalTimeMs    float64 `json:"totalTimeMs"`
	Error          string  `json:"error,omitempty"`
	Timestamp      string  `json:"timestamp"`
}

// LDAPTestResult contains the result of an LDAP health check.
type LDAPTestResult struct {
	Name          string  `json:"name"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	UseTLS        bool    `json:"useTls"`
	Success       bool    `json:"success"`
	ConnectTimeMs float64 `json:"connectTimeMs"`
	BindTimeMs    float64 `json:"bindTimeMs,omitempty"`
	SearchTimeMs  float64 `json:"searchTimeMs,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	EntriesFound  int     `json:"entriesFound,omitempty"`
	ServerInfo    string  `json:"serverInfo,omitempty"`
	Error         string  `json:"error,omitempty"`
	Timestamp     string  `json:"timestamp"`
}

// testSQLEndpoint tests a SQL database endpoint.
//
//nolint:funlen // SQL connection testing requires multiple validation steps: TCP, DB open, ping, query, version.
func (s *Server) testSQLEndpoint(ctx context.Context, endpoint config.SQLEndpoint) SQLTestResult {
	result := SQLTestResult{
		Name:      endpoint.Name,
		Driver:    endpoint.Driver,
		Host:      endpoint.Host,
		Port:      endpoint.Port,
		Database:  endpoint.Database,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Set default port if not specified
	if result.Port == 0 {
		switch endpoint.Driver {
		case DriverMySQL:
			result.Port = DefaultMySQLPort
		case DriverPostgres:
			result.Port = DefaultPostgresPort
		case DriverSQLServer:
			result.Port = DefaultSQLServerPort
		case DriverOracle:
			result.Port = DefaultOraclePort
		}
	}

	connectStart := time.Now()

	// Build connection string based on driver
	dsn, err := buildSQLDSN(endpoint, result.Port)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid configuration: %v", err)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// For SQLite, handle differently
	if endpoint.Driver == DriverSQLite {
		return s.testSQLiteEndpoint(ctx, endpoint, result, connectStart)
	}

	// Test TCP connectivity first
	addr := fmt.Sprintf("%s:%d", endpoint.Host, result.Port)
	dialer := net.Dialer{Timeout: sqlConnectTimeout}
	conn, connErr := dialer.DialContext(ctx, "tcp", addr)
	if connErr != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", connErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}
	_ = conn.Close()
	result.ConnectTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Open database connection
	db, openErr := sql.Open(endpoint.Driver, dsn)
	if openErr != nil {
		result.Error = fmt.Sprintf("Failed to open database: %v", openErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}
	defer func() { _ = db.Close() }()

	// Set connection timeout
	db.SetConnMaxLifetime(sqlConnectTimeout)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Test connection with ping
	pingCtx, cancel := context.WithTimeout(ctx, sqlConnectTimeout)
	defer cancel()

	queryStart := time.Now()
	if pingErr := db.PingContext(pingCtx); pingErr != nil {
		result.Error = fmt.Sprintf("Ping failed: %v", pingErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Execute test query if provided
	testQuery := endpoint.TestQuery
	if testQuery == "" {
		testQuery = getDefaultTestQuery(endpoint.Driver)
	}

	var queryResult string
	queryErr := db.QueryRowContext(pingCtx, testQuery).Scan(&queryResult)
	if queryErr != nil && queryErr != sql.ErrNoRows {
		result.Error = fmt.Sprintf("Query failed: %v", queryErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	result.QueryTimeMs = float64(time.Since(queryStart).Milliseconds())
	result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Try to get server version
	result.ServerVersion = getSQLServerVersion(db, endpoint.Driver)
	result.Success = true

	return result
}

// testSQLiteEndpoint tests a SQLite database file.
func (s *Server) testSQLiteEndpoint(
	_ context.Context,
	endpoint config.SQLEndpoint,
	result SQLTestResult,
	connectStart time.Time,
) SQLTestResult {
	// For SQLite, just check if the file exists and is readable
	dbPath := endpoint.Host // Host field contains the file path for SQLite
	if endpoint.Database != "" {
		dbPath = endpoint.Database
	}

	info, statErr := os.Stat(dbPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			result.Error = "Database file does not exist"
		} else {
			result.Error = fmt.Sprintf("Cannot access database: %v", statErr)
		}
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	if info.IsDir() {
		result.Error = "Path is a directory, not a database file"
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	result.ConnectTimeMs = float64(time.Since(connectStart).Milliseconds())
	result.TotalTimeMs = result.ConnectTimeMs
	result.Success = true
	result.ServerVersion = "SQLite"

	return result
}

// buildSQLDSN builds a connection string for the specified driver.
func buildSQLDSN(endpoint config.SQLEndpoint, port int) (string, error) {
	switch endpoint.Driver {
	case DriverMySQL:
		// user:password@tcp(host:port)/dbname?param=value
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			endpoint.Username, endpoint.Password,
			endpoint.Host, port, endpoint.Database)
		if endpoint.SSLMode != "" {
			dsn += "?tls=" + endpoint.SSLMode
		}
		return dsn, nil

	case DriverPostgres:
		// postgres://user:password@host:port/dbname?sslmode=disable
		hostPort := net.JoinHostPort(endpoint.Host, strconv.Itoa(port))
		dsn := fmt.Sprintf("postgres://%s:%s@%s/%s",
			endpoint.Username, endpoint.Password,
			hostPort, endpoint.Database)
		if endpoint.SSLMode != "" {
			dsn += "?sslmode=" + endpoint.SSLMode
		}
		return dsn, nil

	case DriverSQLServer:
		// sqlserver://user:password@host:port?database=dbname
		hostPort := net.JoinHostPort(endpoint.Host, strconv.Itoa(port))
		dsn := fmt.Sprintf("sqlserver://%s:%s@%s?database=%s",
			endpoint.Username, endpoint.Password,
			hostPort, endpoint.Database)
		return dsn, nil

	case DriverOracle:
		// oracle://user:password@host:port/service_name
		hostPort := net.JoinHostPort(endpoint.Host, strconv.Itoa(port))
		dsn := fmt.Sprintf("oracle://%s:%s@%s/%s",
			endpoint.Username, endpoint.Password,
			hostPort, endpoint.Database)
		return dsn, nil

	case DriverSQLite:
		return endpoint.Database, nil

	default:
		return "", fmt.Errorf("unsupported driver: %s", endpoint.Driver)
	}
}

// getDefaultTestQuery returns a simple test query for the driver.
func getDefaultTestQuery(driver string) string {
	// Oracle requires FROM DUAL for SELECT; others use standard query.
	if driver == DriverOracle {
		return "SELECT 1 FROM DUAL"
	}
	return defaultSQLTestQuery
}

// getSQLServerVersion attempts to retrieve the server version.
func getSQLServerVersion(db *sql.DB, driver string) string {
	var versionQuery string
	switch driver {
	case DriverMySQL:
		versionQuery = "SELECT VERSION()"
	case DriverPostgres:
		versionQuery = "SELECT version()"
	case DriverSQLServer:
		versionQuery = "SELECT @@VERSION"
	case DriverOracle:
		versionQuery = "SELECT BANNER FROM V$VERSION WHERE ROWNUM = 1"
	default:
		return ""
	}

	var version string
	_ = db.QueryRow(versionQuery).Scan(&version) //nolint:noctx // Simple query without context
	// Truncate long version strings
	if len(version) > sqlMaxVersionLen {
		version = version[:sqlMaxVersionLen] + "..."
	}
	return version
}

// testFileShareEndpoint tests a file share endpoint with optional performance testing.
//
//nolint:gocognit // File share testing has inherent complexity: protocol detection, TCP test, and optional performance tests.
func (s *Server) testFileShareEndpoint(
	ctx context.Context,
	endpoint config.FileShareEndpoint,
) FileShareTestResult {
	result := FileShareTestResult{
		Name:      endpoint.Name,
		Protocol:  endpoint.Protocol,
		Host:      endpoint.Host,
		Share:     endpoint.Share,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	connectStart := time.Now()

	// Determine port based on protocol
	var port int
	switch strings.ToLower(endpoint.Protocol) {
	case ProtocolSMB:
		port = DefaultSMBPort
	case ProtocolNFS:
		port = DefaultNFSPort
	default:
		result.Error = fmt.Sprintf("Unsupported protocol: %s", endpoint.Protocol)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Test TCP connectivity
	addr := fmt.Sprintf("%s:%d", endpoint.Host, port)
	dialer := net.Dialer{Timeout: fileShareTimeout}
	conn, connErr := dialer.DialContext(ctx, "tcp", addr)
	if connErr != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", connErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}
	_ = conn.Close()

	result.ConnectTimeMs = float64(time.Since(connectStart).Milliseconds())

	// If performance testing is requested, run actual file operations
	//nolint:nestif // Performance testing requires sequential operations with error handling.
	if endpoint.TestReadPerformance || endpoint.TestWritePerformance {
		testSize := endpoint.TestFileSizeMB
		if testSize <= 0 {
			testSize = DefaultTestFileSizeMB
		}
		result.TestFileSizeMB = testSize

		// Construct the share path
		sharePath := buildSharePath(endpoint)

		if endpoint.TestWritePerformance {
			writeSpeed, writeLatency, writeErr := s.performWriteTest(ctx, sharePath, testSize)
			if writeErr != nil {
				result.Error = fmt.Sprintf("Write test failed: %v", writeErr)
			} else {
				result.WriteSpeedMBps = writeSpeed
				result.WriteLatencyMs = writeLatency
			}
		}

		if endpoint.TestReadPerformance && result.Error == "" {
			readSpeed, readLatency, readErr := s.performReadTest(ctx, sharePath, testSize)
			if readErr != nil {
				if result.Error == "" {
					result.Error = fmt.Sprintf("Read test failed: %v", readErr)
				}
			} else {
				result.ReadSpeedMBps = readSpeed
				result.ReadLatencyMs = readLatency
			}
		}
	}

	result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())

	// Success if we connected (and performance tests passed if enabled)
	if result.Error == "" {
		result.Success = true
	}

	return result
}

// buildSharePath constructs the full path to the share.
func buildSharePath(endpoint config.FileShareEndpoint) string {
	var path string
	switch strings.ToLower(endpoint.Protocol) {
	case ProtocolSMB:
		// SMB path: //host/share/path
		path = fmt.Sprintf("//%s/%s", endpoint.Host, endpoint.Share)
	case ProtocolNFS:
		// NFS path: host:/export/path
		path = fmt.Sprintf("%s:/%s", endpoint.Host, endpoint.Share)
	default:
		path = endpoint.Share
	}

	if endpoint.Path != "" {
		path = filepath.Join(path, endpoint.Path)
	}

	return path
}

// performWriteTest performs a write performance test.
//
//nolint:nonamedreturns // Named returns clarify the multiple return values for performance metrics.
func (s *Server) performWriteTest(
	_ context.Context,
	sharePath string,
	sizeMB int,
) (speedMBps, latencyMs float64, err error) {
	// Generate random test data
	testData := make([]byte, sizeMB*BytesPerMB)
	if _, randErr := rand.Read(testData); randErr != nil {
		return 0, 0, fmt.Errorf("failed to generate test data: %w", randErr)
	}

	// Create temp file for writing
	testFileName := fmt.Sprintf(".seed_perf_test_%d.tmp", time.Now().UnixNano())
	testPath := filepath.Join(sharePath, testFileName)

	// Measure write performance
	writeStart := time.Now()
	file, createErr := os.Create(testPath)
	if createErr != nil {
		return 0, 0, fmt.Errorf("failed to create test file: %w", createErr)
	}

	latencyMs = float64(time.Since(writeStart).Milliseconds())

	written, writeErr := file.Write(testData)
	if writeErr != nil {
		_ = file.Close()
		_ = os.Remove(testPath)
		return 0, 0, fmt.Errorf("failed to write test data: %w", writeErr)
	}
	_ = file.Close()

	writeDuration := time.Since(writeStart)
	speedMBps = float64(written) / BytesPerMB / writeDuration.Seconds()

	// Clean up the test file
	_ = os.Remove(testPath)

	return speedMBps, latencyMs, nil
}

// performReadTest performs a read performance test.
//
//nolint:nonamedreturns // Named returns clarify the multiple return values for performance metrics.
func (s *Server) performReadTest(
	_ context.Context,
	sharePath string,
	sizeMB int,
) (speedMBps, latencyMs float64, err error) {
	// First, create a test file to read
	testData := make([]byte, sizeMB*BytesPerMB)
	if _, randErr := rand.Read(testData); randErr != nil {
		return 0, 0, fmt.Errorf("failed to generate test data: %w", randErr)
	}

	testFileName := fmt.Sprintf(".seed_perf_test_%d.tmp", time.Now().UnixNano())
	testPath := filepath.Join(sharePath, testFileName)

	// Write the test file first
	if writeErr := os.WriteFile(testPath, testData, 0o600); writeErr != nil {
		return 0, 0, fmt.Errorf("failed to create test file: %w", writeErr)
	}
	defer func() { _ = os.Remove(testPath) }()

	// Measure read performance
	readStart := time.Now()
	file, openErr := os.Open(testPath)
	if openErr != nil {
		return 0, 0, fmt.Errorf("failed to open test file: %w", openErr)
	}

	latencyMs = float64(time.Since(readStart).Milliseconds())

	readData, readErr := io.ReadAll(file)
	_ = file.Close()
	if readErr != nil {
		return 0, 0, fmt.Errorf("failed to read test data: %w", readErr)
	}

	readDuration := time.Since(readStart)
	speedMBps = float64(len(readData)) / BytesPerMB / readDuration.Seconds()

	return speedMBps, latencyMs, nil
}

// testLDAPEndpoint tests an LDAP/Active Directory endpoint.
func (s *Server) testLDAPEndpoint(
	ctx context.Context,
	endpoint config.LDAPEndpoint,
) LDAPTestResult {
	result := LDAPTestResult{
		Name:      endpoint.Name,
		Host:      endpoint.Host,
		Port:      endpoint.Port,
		UseTLS:    endpoint.UseTLS,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Set default port
	if result.Port == 0 {
		if endpoint.UseTLS {
			result.Port = DefaultLDAPSPort
		} else {
			result.Port = DefaultLDAPPort
		}
	}

	connectStart := time.Now()

	// Test TCP connectivity
	addr := fmt.Sprintf("%s:%d", endpoint.Host, result.Port)
	dialer := net.Dialer{Timeout: LDAPTimeout}

	var conn net.Conn
	var connErr error

	if endpoint.UseTLS {
		// LDAPS - TLS from the start
		tlsConfig := &tls.Config{
			ServerName:         endpoint.Host,
			InsecureSkipVerify: false,            // Validate certificates
			MinVersion:         tls.VersionTLS12, // Minimum TLS 1.2 for security
		}
		tlsDialer := tls.Dialer{
			NetDialer: &dialer,
			Config:    tlsConfig,
		}
		conn, connErr = tlsDialer.DialContext(ctx, "tcp", addr)
	} else {
		conn, connErr = dialer.DialContext(ctx, "tcp", addr)
	}

	if connErr != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", connErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}
	defer func() { _ = conn.Close() }()

	result.ConnectTimeMs = float64(time.Since(connectStart).Milliseconds())

	// If StartTLS is requested (and not already using LDAPS)
	if endpoint.StartTLS && !endpoint.UseTLS {
		// In a full implementation, we would send the StartTLS extended operation
		// For now, we just note that StartTLS would be used
		result.ServerInfo = "StartTLS supported (not verified)"
	}

	// Set deadline for LDAP operations
	if deadlineErr := conn.SetDeadline(time.Now().Add(LDAPTimeout)); deadlineErr != nil {
		result.Error = fmt.Sprintf("Failed to set deadline: %v", deadlineErr)
		result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
		return result
	}

	// Perform anonymous bind or authenticated bind
	bindStart := time.Now()
	if endpoint.BindDN != "" {
		// Would perform authenticated bind here
		// For TCP-level testing, we verify the connection is responsive
		result.ServerInfo = "Authenticated bind configured"
	} else {
		result.ServerInfo = "Anonymous bind"
	}
	result.BindTimeMs = float64(time.Since(bindStart).Milliseconds())

	// If search filter is configured, note it (actual LDAP search requires ldap library)
	if endpoint.SearchFilter != "" {
		searchStart := time.Now()
		result.SearchTimeMs = float64(time.Since(searchStart).Milliseconds())
		result.ServerInfo += fmt.Sprintf("; Search filter: %s", endpoint.SearchFilter)
	}

	result.TotalTimeMs = float64(time.Since(connectStart).Milliseconds())
	result.Success = true

	return result
}

// EnterpriseCheckResults contains results from all enterprise protocol checks.
type EnterpriseCheckResults struct {
	SQLResults       []SQLTestResult       `json:"sqlResults,omitempty"`
	FileShareResults []FileShareTestResult `json:"fileShareResults,omitempty"`
	LDAPResults      []LDAPTestResult      `json:"ldapResults,omitempty"`
}

// RunEnterpriseChecks runs all configured enterprise protocol health checks.
func (s *Server) RunEnterpriseChecks(ctx context.Context) *EnterpriseCheckResults {
	cfg := s.config
	results := &EnterpriseCheckResults{}

	// Run SQL checks
	for _, endpoint := range cfg.HealthChecks.SQLEndpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testSQLEndpoint(ctx, endpoint)
		results.SQLResults = append(results.SQLResults, result)
	}

	// Run FileShare checks
	for _, endpoint := range cfg.HealthChecks.FileShareEndpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testFileShareEndpoint(ctx, endpoint)
		results.FileShareResults = append(results.FileShareResults, result)
	}

	// Run LDAP checks
	for _, endpoint := range cfg.HealthChecks.LDAPEndpoints {
		if !endpoint.Enabled {
			continue
		}
		result := s.testLDAPEndpoint(ctx, endpoint)
		results.LDAPResults = append(results.LDAPResults, result)
	}

	return results
}
