/**
 * msw handlers for the provider bootstrap Storybook performs on every story.
 *
 * Storybook has no daemon behind it, so these calls used to hit the dev
 * server's HTML fallback and React Query tried to parse `<!doctype` as JSON —
 * 60 error and warning lines in a passing run (#2203). Expected and unexpected
 * failures were indistinguishable, which is how #2201 hid: a real crash in
 * VulnerabilityDetailsModal that was unreachable only because parsing failed
 * before the data existed.
 *
 * Shapes come from the Go handlers, not from guesses:
 *   ProfileListResponse   internal/api/handlers_profiles.go
 *   LicenseStatusResponse internal/api/tokens.go
 *   UserResponse          internal/api/users.go
 * and settings-defaults.json is `config.GetDefaultSettings()` marshalled by
 * the Go type itself, so it cannot drift into a shape the backend never sends.
 */
import { HttpResponse, http } from 'msw';
import settingsDefaults from './settings-defaults.json';

/** Matches internal/api/handlers_profiles.go ProfileResponse. */
const defaultProfile = {
  id: 'profile-default',
  name: 'Default',
  description: 'Storybook fixture profile',
  config: {},
  isDefault: true,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

export const handlers = [
  // ProfileListResponse: { profiles, total }
  http.get('*/api/v1/profiles', () => HttpResponse.json({ profiles: [defaultProfile], total: 1 })),
  http.get('*/api/v1/profiles/active', () => HttpResponse.json(defaultProfile)),

  // config.GetDefaultSettings(), marshalled by the Go type.
  http.get('*/api/v1/settings/defaults', () => HttpResponse.json(settingsDefaults)),

  // LicenseStatusResponse. Free is the honest default for a story: it is the
  // unlicensed state, which the backend treats as a valid result rather than
  // an error, and it keeps tier-gated UI in its most restricted form unless a
  // story overrides this handler deliberately.
  http.get('*/api/v1/license', () =>
    HttpResponse.json({
      tier: 'Free',
      tierValue: 0,
      isTrialMode: false,
      canMintTokens: false,
      activated: false,
    }),
  ),

  // SSOProvidersResponse: { providers } — internal/api/oauth.go. Empty is the
  // real default: providers only appear once one is configured and enabled.
  http.get('*/api/v1/sso/providers', () => HttpResponse.json({ providers: [] })),

  // The discovery/vulnerability pair useNetworkDiscoveryAutoScan reads. Both
  // are capitalised on the wire; the hook documents that inline and reads
  // `options.PortScan.Enabled` and `Enabled` / `AutoScan`.
  http.get('*/api/v1/security/discovery/options', () =>
    HttpResponse.json({ options: { PortScan: { Enabled: false } } }),
  ),
  http.get('*/api/v1/security/vulnerabilities/settings', () =>
    HttpResponse.json({ Enabled: false, AutoScan: false }),
  ),

  // DeviceVulnerabilities — src/types/vulnerabilities.ts. An empty list is a
  // device that scanned clean, which is a real state rather than an error, and
  // it is the state #2201 could never reach: parsing failed before `data`
  // existed, so `data?.vulnerabilities.length` was never evaluated.
  http.get('*/api/v1/security/vulnerabilities/device', ({ request }) =>
    HttpResponse.json({
      deviceIp: new URL(request.url).searchParams.get('ip') ?? '192.168.1.100',
      mac: '00:11:22:33:44:55',
      hostname: 'storybook-device',
      vendor: 'Storybook',
      product: 'Fixture',
      version: '1.0.0',
      vulnerabilities: [],
      scanTime: '2026-01-01T00:00:00Z',
    }),
  ),

  // DiscoveryResponse: { devices, status } — src/hooks/useDiscoveredDevices.ts.
  http.get('*/api/v1/security/devices', () =>
    HttpResponse.json({
      devices: [],
      status: { scanning: false, deviceCount: 0, lastScan: '2026-01-01T00:00:00Z' },
    }),
  ),

  // DiscoveryServiceStatus — src/types/settings.ts. rescanInterval is a Go
  // time.Duration on the wire, so nanoseconds: 300e9 is five minutes.
  http.get('*/api/v1/security/discovery/service/status', () =>
    HttpResponse.json({
      running: false,
      scanning: false,
      deviceCount: 0,
      lastScan: '2026-01-01T00:00:00Z',
      subnet: '192.168.1.0/24',
      localIP: '192.168.1.10',
      interface: 'eth0',
      activeMethods: [],
      rescanInterval: 300_000_000_000,
    }),
  ),

  // The telemetry PerformanceCard polls. Idle is the resting state.
  http.get('*/api/v1/telemetry/speedtest/status', () =>
    HttpResponse.json({ running: false, progress: 0 }),
  ),
  http.get('*/api/v1/telemetry/iperf/info', () =>
    HttpResponse.json({ installed: false, version: '' }),
  ),
  http.get('*/api/v1/telemetry/iperf/client/status', () => HttpResponse.json({ running: false })),
  http.get('*/api/v1/telemetry/iperf/server/status', () => HttpResponse.json({ running: false })),

  // UpdateStatusResponse — internal/api/update.go, embedding update.UpdateStatus.
  http.get('*/api/v1/updates/status', () =>
    HttpResponse.json({
      state: 'idle',
      progress: 0,
      message: '',
      error: '',
      downloadedBytes: 0,
      totalBytes: 0,
      startedAt: '',
      updateReady: false,
      requiresAction: false,
    }),
  ),

  // UpdateConfigResponse — same file.
  http.get('*/api/v1/updates/config', () =>
    HttpResponse.json({
      enabled: false,
      checkInterval: '24h',
      autoDownload: false,
      autoApply: false,
      includePrerelease: false,
    }),
  ),

  // GET /api/v1/settings returns the live settings object. The settings-drawer
  // loaders read slices of it -- thresholds, tests, snmp, ipconfig -- and the
  // defaults document carries exactly those keys, so it stands in without
  // inventing a second shape that could drift from the first.
  http.get('*/api/v1/settings', () => HttpResponse.json(settingsDefaults)),

  // The Wi-Fi settings path really does double the segment; the route is
  // registered as /wifi/wifi/settings in internal/api/server_routes.go, so the
  // client and server agree and this mirrors them rather than correcting one.
  http.get('*/api/v1/wifi/wifi/settings', () =>
    HttpResponse.json({ enabled: false, interface: '', scanIntervalSeconds: 0 }),
  ),

  // The rest of the settings-drawer loaders. Each is read as a slice of the
  // defaults document, which carries the same keys, so these reuse it rather
  // than inventing a parallel shape that could drift from it.
  http.get('*/api/v1/settings/cable', () => HttpResponse.json(settingsDefaults.cableTest)),
  http.get('*/api/v1/settings/link', () => HttpResponse.json(settingsDefaults.link)),
  http.get('*/api/v1/telemetry/snmp/settings', () => HttpResponse.json(settingsDefaults.snmp)),
  http.get('*/api/v1/telemetry/ipconfig/settings', () => HttpResponse.json({})),
  http.get('*/api/v1/telemetry/probes/settings', () => HttpResponse.json({})),
  http.get('*/api/v1/telemetry/iperf/suggestions', () => HttpResponse.json({ suggestions: [] })),
  http.get('*/api/v1/security/devices/settings', () =>
    HttpResponse.json(settingsDefaults.networkDiscovery),
  ),
  // guestaudit.Settings. Disabled with no targets is the shipped default and
  // the state GuestNetworkAuditSettings exists to move an operator out of.
  http.get('*/api/v1/security/guest-audit/settings', () =>
    HttpResponse.json({ enabled: false, targets: [] }),
  ),

  // StatusResponse. usePlatformCapabilities reads the capabilities array from
  // here (#749/#750), and without a handler every story that renders a gated
  // surface logged "Failed to read platform capabilities" and failed the
  // console gate.
  //
  // Linux's row set, because that is the platform with nothing degraded — a
  // story should show the feature rather than the "not available here" notice
  // unless it overrides this deliberately.
  http.get('*/api/v1/status', () =>
    HttpResponse.json({
      status: 'ok',
      version: '0.0.0-storybook',
      uptime: 0,
      interface: 'eth0',
      isWireless: false,
      icmpAvailable: true,
      capabilities: [
        { capability: 'interface_listing', title: 'Interface listing', level: 'full' },
        { capability: 'static_ip', title: 'Static IP configuration', level: 'full' },
        { capability: 'dhcp_config', title: 'DHCP configuration', level: 'full' },
        { capability: 'mtu_config', title: 'MTU configuration', level: 'full' },
        { capability: 'link_monitoring', title: 'Link status monitoring', level: 'full' },
        { capability: 'speed_duplex', title: 'Speed/duplex detection', level: 'full' },
        { capability: 'wifi_scanning', title: 'Wi-Fi scanning', level: 'full' },
        { capability: 'wifi_connection', title: 'Wi-Fi connect/disconnect', level: 'full' },
        { capability: 'arp_table', title: 'ARP table reading', level: 'full' },
        { capability: 'ndp_discovery', title: 'IPv6 NDP discovery', level: 'full' },
        { capability: 'bluetooth_scanning', title: 'Bluetooth scanning', level: 'full' },
        { capability: 'gateway_detection', title: 'Gateway detection', level: 'full' },
        { capability: 'dns_detection', title: 'DNS server detection', level: 'full' },
        { capability: 'dhcp_lease_info', title: 'DHCP lease info', level: 'full' },
        { capability: 'vlan_detection', title: 'VLAN detection', level: 'full' },
        { capability: 'vlan_management', title: 'VLAN creation/deletion', level: 'full' },
        {
          capability: 'cable_diagnostics',
          title: 'Cable diagnostics (TDR)',
          level: 'partial',
          note: "Needs a NIC driver that implements ethtool's cable test.",
        },
        { capability: 'phy_info', title: 'PHY layer info', level: 'full' },
        {
          capability: 'optical_monitoring',
          title: 'Digital Optical Monitoring',
          level: 'partial',
          note: 'Needs an SFP/QSFP transceiver that reports diagnostics over ethtool.',
        },
        { capability: 'driver_statistics', title: 'Driver error counters', level: 'full' },
      ],
    }),
  ),

  // The driver counters the card curates (#416).
  http.get('*/api/v1/telemetry/interface/driver-stats', () =>
    HttpResponse.json({
      interface: 'eth0',
      total: 84,
      counters: [
        {
          key: 'rx_crc_errors',
          label: 'CRC errors',
          value: 0,
          meaning:
            'Frames arrived corrupted. Usually cabling, a bad port, or a duplex mismatch — not congestion.',
        },
      ],
    }),
  ),

  http.get('*/api/v1/reporting/logs', () => HttpResponse.json({ logs: [], total: 0 })),

  // useSubnetSettings expects a bare array (it checks Array.isArray and falls
  // back to [] otherwise), not an envelope.
  http.get('*/api/v1/security/devices/subnets', () => HttpResponse.json([])),

  // UserResponse for the current user.
  http.get('*/api/v1/users/me', () =>
    HttpResponse.json({
      id: 1,
      username: 'storybook',
      role: 'admin',
      isActive: true,
      authProvider: 'local',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    }),
  ),
];
