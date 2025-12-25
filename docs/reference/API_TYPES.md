# Seed API Type Contracts

This document defines the exact types exchanged between the frontend and backend for settings-related APIs.
It serves as the source of truth for type contracts and should be kept in sync with both:
- Backend: `internal/config/` and `internal/api/handlers_*.go`
- Frontend: `web/src/types/settings.ts`

## Authentication

All API endpoints (except `/api/auth/login` and `/api/health`) require JWT authentication via:
- `Authorization: Bearer <token>` header, OR
- `auth_token` cookie (set automatically on login)

Error responses for auth failures:
```json
{
  "error": "Unauthorized",
  "code": "AUTH_UNAUTHORIZED"
}
```

---

## Settings Endpoints

### GET/PUT `/api/settings`

General settings endpoint for thresholds, health checks, speedtest, iperf, and display options.

#### Response Type (GET)

```typescript
interface SettingsResponse {
  interface: {
    current: string;
    available: string[];
  };
  vlan: {
    enabled: boolean;
    id: number;
  };
  ip: {
    mode: "dhcp" | "static";
  };
  thresholds: ThresholdsSettings;
  healthChecks: {
    runPerformance: boolean;
    runSpeedtest: boolean;
    runIperf: boolean;
    runDiscovery: boolean;
  };
  speedtest: {
    serverId: string;
    autoRunOnLink: boolean;
  };
  iperf: IperfSettings;
  cardSettings: CardSettingsMap;
  displayOptions: DisplayOptions;
}
```

#### Request Type (PUT)

Partial updates supported. Only include fields you want to change.

```typescript
interface SettingsUpdateRequest {
  thresholds?: Partial<ThresholdsSettings>;
  healthChecks?: Partial<HealthChecksSettings>;
  speedtest?: Partial<SpeedtestSettings>;
  iperf?: Partial<IperfSettings>;
  fabOptions?: Partial<FABOptionsSettings>;
  displayOptions?: Partial<DisplayOptions>;
}
```

---

### GET/PUT `/api/settings/link`

Link speed/duplex configuration.

#### Response/Request Type

```typescript
// Backend JSON field names (snake_case)
interface LinkSettingsAPI {
  auto_negotiation: boolean;
  speed: "auto" | "10" | "100" | "1000" | "2500" | "5000" | "10000";
  duplex: "auto" | "full" | "half";
  available_modes: string[];
}

// Frontend TypeScript names (camelCase)
interface LinkSettings {
  autoNegotiation: boolean;
  speed: LinkSpeed;
  duplex: DuplexMode;
  availableModes: string[];
}

type LinkSpeed = "auto" | "10" | "100" | "1000" | "2500" | "5000" | "10000";
type DuplexMode = "auto" | "full" | "half";
```

**Mapping**: Frontend converts camelCase to snake_case when sending, and vice versa when receiving.

---

### GET/PUT `/api/settings/cable`

Cable test (TDR) settings.

#### Response/Request Type

```typescript
// Backend JSON field names (snake_case)
interface CableTestSettingsAPI {
  enabled: boolean;
  auto_run_on_link_down: boolean;
}

// Frontend TypeScript names (camelCase)
interface CableTestSettings {
  enabled: boolean;
  autoRunOnLinkDown: boolean;
}
```

---

### GET/PUT `/api/snmp/settings`

SNMP configuration for device interrogation.

#### Response/Request Type

```typescript
// Backend JSON (snake_case)
interface SNMPSettingsAPI {
  communities: string[];
  v3_credentials: SNMPv3CredentialAPI[];
  timeout: number;  // milliseconds
  retries: number;
  port: number;
}

interface SNMPv3CredentialAPI {
  name: string;
  username: string;
  auth_protocol?: "MD5" | "SHA";  // MD5 deprecated - shows warning
  auth_password?: string;
  priv_protocol?: "DES" | "AES";
  priv_password?: string;
  context_name?: string;
  security_level?: "noAuthNoPriv" | "authNoPriv" | "authPriv";
}

// Frontend TypeScript (camelCase)
interface SNMPSettings {
  communities: string[];
  v3Credentials: SNMPv3Credential[];
  timeout: number;
  retries: number;
  port: number;
}
```

---

### GET/PUT `/api/wifi/settings`

WiFi interface selection.

#### Response/Request Type

```typescript
interface WifiSettingsAPI {
  interface: string;
}

interface WifiSettings {
  interface: string;
}
```

---

### GET/PUT `/api/devices/settings`

Network discovery settings.

#### Response Type

```typescript
interface NetworkDiscoverySettingsAPI {
  enabled: boolean;
  auto_scan: boolean;
  scan_interval_secs: number;
  additional_subnets: SubnetAPI[];
  fingerprinting: FingerprintingConfigAPI;
  ipv6_enabled: boolean;
}

interface SubnetAPI {
  cidr: string;
  name?: string;
  enabled: boolean;
}

interface FingerprintingConfigAPI {
  enabled: boolean;
  os_detection: boolean;
  service_probes: boolean;
}
```

---

## Threshold Types

Used in `/api/settings` thresholds field.

```typescript
interface ThresholdsSettings {
  dns: ThresholdPair;
  gateway: ThresholdPair;
  wifi: WiFiThreshold;
  customPing: ThresholdPair;
  customTcp: ThresholdPair;
  customHttp: ThresholdPair;
  httpTimings: HTTPTimingThresholds;
}

interface ThresholdPair {
  good: number;     // milliseconds - "good" threshold
  warning: number;  // milliseconds - "warning" threshold (becomes critical)
}

interface WiFiThreshold {
  good: number;     // dBm - good signal threshold
  warning: number;  // dBm - warning signal threshold
}

interface HTTPTimingThresholds {
  dns: ThresholdPair;
  tcp: ThresholdPair;
  tls: ThresholdPair;
  ttfb: ThresholdPair;
}
```

---

## Display Options

```typescript
interface DisplayOptions {
  showPublicIP: boolean;
  unitSystem: "metric" | "sae";
}
```

---

## iPerf Settings

```typescript
interface IperfSettings {
  autoRunOnLink: boolean;
  server: string;
  port: number;
  protocol: "tcp" | "udp";
  direction: "upload" | "download" | "both";
  duration: number;
  serverPort: number;
  enableServer: boolean;
}
```

---

## FAB Options

Controls which tests run when the Floating Action Button is pressed.

```typescript
interface FABOptionsSettings {
  runLink: boolean;
  runSwitch: boolean;
  runVLAN: boolean;
  runIPConfig: boolean;
  runGateway: boolean;
  runDNS: boolean;
  runHealthChecks: boolean;
  runNetworkDiscovery: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runPerformance: boolean;
  autoScanOnLink: boolean;
}
```

---

## Card Settings

Per-card visibility and auto-run settings.

```typescript
interface CardSettings {
  visible: boolean;
  autoRunOnLink: boolean;
}

interface CardSettingsMap {
  link: CardSettings;
  switch: CardSettings;
  vlan: CardSettings;
  network: CardSettings;
  gateway: CardSettings;
  dns: CardSettings;
  healthChecks: CardSettings;
  networkDiscovery: CardSettings;
  performance: PerformanceCardSettings;
}

interface PerformanceCardSettings extends CardSettings {
  speedtest: {
    enabled: boolean;
    autoRunOnLink: boolean;
  };
  iperf: {
    enabled: boolean;
    autoRunOnLink: boolean;
  };
}
```

---

## Error Response Format

All API errors follow this format:

```typescript
interface APIError {
  error: string;      // Human-readable message
  code: ErrorCode;    // Machine-readable code
  details?: string;   // Optional additional details
}

type ErrorCode =
  | "AUTH_UNAUTHORIZED"
  | "AUTH_TOKEN_EXPIRED"
  | "VALIDATION_ERROR"
  | "BAD_REQUEST"
  | "METHOD_NOT_ALLOWED"
  | "INTERNAL_ERROR"
  | "RATE_LIMITED";
```

---

## Request Body Size Limits

Defined in `internal/api/limits.go`:

| Endpoint Type | Limit |
|--------------|-------|
| Auth (login) | 1 KB |
| Config/Settings | 64 KB |
| General JSON | 256 KB |
| Floor Plan Upload | 10 MB |
| AirMapper Import | 50 MB |
| Default | 1 MB |

---

## Profile Settings

Profile-specific settings stored in database.

```typescript
interface ProfileSettings {
  version: number;
  interfaces?: ProfileInterfaceConfigs;
  thresholds?: ProfileThresholds;
  healthChecks?: ProfileHealthChecks;
  speedtest?: ProfileSpeedtest;
  iperf?: ProfileIperf;
  fabOptions?: ProfileFABOptions;
  displayOptions?: ProfileDisplayOptions;
  dns?: ProfileDNS;
  snmp?: ProfileSNMP;
  networkDiscovery?: ProfileNetworkDiscovery;
  link?: ProfileLinkSettings;
  cableTest?: ProfileCableTestSettings;
  notes?: string;
}
```

---

## Backend Go Types Location

| Type | File |
|------|------|
| ProfileSettings | `internal/config/profile_settings.go` |
| Config | `internal/config/config.go` |
| API Handlers | `internal/api/handlers_settings.go` |
| Body Limits | `internal/api/limits.go` |
| Error Codes | `internal/api/response.go` |

## Frontend TypeScript Types Location

| Type | File |
|------|------|
| Settings | `web/src/types/settings.ts` |
| API Responses | `web/src/types/api.ts` |

---

## Naming Convention

- **Backend (Go/JSON)**: `snake_case` for JSON field names
- **Frontend (TypeScript)**: `camelCase` for property names
- **Mapping**: Frontend code converts between conventions when making API calls

Example:
```typescript
// Sending to backend
const apiData = {
  auto_negotiation: settings.autoNegotiation,
  auto_run_on_link_down: settings.autoRunOnLinkDown
};

// Receiving from backend
const frontendData = {
  autoNegotiation: response.auto_negotiation,
  autoRunOnLinkDown: response.auto_run_on_link_down
};
```
