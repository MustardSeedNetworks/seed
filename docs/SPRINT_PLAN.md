# LuminetIQ - Detailed Sprint Plan
**Code Review Remediation Plan**

**Created:** 2025-12-14
**Sprint Duration:** 2 weeks per sprint
**Total Duration:** 8 weeks (4 sprints)

---

## Sprint Overview

| Sprint | Focus | Issues | Hours | Priority |
|--------|-------|--------|-------|----------|
| Sprint 1 | Security Critical | 5 CRITICAL + 4 HIGH | 36h | P0 |
| Sprint 2 | Concurrency & Stability | 5 CRITICAL + 3 HIGH | 34h | P0 |
| Sprint 3 | Data Integrity & Resources | 4 CRITICAL + 6 HIGH | 42h | P0 |
| Sprint 4 | Architecture & Tech Debt | 0 CRITICAL + 6 HIGH | 35h | P1 |

**Total Estimated Time:** 147 hours (~18 working days)

---

## 🔥 SPRINT 1: Security Critical (Week 1-2)

**Goal:** Eliminate immediate security vulnerabilities that block production deployment

**Duration:** 2 weeks
**Total Hours:** 36 hours
**Team Size:** 2 engineers recommended

### Issues in Sprint 1

#### CRITICAL Issues

##### #518: SNMP Credentials in Plaintext (12h) 🔴
**Priority:** P0 - BLOCKER
**Assignee:** Security-focused engineer

**Tasks:**
1. Implement AES-256-GCM encryption for credentials (4h)
   - Use `crypto/aes` and `crypto/cipher`
   - Key derivation from master password/config secret
   - Store encrypted credentials in config

2. Never return passwords in API responses (2h)
   - Modify GET `/api/snmp/settings` to return `*****` for passwords
   - Only accept passwords in PUT requests
   - Validate before storing

3. Redact from logs (2h)
   - Add password sanitization middleware
   - Scan all log statements for password fields
   - Add test for log sanitization

4. Migration script for existing configs (2h)
   - Detect plaintext passwords
   - Encrypt on first load
   - Backup original config

5. Documentation (2h)
   - Document encryption scheme
   - Add to security.md

**Test Requirements:**
```go
// Test encryption/decryption
func TestSNMPCredentialEncryption(t *testing.T) {
    original := "mySecretPassword"
    encrypted := encryptCredential(original, masterKey)
    assert.NotEqual(t, original, encrypted)

    decrypted := decryptCredential(encrypted, masterKey)
    assert.Equal(t, original, decrypted)
}

// Test API doesn't expose passwords
func TestSNMPSettingsNoPasswordInResponse(t *testing.T) {
    resp := getAPI("/api/snmp/settings")
    // Verify response doesn't contain actual password
    assert.Equal(t, "*****", resp.V3Credentials[0].AuthPassword)
}

// Test log sanitization
func TestPasswordNotLoggedOnError(t *testing.T) {
    // Trigger error that might log request
    logOutput := captureLog(func() {
        updateSNMPSettings(invalidRequest)
    })
    assert.NotContains(t, logOutput, "myPassword123")
}
```

**Acceptance Criteria:**
- ✅ All credentials encrypted in config file
- ✅ No passwords in API GET responses
- ✅ No passwords in logs
- ✅ Migration works for existing configs
- ✅ All tests pass

---

##### #513: Timing Attack in Authentication (1h) 🔴
**Priority:** P0 - SECURITY
**Assignee:** Security engineer

**Tasks:**
1. Replace string comparison with `subtle.ConstantTimeCompare` (30min)
2. Add timing attack test (30min)

**Implementation:**
```go
import "crypto/subtle"

func (m *Manager) Authenticate(username, password string) (string, error) {
    // Constant-time username comparison
    usernameMatch := subtle.ConstantTimeCompare(
        []byte(username),
        []byte(m.username),
    ) == 1

    // Password comparison (bcrypt already constant-time)
    passwordMatch := bcrypt.CompareHashAndPassword(
        []byte(m.passwordHash),
        []byte(password),
    ) == nil

    // Combine checks - both must succeed
    if !usernameMatch || !passwordMatch {
        return "", ErrInvalidCredentials
    }

    return m.GenerateToken(username)
}
```

**Test Requirements:**
```go
func TestTimingAttackResistance(t *testing.T) {
    mgr := auth.NewManager(secret, timeout, "admin", hash)

    iterations := 1000
    validTimes := []time.Duration{}
    invalidTimes := []time.Duration{}

    for i := 0; i < iterations; i++ {
        // Measure valid username
        start := time.Now()
        mgr.Authenticate("admin", "wrongpass")
        validTimes = append(validTimes, time.Since(start))

        // Measure invalid username
        start = time.Now()
        mgr.Authenticate("wronguser", "wrongpass")
        invalidTimes = append(invalidTimes, time.Since(start))
    }

    // Statistical analysis - mean times should be similar
    validMean := mean(validTimes)
    invalidMean := mean(invalidTimes)

    // Allow 10% variance
    ratio := float64(validMean) / float64(invalidMean)
    assert.InDelta(t, 1.0, ratio, 0.1, "Timing difference suggests vulnerability")
}
```

**Acceptance Criteria:**
- ✅ Username comparison uses constant-time function
- ✅ Timing test shows no statistical difference
- ✅ Existing auth tests still pass

---

##### #517: Modulo Bias in Password Generation (2h) 🔴
**Priority:** P0 - CRYPTO
**Assignee:** Backend engineer

**Tasks:**
1. Implement rejection sampling (1h)
2. Add entropy test (30min)
3. Update password generation calls (30min)

**Implementation:**
```go
// Unbiased random selection from character set
func randomChar(chars string) byte {
    charsLen := byte(len(chars))
    maxValid := 256 - (256 % charsLen)

    for {
        var b [1]byte
        if _, err := rand.Read(b[:]); err != nil {
            panic(err)
        }
        if b[0] < maxValid {
            return chars[b[0]%charsLen]
        }
        // Reject and retry if in biased range
    }
}

func GenerateSecurePassword(length int) (string, error) {
    if length < MinPasswordLength {
        length = MinPasswordLength
    }

    password := make([]byte, length)

    // Ensure at least one of each type
    password[0] = randomChar(lowerChars)
    password[1] = randomChar(upperChars)
    password[2] = randomChar(digitChars)

    // Fill rest
    for i := 3; i < length; i++ {
        password[i] = randomChar(allChars)
    }

    // Shuffle using Fisher-Yates
    for i := len(password) - 1; i > 0; i-- {
        j := int(randomChar(string(make([]byte, i+1))))
        password[i], password[j] = password[j], password[i]
    }

    return string(password), nil
}
```

**Test Requirements:**
```go
func TestPasswordGenerationUniformity(t *testing.T) {
    iterations := 100000
    charCounts := make(map[rune]int)

    for i := 0; i < iterations; i++ {
        pwd, _ := GenerateSecurePassword(16)
        for _, c := range pwd {
            charCounts[c]++
        }
    }

    // Chi-square test for uniform distribution
    expected := float64(iterations*16) / float64(len(allChars))
    chiSquare := 0.0
    for _, count := range charCounts {
        diff := float64(count) - expected
        chiSquare += (diff * diff) / expected
    }

    // Reject if chi-square too high (indicates bias)
    // Critical value for 62 chars at p=0.05: ~81.38
    assert.Less(t, chiSquare, 81.38, "Non-uniform distribution detected")
}
```

**Acceptance Criteria:**
- ✅ No modulo bias in character selection
- ✅ Statistical tests show uniform distribution
- ✅ All password requirements still met

---

##### #522: Input Validation on VLAN ID (4h) 🔴
**Priority:** P0 - SECURITY
**Assignee:** Backend engineer

**Tasks:**
1. Audit all numeric input endpoints (2h)
2. Add comprehensive validation (1h)
3. Add boundary tests (1h)

**Implementation:**
```go
// validation/numeric.go
package validation

import "fmt"

func ValidateVLANID(vlanID int) error {
    if vlanID < 1 || vlanID > 4094 {
        return fmt.Errorf("VLAN ID must be between 1 and 4094, got %d", vlanID)
    }
    return nil
}

func ValidatePort(port int) error {
    if port < 1 || port > 65535 {
        return fmt.Errorf("port must be between 1 and 65535, got %d", port)
    }
    return nil
}

func ValidatePositiveInt(val int, name string) error {
    if val < 0 {
        return fmt.Errorf("%s must be non-negative, got %d", name, val)
    }
    return nil
}

// handlers.go
func (s *Server) deleteVLANInterface(w http.ResponseWriter, r *http.Request) {
    var req VLANInterfaceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate VLAN ID
    if err := validation.ValidateVLANID(req.VlanID); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // ... rest of handler
}
```

**Endpoints to Audit:**
- `/api/vlan` - VLAN ID
- `/api/network/mtu` - MTU value
- `/api/snmp/settings` - Port, timeout, retries
- `/api/discovery/portscan` - Port ranges
- `/api/iperf/client` - Port, duration, bandwidth

**Test Requirements:**
```go
func TestVLANValidation(t *testing.T) {
    tests := []struct {
        vlanID int
        valid  bool
    }{
        {0, false},      // Too low
        {1, true},       // Min valid
        {4094, true},    // Max valid
        {4095, false},   // Too high
        {-1, false},     // Negative
        {99999, false},  // Way too high
    }

    for _, tt := range tests {
        err := validation.ValidateVLANID(tt.vlanID)
        if tt.valid {
            assert.NoError(t, err)
        } else {
            assert.Error(t, err)
        }
    }
}

func TestAPIVLANValidation(t *testing.T) {
    // Test actual API endpoint
    resp := postAPI("/api/vlan/interface", map[string]interface{}{
        "vlanId": -1,  // Invalid
    })
    assert.Equal(t, 400, resp.StatusCode)
    assert.Contains(t, resp.Body, "must be between 1 and 4094")
}
```

**Acceptance Criteria:**
- ✅ All numeric inputs validated
- ✅ Boundary tests for all validators
- ✅ Clear error messages returned
- ✅ No command injection possible

---

##### #523: TLS Configuration Incorrect (1h) 🔴
**Priority:** P0 - SECURITY CONFIG
**Assignee:** DevOps/Backend engineer

**Tasks:**
1. Remove CipherSuites from TLS 1.3 config (15min)
2. Document TLS configuration (30min)
3. Add TLS handshake test (15min)

**Implementation:**
```go
// startHTTPS
func (s *Server) startHTTPS() error {
    // ... cert selection logic ...

    // TLS 1.3 configuration (ciphers not configurable)
    tlsConfig := &tls.Config{
        MinVersion: tls.VersionTLS13,
        // CipherSuites removed - TLS 1.3 uses its own mandatory ciphers
        // If you need to control ciphers, use MinVersion: tls.VersionTLS12
    }

    s.httpServer.TLSConfig = tlsConfig

    log.Printf("Starting HTTPS server with TLS 1.3 (ciphers: mandatory TLS 1.3 suite)")
    return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}
```

**Test Requirements:**
```go
func TestTLSConfiguration(t *testing.T) {
    server := startTestServer()
    defer server.Close()

    // Test TLS handshake
    conn, err := tls.Dial("tcp", server.Addr, &tls.Config{
        InsecureSkipVerify: true,
    })
    require.NoError(t, err)
    defer conn.Close()

    // Verify TLS 1.3
    state := conn.ConnectionState()
    assert.Equal(t, uint16(tls.VersionTLS13), state.Version)

    // Verify cipher is TLS 1.3 cipher
    validCiphers := []uint16{
        tls.TLS_AES_128_GCM_SHA256,
        tls.TLS_AES_256_GCM_SHA384,
        tls.TLS_CHACHA20_POLY1305_SHA256,
    }
    assert.Contains(t, validCiphers, state.CipherSuite)
}
```

**Acceptance Criteria:**
- ✅ CipherSuites removed from TLS 1.3 config
- ✅ Documentation updated
- ✅ TLS handshake test passes

---

#### HIGH Priority Issues

##### #532: CSP Too Permissive (8h) 🟠
**Priority:** P1 - SECURITY
**Assignee:** Frontend + Backend engineer

**Tasks:**
1. Frontend: Remove inline scripts/styles (5h)
2. Backend: Implement nonce-based CSP (2h)
3. Test CSP violations (1h)

**Implementation:**
```go
// Use nonce for inline scripts
func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Generate nonce for this request
        nonce := generateNonce()
        r.Header.Set("X-CSP-Nonce", nonce)

        // Strict CSP
        csp := fmt.Sprintf(
            "default-src 'self'; "+
            "script-src 'self' 'nonce-%s'; "+
            "style-src 'self' 'nonce-%s'; "+
            "img-src 'self' data:; "+
            "connect-src 'self' ws: wss:",
            nonce, nonce,
        )
        w.Header().Set("Content-Security-Policy", csp)

        next.ServeHTTP(w, r)
    })
}
```

**Acceptance Criteria:**
- ✅ No unsafe-inline in CSP
- ✅ Nonce-based inline scripts work
- ✅ CSP violation tests pass

---

##### #535: Password Requirements Too Weak (3h) 🟠
**Priority:** P1 - SECURITY
**Assignee:** Backend engineer

**Tasks:**
1. Increase min length to 12 (30min)
2. Add special character requirement (1h)
3. Add password strength meter (1h 30min)

**Implementation:**
```go
const (
    MinPasswordLength = 12
)

func ValidatePasswordStrength(password string) error {
    if len(password) < MinPasswordLength {
        return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
    }

    var hasUpper, hasLower, hasDigit, hasSpecial bool
    for _, c := range password {
        switch {
        case unicode.IsUpper(c):
            hasUpper = true
        case unicode.IsLower(c):
            hasLower = true
        case unicode.IsDigit(c):
            hasDigit = true
        case unicode.IsPunct(c) || unicode.IsSymbol(c):
            hasSpecial = true
        }
    }

    if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
        return ErrWeakPassword
    }

    return nil
}
```

**Acceptance Criteria:**
- ✅ Min 12 characters enforced
- ✅ Special character required
- ✅ Clear error messages

---

##### #537: Log File Permissions (1h) 🟠
**Priority:** P1 - SECURITY
**Assignee:** Backend engineer

**Tasks:**
1. Set log file permissions to 0600 (30min)
2. Test file permissions (30min)

**Acceptance Criteria:**
- ✅ Log files created with 0600
- ✅ Only owner can read/write

---

##### #533: Self-Signed Cert Weak Key (5min) 🟠
**Priority:** P1 - QUICK WIN
**Assignee:** Any engineer

**Implementation:**
```go
privateKey, err := rsa.GenerateKey(rand.Reader, 4096)  // Changed from 2048
```

**Acceptance Criteria:**
- ✅ 4096-bit RSA key used
- ✅ Cert generation test passes

---

### Sprint 1 Testing Strategy

**Daily Testing:**
- Run `go test -race ./...` before every commit
- Check gosec: `gosec ./...`
- Check govulncheck: `govulncheck ./...`

**End of Sprint:**
- Full integration test suite
- Manual security testing
- Penetration test (if security team available)
- Performance test (ensure security fixes don't degrade performance)

**Definition of Done:**
- ✅ All code reviewed
- ✅ All tests pass (unit, integration, security)
- ✅ No new gosec/govulncheck warnings
- ✅ Documentation updated
- ✅ Deployed to staging, tested

---

## 🔧 SPRINT 2: Concurrency & Stability (Week 3-4)

**Goal:** Fix race conditions and prevent crashes

**Duration:** 2 weeks
**Total Hours:** 34 hours

### Issues in Sprint 2

##### #514: No Mutex Protection (8h) 🔴
**Priority:** P0 - CRITICAL RACE
**Assignee:** Senior backend engineer

**Tasks:**
1. Add sync.RWMutex to Server struct (1h)
2. Protect all shared state access (4h)
3. Run race detector extensively (1h)
4. Add concurrent access tests (2h)

**Implementation:**
```go
type Server struct {
    mu sync.RWMutex  // Protects all fields below

    config           *config.Config
    discoveryManager *discovery.Manager
    deviceDiscovery  *discovery.DeviceDiscovery
    // ... all other fields
}

// Example usage
func (s *Server) GetConfig() *config.Config {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.config
}

func (s *Server) UpdateConfig(cfg *config.Config) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.config = cfg
}
```

**Test Requirements:**
```go
func TestConcurrentConfigAccess(t *testing.T) {
    server := setupTestServer()
    var wg sync.WaitGroup

    // 100 concurrent readers
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                _ = server.GetConfig()
            }
        }()
    }

    // 10 concurrent writers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 10; j++ {
                cfg := config.DefaultConfig()
                server.UpdateConfig(cfg)
            }
        }()
    }

    wg.Wait()
    // If no race detector errors, test passes
}
```

**Acceptance Criteria:**
- ✅ All Server fields protected by mutex
- ✅ `go test -race` shows no warnings
- ✅ Concurrent access tests pass
- ✅ Performance acceptable (benchmark)

---

##### #512: Server Init Race (2h) 🔴
**Priority:** P0
**Assignee:** Backend engineer

**Implementation:**
```go
func NewServer(...) *Server {
    s := &Server{
        // ... initialize fields
    }

    s.wsHub = NewHub()
    go s.wsHub.Run()  // Start immediately

    // Wait for hub to be ready
    <-s.wsHub.Ready()

    s.setupRoutes()  // Now safe
    return s
}
```

**Acceptance Criteria:**
- ✅ Hub started before routes registered
- ✅ No panic on startup requests
- ✅ Startup load test passes

---

##### #519: No Panic Recovery (2h) 🔴
**Priority:** P0
**Assignee:** Backend engineer

**Implementation:**
```go
func recoverMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("PANIC: %v\n%s", err, debug.Stack())
                http.Error(w, "Internal Server Error", 500)

                // Optional: Send to error tracking (Sentry, etc.)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// In setupRoutes:
handler := recoverMiddleware(
    securityHeadersMiddleware(
        corsMiddleware(
            s.authManager.Middleware(s.mux)
        )
    )
)
```

**Test Requirements:**
```go
func TestPanicRecovery(t *testing.T) {
    server := setupTestServer()

    // Add handler that panics
    server.mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
        panic("test panic")
    })

    resp, err := http.Get(server.URL + "/panic")
    require.NoError(t, err)

    // Should get 500, not crash
    assert.Equal(t, 500, resp.StatusCode)

    // Server should still be alive
    resp2, err := http.Get(server.URL + "/api/status")
    require.NoError(t, err)
    assert.Equal(t, 200, resp2.StatusCode)
}
```

**Acceptance Criteria:**
- ✅ Panic recovery at top of middleware stack
- ✅ Panics don't crash server
- ✅ Stack trace logged
- ✅ Tests for all handlers

---

##### #520: Password Hash Race (2h) 🔴
**Priority:** P0
**Assignee:** Backend engineer

**Implementation:**
```go
type Manager struct {
    mu             sync.RWMutex
    jwtSecret      []byte
    sessionTimeout time.Duration
    passwordHash   string
    username       string
}

func (m *Manager) Authenticate(username, password string) (string, error) {
    m.mu.RLock()
    currentUsername := m.username
    currentHash := m.passwordHash
    m.mu.RUnlock()

    // Now do comparison outside lock
    // ... rest of authentication
}

func (m *Manager) UpdatePasswordHash(hash string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.passwordHash = hash
}
```

**Acceptance Criteria:**
- ✅ Mutex protects passwordHash
- ✅ Race detector clean
- ✅ Concurrent auth + password change test

---

##### #515: HTTP Redirect Server Leak (3h) 🔴
**Priority:** P0
**Assignee:** Backend engineer

**Implementation:**
```go
type Server struct {
    // ... existing fields
    redirectServer *http.Server
}

func (s *Server) Start() error {
    // ... existing code ...

    if s.config.Server.HTTPS && s.config.Server.HTTPRedirectPort > 0 {
        s.redirectServer = s.startHTTPRedirect(s.config.Server.HTTPRedirectPort)
    }

    // ... start HTTPS
}

func (s *Server) Shutdown(ctx context.Context) error {
    // ... existing code ...

    // Shutdown redirect server
    if s.redirectServer != nil {
        if err := s.redirectServer.Shutdown(ctx); err != nil {
            log.Printf("Redirect server shutdown error: %v", err)
        }
    }

    return s.httpServer.Shutdown(ctx)
}
```

**Test Requirements:**
```go
func TestRedirectServerShutdown(t *testing.T) {
    server := startServerWithRedirect()

    // Verify redirect works
    resp, _ := http.Get("http://localhost:8080")
    assert.Equal(t, 301, resp.StatusCode)

    // Shutdown
    server.Shutdown(context.Background())

    // Verify port released
    time.Sleep(100 * time.Millisecond)
    ln, err := net.Listen("tcp", ":8080")
    require.NoError(t, err, "Port should be available")
    ln.Close()
}
```

**Acceptance Criteria:**
- ✅ Redirect server tracked
- ✅ Shutdown releases port
- ✅ No goroutine leak

---

#### HIGH Priority Issues in Sprint 2

##### #534: No Context Propagation (4h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Implementation:**
```go
func (s *Server) Start() error {
    ctx, cancel := context.WithCancel(context.Background())
    s.shutdownCancel = cancel

    // Start services with context
    errChan := make(chan error, 5)

    go func() {
        if err := s.linkMonitor.StartWithContext(ctx); err != nil {
            errChan <- fmt.Errorf("link monitor: %w", err)
        }
    }()

    go func() {
        if err := s.discoveryService.StartWithContext(ctx); err != nil {
            errChan <- fmt.Errorf("discovery: %w", err)
        }
    }()

    // Wait for critical services or failure
    select {
    case err := <-errChan:
        cancel()  // Stop all services
        return err
    case <-time.After(5 * time.Second):
        // All started successfully
    }

    // ... start HTTP server
}
```

**Acceptance Criteria:**
- ✅ All services use context
- ✅ Failure stops all services
- ✅ Graceful startup/shutdown

---

##### #530: No Rate Limiting on Expensive Endpoints (6h) 🟠
**Priority:** P1 - DOS PREVENTION
**Assignee:** Backend engineer

**Implementation:**
```go
// Per-endpoint rate limiters
var (
    scanLimiter = NewRateLimiter(RateLimitConfig{
        RequestsPerMinute: 2,   // Max 2 scans per minute
        BurstSize:         1,
        BlockDuration:     5 * time.Minute,
    })

    vulnScanLimiter = NewRateLimiter(RateLimitConfig{
        RequestsPerMinute: 1,   // Max 1 vuln scan per minute
        BurstSize:         1,
        BlockDuration:     10 * time.Minute,
    })
)

func (s *Server) handleDevicesScan(w http.ResponseWriter, r *http.Request) {
    clientIP := GetClientIP(r)

    if scanLimiter.IsBlocked(clientIP) {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    // ... rest of handler
}
```

**Test Requirements:**
```go
func TestScanRateLimit(t *testing.T) {
    server := setupTestServer()

    // First 2 requests should succeed
    resp1, _ := http.Post(server.URL+"/api/devices/scan", "application/json", nil)
    assert.Equal(t, 200, resp1.StatusCode)

    resp2, _ := http.Post(server.URL+"/api/devices/scan", "application/json", nil)
    assert.Equal(t, 200, resp2.StatusCode)

    // Third should be rate limited
    resp3, _ := http.Post(server.URL+"/api/devices/scan", "application/json", nil)
    assert.Equal(t, 429, resp3.StatusCode)
}
```

**Acceptance Criteria:**
- ✅ Rate limits on scan, vuln, speedtest
- ✅ Different limits per endpoint
- ✅ Tests for each endpoint

---

##### #538: Environment Override Unlogged (30min) 🟠
**Priority:** P1 - QUICK WIN
**Assignee:** Any engineer

**Acceptance Criteria:**
- ✅ All env overrides logged at startup
- ✅ Clear visibility into config source

---

##### #539: Duplicated JWT Secret Logic (1h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Acceptance Criteria:**
- ✅ Single code path for JWT secret generation
- ✅ Code deduplication complete

---

### Sprint 2 Testing Strategy

**Concurrency Testing:**
- All tests run with `-race` flag
- Load test with 1000 concurrent requests
- Chaos test: Random service failures

**Definition of Done:**
- ✅ Race detector clean
- ✅ Load tests pass
- ✅ All handlers have panic tests
- ✅ Performance benchmarks meet targets

---

## 🗄️ SPRINT 3: Data Integrity & Resources (Week 5-6)

**Goal:** Prevent data loss and resource leaks

**Duration:** 2 weeks
**Total Hours:** 42 hours

### Issues in Sprint 3

##### #524: Shutdown Data Loss (4h) 🔴
**Priority:** P0
**Assignee:** Backend engineer

**Implementation:**
```go
type Server struct {
    shutdownWg sync.WaitGroup
    // ... existing fields
}

func (s *Server) Start() error {
    // Start services and track with WaitGroup
    s.shutdownWg.Add(4)

    go func() {
        defer s.shutdownWg.Done()
        s.linkMonitor.Run()
    }()

    go func() {
        defer s.shutdownWg.Done()
        s.discoveryService.Run()
    }()

    // ... other services
}

func (s *Server) Shutdown(ctx context.Context) error {
    // Signal all services to stop
    s.wsHub.Shutdown()
    s.linkMonitor.Stop()
    s.discoveryService.Stop()
    // ... etc

    // Wait for all to complete or timeout
    done := make(chan struct{})
    go func() {
        s.shutdownWg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("All services stopped gracefully")
    case <-ctx.Done():
        log.Println("Shutdown timeout - some services may not have stopped")
    }

    return s.httpServer.Shutdown(ctx)
}
```

**Test Requirements:**
```go
func TestGracefulShutdown(t *testing.T) {
    server := startServer()

    // Start some long-running operations
    go server.discoveryService.Scan()
    go server.surveyManager.SaveSurvey(largeSurvey)

    // Shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err := server.Shutdown(ctx)
    require.NoError(t, err)

    // Verify data was saved
    surveys, _ := server.surveyManager.LoadSurveys()
    assert.Contains(t, surveys, largeSurvey.ID)
}
```

**Acceptance Criteria:**
- ✅ WaitGroup tracks all services
- ✅ Shutdown waits for completion
- ✅ Data integrity test passes
- ✅ No corrupted files after shutdown

---

##### #525: JWT Token Revocation (8h) 🔴
**Priority:** P0
**Assignee:** Backend engineer

**Implementation:**
```go
type Manager struct {
    mu             sync.RWMutex
    jwtSecret      []byte
    sessionTimeout time.Duration
    passwordHash   string
    username       string
    tokenVersion   int  // Incremented on password change
}

type Claims struct {
    Username     string `json:"username"`
    TokenVersion int    `json:"token_version"`
    jwt.RegisteredClaims
}

func (m *Manager) GenerateToken(username string) (string, error) {
    m.mu.RLock()
    version := m.tokenVersion
    m.mu.RUnlock()

    claims := &Claims{
        Username:     username,
        TokenVersion: version,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.sessionTimeout)),
            // ...
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.jwtSecret)
}

func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return m.jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    claims := token.Claims.(*Claims)

    // Check token version
    m.mu.RLock()
    currentVersion := m.tokenVersion
    m.mu.RUnlock()

    if claims.TokenVersion != currentVersion {
        return nil, errors.New("token revoked")
    }

    return claims, nil
}

func (m *Manager) RevokeAllTokens() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.tokenVersion++
}
```

**Test Requirements:**
```go
func TestTokenRevocation(t *testing.T) {
    mgr := auth.NewManager(secret, timeout, "admin", hash)

    // Generate token
    token1, _ := mgr.Authenticate("admin", "password")

    // Verify token works
    _, err := mgr.ValidateToken(token1)
    assert.NoError(t, err)

    // Change password (revokes all tokens)
    newHash, _ := auth.HashPassword("newPassword")
    mgr.UpdatePasswordHash(newHash)
    mgr.RevokeAllTokens()

    // Old token should be invalid
    _, err = mgr.ValidateToken(token1)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "revoked")

    // New token should work
    token2, _ := mgr.Authenticate("admin", "newPassword")
    _, err = mgr.ValidateToken(token2)
    assert.NoError(t, err)
}
```

**Acceptance Criteria:**
- ✅ Token version in claims
- ✅ Password change revokes all tokens
- ✅ Revocation test passes
- ✅ API endpoint to force logout all sessions

---

##### #516: FD Leak in SPA Handler (1h) 🔴
**Priority:** P0 - QUICK WIN
**Assignee:** Any engineer

**Implementation:**
```go
func spaHandler(fsys http.FileSystem) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ... path logic ...

        f, err := fsys.Open(path)
        if err != nil {
            // ... fallback logic ...
        }
        defer f.Close()  // Always close

        stat, err := f.Stat()
        if err != nil {
            http.Error(w, "Internal Server Error", 500)
            return
        }

        if stat.IsDir() {
            indexPath := strings.TrimSuffix(path, "/") + "/index.html"
            f2, err := fsys.Open(indexPath)
            if err != nil {
                // Fallback
                path = "/index.html"
                f2, err = fsys.Open(path)
                if err != nil {
                    return
                }
            }
            defer f2.Close()  // Ensure f2 is closed
            // Don't reassign f - let it close naturally

            stat2, err := f2.Stat()
            if err != nil {
                return
            }

            rs, ok := f2.(io.ReadSeeker)
            if !ok {
                http.Error(w, "Internal Server Error", 500)
                return
            }
            http.ServeContent(w, r, stat2.Name(), stat2.ModTime(), rs)
            return
        }

        // ... serve file
    })
}
```

**Test Requirements:**
```go
func TestNoFDLeak(t *testing.T) {
    server := startTestServer()

    // Get initial FD count
    initialFDs := getOpenFDCount()

    // Make 1000 requests
    for i := 0; i < 1000; i++ {
        resp, _ := http.Get(server.URL + "/")
        resp.Body.Close()
    }

    // FD count should be stable
    finalFDs := getOpenFDCount()
    assert.InDelta(t, initialFDs, finalFDs, 10, "FD leak detected")
}

func getOpenFDCount() int {
    // Linux: count files in /proc/self/fd/
    files, _ := ioutil.ReadDir("/proc/self/fd")
    return len(files)
}
```

**Acceptance Criteria:**
- ✅ All file handles properly closed
- ✅ FD leak test passes
- ✅ Load test shows stable FD count

---

##### #521: Config Mutation Without Persistence (1h) 🔴
**Priority:** P0 - QUICK WIN
**Assignee:** Backend engineer

**Implementation:**
```go
if activeInterface != cfg.Interface.Default {
    log.Printf("Using detected active interface %s instead of configured default %s",
        activeInterface, cfg.Interface.Default)
    cfg.Interface.Default = activeInterface

    // Persist the change
    if err := cfg.Save(*configPath); err != nil {
        log.Printf("Warning: Failed to persist interface change: %v", err)
    } else {
        log.Printf("Persisted interface change to config")
    }
}
```

**Acceptance Criteria:**
- ✅ Interface change persisted
- ✅ Restart test shows interface retained
- ✅ Config file updated

---

#### HIGH Priority Issues in Sprint 3

##### #529: WriteTimeout Too Short (2h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Implementation:**
```go
s.httpServer = &http.Server{
    Addr:         addr,
    Handler:      handler,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 60 * time.Second,  // Increased from 15s
    IdleTimeout:  60 * time.Second,
}
```

**Or per-endpoint timeout:**
```go
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.TimeoutHandler(next, timeout, "Request timeout")
    }
}

// For large uploads
s.mux.Handle("/api/survey/floorplan",
    timeoutMiddleware(5*time.Minute)(http.HandlerFunc(s.updateSurveyFloorPlan)))
```

**Acceptance Criteria:**
- ✅ Large uploads don't timeout
- ✅ Survey uploads work
- ✅ Vulnerability scans complete

---

##### #531: CORS Preflight Cache (15min) 🟠
**Priority:** P1 - QUICK WIN
**Assignee:** Any engineer

**Implementation:**
```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")

        if origin == "" || isAllowedOrigin(origin) {
            if origin != "" {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Vary", "Origin")
            }
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
            w.Header().Set("Access-Control-Max-Age", "86400")  // Cache for 24 hours
        }

        // ... rest
    })
}
```

**Acceptance Criteria:**
- ✅ Preflight cached for 24h
- ✅ Request volume reduced

---

##### #536: Token Blacklist Implementation (8h) 🟠
**Priority:** P1 - ENHANCEMENT
**Assignee:** Backend engineer

**Implementation:**
```go
// Simple in-memory blacklist (or use Redis for distributed)
type TokenBlacklist struct {
    mu        sync.RWMutex
    blacklist map[string]time.Time  // token ID -> expiration
}

func (b *TokenBlacklist) Add(tokenID string, expiration time.Time) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.blacklist[tokenID] = expiration
}

func (b *TokenBlacklist) IsBlacklisted(tokenID string) bool {
    b.mu.RLock()
    defer b.mu.RUnlock()
    exp, exists := b.blacklist[tokenID]
    if !exists {
        return false
    }
    if time.Now().After(exp) {
        // Expired, can remove
        delete(b.blacklist, tokenID)
        return false
    }
    return true
}

// In ValidateToken
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
    // ... parse token ...

    // Check blacklist
    if m.blacklist.IsBlacklisted(claims.ID) {
        return nil, errors.New("token blacklisted")
    }

    return claims, nil
}
```

**Acceptance Criteria:**
- ✅ Tokens can be individually blacklisted
- ✅ Blacklist automatically cleans expired entries
- ✅ API endpoint to revoke specific token

---

##### #540: Health Endpoint & Metrics (8h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Implementation:**
```go
type HealthResponse struct {
    Status     string                 `json:"status"`  // "healthy", "degraded", "unhealthy"
    Version    string                 `json:"version"`
    Uptime     int64                  `json:"uptime"`
    Components map[string]ComponentHealth `json:"components"`
}

type ComponentHealth struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    components := map[string]ComponentHealth{
        "discovery": {
            Status:  s.discoveryService.HealthStatus(),
            Message: s.discoveryService.HealthMessage(),
        },
        "websocket": {
            Status: s.wsHub.HealthStatus(),
        },
        // ... other components
    }

    // Overall status
    status := "healthy"
    for _, comp := range components {
        if comp.Status == "unhealthy" {
            status = "unhealthy"
            break
        } else if comp.Status == "degraded" {
            status = "degraded"
        }
    }

    resp := HealthResponse{
        Status:     status,
        Version:    version.Version,
        Uptime:     time.Since(startTime).Milliseconds(),
        Components: components,
    }

    statusCode := http.StatusOK
    if status == "unhealthy" {
        statusCode = http.StatusServiceUnavailable
    }

    sendJSONResponse(w, statusCode, resp)
}

// Prometheus metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
    // Use prometheus library
    promhttp.Handler().ServeHTTP(w, r)
}
```

**Acceptance Criteria:**
- ✅ /health endpoint returns component status
- ✅ /metrics endpoint exposes Prometheus metrics
- ✅ Kubernetes health probes work
- ✅ Grafana dashboards can be created

---

##### #541: Signal Handling Race (1h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Implementation:**
```go
// Signal handler
sigChan := make(chan os.Signal, 10)  // Larger buffer
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

select {
case sig := <-sigChan:
    log.Printf("Received signal: %v, shutting down...", sig)
    // ... shutdown logic
}
```

**Acceptance Criteria:**
- ✅ Larger signal buffer
- ✅ Multiple signals handled correctly

---

##### #542: Config Validation (4h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Implementation:**
```go
func (c *Config) Validate() error {
    var errs []error

    // Port validation
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        errs = append(errs, fmt.Errorf("invalid port: %d", c.Server.Port))
    }

    // Timeout validation
    if c.DNS.Timeout <= 0 {
        errs = append(errs, fmt.Errorf("DNS timeout must be positive"))
    }

    // Path validation
    if c.Survey.StoragePath != "" {
        if err := os.MkdirAll(c.Survey.StoragePath, 0750); err != nil {
            errs = append(errs, fmt.Errorf("invalid storage path: %w", err))
        }
    }

    // Interface validation
    if c.Interface.Default == "" {
        errs = append(errs, fmt.Errorf("default interface not specified"))
    }

    if len(errs) > 0 {
        return fmt.Errorf("config validation failed: %v", errs)
    }
    return nil
}

// In main.go
cfg, _, err := config.EnsureConfig(*configPath, auth.IsDefaultPasswordHash)
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

if err := cfg.Validate(); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

**Acceptance Criteria:**
- ✅ All critical fields validated
- ✅ Clear error messages
- ✅ Server refuses to start with invalid config

---

##### #543: Fallback JWT Secret Predictable (1h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Implementation:**
```go
func generateRandomSecret() string {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        // If crypto/rand fails, this is a critical system issue
        panic(fmt.Sprintf("crypto/rand failed: %v - system is insecure", err))
    }
    return base64.URLEncoding.EncodeToString(bytes)
}
```

**Acceptance Criteria:**
- ✅ Panic instead of fallback
- ✅ No predictable secrets

---

### Sprint 3 Testing Strategy

**Data Integrity Testing:**
- Test shutdown during write operations
- Test config persistence
- Test survey data integrity

**Resource Testing:**
- FD leak tests
- Memory leak tests
- Goroutine leak tests

**Definition of Done:**
- ✅ No data loss on shutdown
- ✅ No resource leaks
- ✅ Health checks working
- ✅ Metrics available

---

## 🏗️ SPRINT 4: Architecture & Tech Debt (Week 7-8)

**Goal:** Improve code quality and maintainability

**Duration:** 2 weeks
**Total Hours:** 35 hours

### Issues in Sprint 4

##### #544: Split Large Handlers File (6h) 🟠
**Priority:** P1
**Assignee:** Backend engineer

**Implementation:**
Split `handlers.go` (4619 lines) into:
- `handlers_auth.go` - Login, logout, setup
- `handlers_discovery.go` - Discovery, devices, scan
- `handlers_network.go` - Interfaces, link, VLAN, WiFi
- `handlers_tests.go` - Custom tests, speedtest, iperf
- `handlers_survey.go` - Survey endpoints
- `handlers_vulnerabilities.go` - Vulnerability scanning
- `handlers_system.go` - Health, logs, status

**Acceptance Criteria:**
- ✅ All handlers split logically
- ✅ No change in functionality
- ✅ All tests still pass
- ✅ Easier to navigate

---

##### #526: Remove Unused padRight Function (5min) 🟠
**Priority:** P1 - QUICK WIN

```go
// Remove lines 242-247 from main.go
```

**Acceptance Criteria:**
- ✅ Dead code removed

---

##### #527: Fix Brand Name (30min) 🟠
**Priority:** P1 - QUICK WIN

```bash
# Find and replace
grep -r "NetScope" . --exclude-dir=vendor
# Replace with "LuminetIQ"
```

**Acceptance Criteria:**
- ✅ All "NetScope" → "LuminetIQ"
- ✅ JWT issuer updated
- ✅ Cert organization updated

---

##### #528: Configurable Retry Logic (2h) 🟠
**Priority:** P1

**Implementation:**
```yaml
# luminetiq.yaml
interface:
  default: eth0
  fallbacks: ["enp0s3", "wlan0"]
  detection_retries: 3
  detection_retry_delay: 5s
```

```go
// Use config values
for retryCount < cfg.Interface.DetectionRetries {
    time.Sleep(cfg.Interface.DetectionRetryDelay)
    // ... retry logic
}
```

**Acceptance Criteria:**
- ✅ Retry configurable
- ✅ Default values preserved
- ✅ Works in CI (fast failure)

---

##### #540: Observability (Continued from Sprint 3)

Add comprehensive metrics:
- Request count by endpoint
- Request duration by endpoint
- Active connections
- Discovery scan count
- Error count by type

**Acceptance Criteria:**
- ✅ Prometheus metrics exposed
- ✅ Grafana dashboard created
- ✅ Alerting rules defined

---

##### MEDIUM Priority Issues (Selected)

Pick 5-10 MEDIUM issues based on business priority:
- API versioning
- Request size limits
- Structured logging
- Error message sanitization
- Documentation updates

---

### Sprint 4 Testing Strategy

**Code Quality:**
- Run golangci-lint with strict rules
- Code coverage report
- Complexity analysis

**Definition of Done:**
- ✅ Code organization improved
- ✅ Tech debt reduced
- ✅ Metrics available
- ✅ Documentation updated

---

## 📊 Overall Testing Requirements

### Continuous Testing (All Sprints)

**Daily:**
```bash
# Race detector (mandatory)
go test -race ./...

# Security scan
gosec ./...
govulncheck ./...

# Coverage
go test -cover ./...
```

**Weekly:**
```bash
# Full integration suite
go test -tags=integration ./...

# Load test
k6 run loadtest.js

# Security scan
trivy fs .
```

**End of Each Sprint:**
```bash
# Full regression suite
go test -v ./...

# Performance benchmarks
go test -bench=. -benchmem ./...

# E2E tests
npm run test:e2e
```

### Test Coverage Targets

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| internal/api | 13.4% | 70% | High |
| internal/auth | 45% | 90% | High |
| internal/discovery | 17.4% | 60% | High |
| internal/config | 62% | 80% | Medium |
| All packages | ~35% | 70% | High |

---

## 🎯 Success Metrics

### Sprint 1 Success Criteria
- ✅ 0 critical security vulnerabilities
- ✅ All credentials encrypted
- ✅ Timing attack mitigated
- ✅ Input validation comprehensive

### Sprint 2 Success Criteria
- ✅ 0 race detector warnings
- ✅ Load test: 1000 concurrent requests without crash
- ✅ No panics cause server crash

### Sprint 3 Success Criteria
- ✅ 0 data loss on shutdown
- ✅ 0 FD leaks
- ✅ Health endpoint operational
- ✅ Token revocation working

### Sprint 4 Success Criteria
- ✅ Code complexity reduced
- ✅ Coverage >70%
- ✅ Observability dashboard live

---

## 📋 Definition of Done (All Sprints)

For each issue to be considered "done":

1. **Code Complete**
   - ✅ Implementation matches specification
   - ✅ Code review approved by 2+ engineers
   - ✅ No new linter warnings

2. **Tests Complete**
   - ✅ Unit tests written and passing
   - ✅ Integration tests passing
   - ✅ Coverage meets target
   - ✅ Race detector clean

3. **Documentation Complete**
   - ✅ Code comments added
   - ✅ README updated (if needed)
   - ✅ API docs updated (if needed)

4. **Deployed & Verified**
   - ✅ Deployed to staging
   - ✅ Manual testing passed
   - ✅ No new errors in logs
   - ✅ Performance acceptable

5. **Issue Closed**
   - ✅ GitHub issue updated
   - ✅ Linked to PR
   - ✅ Closed with summary

---

**End of Sprint Plan**
