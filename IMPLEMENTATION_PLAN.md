# Seed Project Implementation Plan

This document outlines the implementation status and remaining work for the Seed project.

**Last Updated**: December 2024

---

## Completed Fixes (Security Session)

### P0 Security - Critical (All Complete)

#### ✅ #660 - WebSocket Cookie-Based Authentication

**Status**: IMPLEMENTED

**Changes Made**:

- `/internal/api/websocket.go`: Changed `handleWebSocket` to use httpOnly cookie authentication via
  `auth.GetTokenFromRequest(r)` instead of `Sec-WebSocket-Protocol` header
- `/web/src/hooks/useWebSocket.ts`: Deprecated `token` and `onRefreshToken` parameters, removed protocol header
  authentication

**Security Impact**: Tokens no longer exposed in WebSocket protocol headers which could be logged by proxies/load
balancers.

#### ✅ #724/#758 - Setup Endpoint Hardening

**Status**: IMPLEMENTED

**Changes Made**:

- `/internal/api/setup_token.go`: NEW FILE - Implements `SetupTokenManager` with one-time token generation and
  validation
- `/internal/api/server.go`: Added `setupTokenManager` field to Server struct
- `/internal/api/handlers_auth.go`: Updated `handleSetupStatus` to generate tokens, `handleSetupComplete` to validate
  them
- `/web/src/components/setup/SetupWizard.tsx`: Added `setupToken` prop
- `/web/src/components/setup/setupApi.ts`: Added `setupToken` to response interface
- `/web/src/App.tsx`: Added setupToken state management

**Security Impact**: Setup endpoint now requires a one-time token, preventing CSRF attacks and replay attacks on the
setup process.

### P1 Reliability (Partial)

#### ✅ #714 - Logging IP Header Security

**Status**: IMPLEMENTED

**Changes Made**:

- `/internal/logging/redact.go`: Added IP-related headers (`x-forwarded-for`, `x-real-ip`, `cf-connecting-ip`,
  `true-client-ip`, `x-client-ip`, `x-cluster-client-ip`, `forwarded`) to `sensitiveHeaders` map
- Updated `GetClientIP` documentation with security warnings about spoofable headers

**Security Impact**: Client IP addresses are now redacted from logs, protecting user privacy.

#### ⚠️ #572 - Remove Hardcoded Interface Names

**Status**: PARTIALLY IMPLEMENTED

**Changes Made**:

- `/web/src/App.tsx`: Changed default interface from `"eth0"` to `""` (empty string)
- `/web/src/components/cards/WiFiSurveyCard.tsx`: Added `currentInterface` prop instead of hardcoding `"wlan0"`

**Remaining Work**: See #571 below for full interface auto-detection integration.

### P3 UX

#### ✅ #769 - SSO Button Visibility

**Status**: IMPLEMENTED

**Changes Made**:

- `/web/src/App.tsx`: `LoginForm` component now fetches SSO providers from `/api/sso/providers` and conditionally
  renders SSO buttons based on `hasEnabledSSO`

---

## Remaining Issues by Priority

### P1 Reliability - High Priority

#### 🔴 #669 - Remove getAuthHeaders() Usage

**Priority**: P1 - High **Effort**: Medium (90+ occurrences) **Dependencies**: None

**Current State**: `getAuthHeaders()` returns an empty object `{}` but is still called 90+ times throughout the
codebase. Most API calls should use `credentials: 'include'` instead.

**Implementation Plan**:

1. **Audit all usages**:

   ```bash
   grep -r "getAuthHeaders" web/src --include="*.ts" --include="*.tsx"
   ```

2. **Categorize by pattern**:
   - Fetch calls with `headers: getAuthHeaders()`
   - Axios calls with `headers: getAuthHeaders()`
   - Custom hook usages

3. **Replace patterns**:

   ```typescript
   // Before
   fetch(url, { headers: getAuthHeaders() });

   // After
   fetch(url, { credentials: "include" });
   ```

4. **Files to update** (known from investigation):
   - `web/src/lib/api.ts` - Core API utilities
   - `web/src/hooks/useProfiles.ts`
   - `web/src/hooks/useSurvey.ts`
   - `web/src/hooks/useFirewall.ts`
   - `web/src/hooks/useUsers.ts`
   - `web/src/hooks/useLogs.ts`
   - `web/src/components/settings/*.tsx`
   - All other data fetching hooks

5. **Deprecate and remove**:
   - Mark `getAuthHeaders()` as deprecated
   - After all usages removed, delete the function

**Testing**:

- Test all API endpoints still work with cookie auth
- Verify CORS preflight requests succeed
- Test token refresh flow still works

---

#### 🔴 #571 - Interface Auto-Detection Integration

**Priority**: P1 - High **Effort**: Medium **Dependencies**: Backend interface detection endpoint

**Current State**: Backend has interface detection in `/internal/network/interfaces.go` but frontend doesn't use it
properly.

**Implementation Plan**:

1. **Backend**: Verify `/api/system/interfaces` endpoint returns:

   ```json
   {
     "interfaces": [
       { "name": "eth0", "displayName": "Ethernet", "type": "ethernet", "state": "up" },
       { "name": "wlan0", "displayName": "WiFi", "type": "wifi", "state": "up" }
     ],
     "defaultInterface": "eth0"
   }
   ```

2. **Frontend**: Create `useInterfaces` hook:

   ```typescript
   // web/src/hooks/useInterfaces.ts
   export function useInterfaces() {
     const [interfaces, setInterfaces] = useState<Interface[]>([]);
     const [defaultInterface, setDefaultInterface] = useState<string>("");
     // Fetch from /api/system/interfaces on mount
   }
   ```

3. **Update App.tsx**:
   - Replace hardcoded interface with auto-detected default
   - Pass detected interface to child components

4. **Update components**:
   - `InterfaceCard.tsx` - Use detected interfaces
   - `WiFiSurveyCard.tsx` - Already has prop, just pass correct value
   - `ConnectionsCard.tsx` - Use detected interface
   - Settings pages - Use interface dropdown

**Testing**:

- Test on system with multiple interfaces
- Verify correct interface selected by default
- Test interface switching UI

---

#### 🔴 #608 - WebSocket Client-to-Server Messages

**Priority**: P1 - Medium **Effort**: Medium **Dependencies**: #660 (completed)

**Current State**: WebSocket is one-way (server pushes to client). Need to implement client-to-server message handling.

**Implementation Plan**:

1. **Define message protocol**:

   ```typescript
   interface WSMessage {
     type: string;
     payload: unknown;
   }
   ```

2. **Backend handler** (`/internal/api/websocket.go`):

   ```go
   func (s *Server) handleWSMessage(conn *websocket.Conn, msg []byte) {
     var message WSMessage
     if err := json.Unmarshal(msg, &message); err != nil {
       return
     }
     switch message.Type {
     case "subscribe":
       // Handle subscription to specific topics
     case "ping":
       // Handle heartbeat
     }
   }
   ```

3. **Frontend hook** (`useWebSocket.ts`):

   ```typescript
   const send = useCallback(
     (type: string, payload: unknown) => {
       if (ws?.readyState === WebSocket.OPEN) {
         ws.send(JSON.stringify({ type, payload }));
       }
     },
     [ws]
   );
   ```

4. **Message types to implement**:
   - `subscribe` - Subscribe to specific data topics
   - `unsubscribe` - Unsubscribe from topics
   - `ping` - Client heartbeat
   - `action` - Trigger server-side actions (with proper auth)

**Testing**:

- Test message serialization/deserialization
- Test subscription management
- Test error handling for malformed messages

---

### P2 Feature Completeness

#### 🟡 #754 - MSP Profile System Runtime Config

**Priority**: P2 - Medium **Effort**: Large **Dependencies**: Profile management API

**Current State**: Profile system exists but runtime configuration (applying firewall rules, DNS settings, etc.) not
implemented.

**Implementation Plan**:

1. **Profile application service** (`/internal/profiles/apply.go`):

   ```go
   type ProfileApplicator struct {
     firewall *firewall.Manager
     dns      *dns.Manager
     network  *network.Manager
   }

   func (p *ProfileApplicator) Apply(profile *Profile) error {
     // Apply firewall rules
     // Apply DNS settings
     // Apply network settings
     // Apply service configurations
   }
   ```

2. **API endpoint** (`/api/profiles/:id/apply`):
   - POST to apply profile
   - GET to check application status
   - DELETE to revert to defaults

3. **Frontend**:
   - Add "Apply" button to profile cards
   - Show application status
   - Show revert option

4. **Rollback mechanism**:
   - Store current config before applying
   - Implement rollback on failure
   - Timeout auto-rollback for critical changes

**Testing**:

- Test each configuration type separately
- Test rollback on failure
- Test persistence across restarts

---

#### 🟡 #694 - Standardized Error Response Format

**Priority**: P2 - Medium **Effort**: Medium **Dependencies**: None

**Current State**: Many handlers use `http.Error()` with inconsistent formats. Need standardized JSON responses.

**Implementation Plan**:

1. **Define error structure** (`/internal/api/errors.go`):

   ```go
   type APIError struct {
     Code    string `json:"code"`
     Message string `json:"message"`
     Details any    `json:"details,omitempty"`
   }

   func WriteError(w http.ResponseWriter, status int, code, message string) {
     w.Header().Set("Content-Type", "application/json")
     w.WriteHeader(status)
     json.NewEncoder(w).Encode(APIError{Code: code, Message: message})
   }
   ```

2. **Common error codes**:

   ```go
   const (
     ErrCodeBadRequest     = "BAD_REQUEST"
     ErrCodeUnauthorized   = "UNAUTHORIZED"
     ErrCodeForbidden      = "FORBIDDEN"
     ErrCodeNotFound       = "NOT_FOUND"
     ErrCodeConflict       = "CONFLICT"
     ErrCodeInternal       = "INTERNAL_ERROR"
     ErrCodeValidation     = "VALIDATION_ERROR"
   )
   ```

3. **Migrate handlers**:
   - Search for `http.Error(` usages
   - Replace with `WriteError()` calls
   - Add appropriate error codes

4. **Frontend error handling**:

   ```typescript
   interface APIError {
     code: string;
     message: string;
     details?: unknown;
   }

   function handleAPIError(error: APIError) {
     // Display user-friendly error based on code
   }
   ```

**Testing**:

- Test all error paths return JSON
- Test error codes are consistent
- Test frontend error display

---

### P3 UX - Lower Priority

#### 🟢 #574 - Display Friendly Interface Names

**Priority**: P3 - Low **Effort**: Small **Dependencies**: #571

**Implementation Plan**:

1. **Backend mapping** (`/internal/network/interfaces.go`):

   ```go
   func GetDisplayName(iface string) string {
     // Map technical names to friendly names
     // eth0 -> "Ethernet"
     // wlan0 -> "WiFi"
     // etc.
   }
   ```

2. **Frontend display**:
   - Use `displayName` from interface API
   - Show technical name in tooltip/secondary text

---

#### 🟢 #681 - Console Statement Cleanup

**Priority**: P3 - Low **Effort**: Small **Dependencies**: None

**Current State**: 4 files still have console statements.

**Implementation Plan**:

1. **Find remaining console statements**:

   ```bash
   grep -r "console\." web/src --include="*.ts" --include="*.tsx"
   ```

2. **Replace with logger**:

   ```typescript
   // Before
   console.log("Debug:", data);

   // After
   logger.debug(LogComponents.API, "Debug message", data);
   ```

3. **Add ESLint rule enforcement**:
   ```json
   {
     "rules": {
       "no-console": "error"
     }
   }
   ```

---

#### 🟢 #625/#624/#629 - i18n Extraction

**Priority**: P3 - Low **Effort**: Medium **Dependencies**: None

**Current State**: Some hardcoded strings remain in components.

**Implementation Plan**:

1. **Audit for hardcoded strings**:
   - Focus on user-facing text
   - Ignore technical strings (CSS classes, etc.)

2. **Add to translation files**:
   - `web/public/locales/en/cards.json`
   - `web/public/locales/en/common.json`
   - Other namespace files as needed

3. **Update components**:

   ```typescript
   // Before
   <span>Loading...</span>

   // After
   <span>{t("common.loading")}</span>
   ```

---

## Implementation Order

Based on dependencies and priority:

### Phase 1: API Cleanup (Immediate)

1. ✅ #660 - WebSocket auth (DONE)
2. ✅ #724/#758 - Setup tokens (DONE)
3. ✅ #714 - Log redaction (DONE)
4. 🔴 #669 - Remove getAuthHeaders (NEXT)
5. 🔴 #694 - Standardize error responses

### Phase 2: Interface Management

1. 🔴 #571 - Interface auto-detection
2. ✅ #572 - Remove hardcoded names (partially done)
3. 🟢 #574 - Friendly interface names

### Phase 3: WebSocket Enhancement

1. 🔴 #608 - Client-to-server messages

### Phase 4: Profile System

1. 🟡 #754 - Runtime config application

### Phase 5: Polish

1. 🟢 #681 - Console cleanup
2. 🟢 #625/#624/#629 - i18n extraction
3. ✅ #769 - SSO visibility (DONE)

---

## Legacy Issues (From Previous Plan)

The following issues from the original plan may still need attention:

### Code Quality

- #565 - Staticcheck issues (potential bugs)
- #560 - Cyclomatic complexity (29 issues)
- #561 - Revive style issues (50 issues)
- #562 - Repeated strings to constants (23 issues)
- #563 - Code duplication (13 issues)
- #564 - Exhaustive switch statements (7 issues)
- #566 - Minor lint issues (10 issues)

### Node.js Updates

- #568 - Native Node.js 22+ features
- #569 - Add .nvmrc file
- #570 - Evaluate native test runner
- #573 - WiFi multi-adapter support

---

## Testing Strategy

### Unit Tests

- Test new `SetupTokenManager` token generation/validation
- Test error response formatting
- Test interface detection logic

### Integration Tests

- Test WebSocket authentication flow
- Test setup wizard with tokens
- Test profile application

### E2E Tests

- Full authentication flow
- WebSocket real-time updates
- Profile creation and application

---

## Git Workflow

```bash
# 1. Create branch
git checkout -b fix/issue-669-remove-getauthheaders

# 2. Make changes
# ... code changes ...

# 3. Run verification
make lint && make test

# 4. Commit with issue reference
git commit -m "fix(auth): remove deprecated getAuthHeaders usage

- Replace with credentials: 'include' for cookie auth
- Update all API fetch calls
- Remove unused function

Closes #669"

# 5. Push and create PR
git push -u origin fix/issue-669-remove-getauthheaders
gh pr create --title "fix(auth): remove deprecated getAuthHeaders" \
  --body "Closes #669" --base main
```

---

## Notes

- All P0 security issues have been addressed
- The most impactful P1 change remaining is #669 (getAuthHeaders removal)
- Interface-related issues (#571, #572, #574) should be done together
- Lower priority items (P3) can be addressed in future sprints
- All work follows `CLAUDE.md` guidelines
- Commits use conventional commit format
