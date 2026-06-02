/**
 * theme_colors.ts — color tokens for status indicators, severity ratings,
 * timing/perf phases, device categories, module accent colors, brand
 * highlights, gauge thresholds, discovery-method badges, and progress bars.
 * Re-exported through theme.ts.
 */

/**
 * Discovery method colors - for network discovery badges.
 * Backed by the categorical data-viz palette (--color-cat-*), which flips
 * light/dark automatically — no dark: variants needed.
 */
export const discoveryMethod = {
  arp: 'bg-cat-1/20 text-cat-1',
  ping: 'bg-cat-2/20 text-cat-2',
  ndp: 'bg-cat-3/20 text-cat-3',
  lldp: 'bg-cat-4/20 text-cat-4',
  cdp: 'bg-cat-5/20 text-cat-5',
  snmp: 'bg-cat-6/20 text-cat-6',
  edp: 'bg-cat-7/20 text-cat-7',
  mdns: 'bg-cat-8/20 text-cat-8',
} as const;

/**
 * Progress bar colors - for timing/performance visualization
 */
export const progressBar = {
  http: 'bg-cat-1',
  tcp: 'bg-cat-5',
  success: 'bg-status-success',
} as const;

/**
 * Status indicator variants - for connection status, health, etc.
 *
 * Composition surfaces (pick by what the call site needs):
 *   - text.*    — text color only
 *   - bg.*      — background at 100/20/10/5% alpha
 *   - border.*  — solid + 20% alpha borders
 *   - badge.*   — compound bg+text for chip/pill/banner patterns
 *   - color.*   — legacy alias for bg.{success,warning,error,info}
 *   - dot       — small circular indicator base class
 *   - withLabel — inline flex helper for "dot + label" pairings
 */
export const status = {
  dot: 'inline-block w-2 h-2 rounded-full',
  withLabel: 'inline-flex items-center gap-2',

  text: {
    success: 'text-status-success',
    warning: 'text-status-warning',
    error: 'text-status-error',
    info: 'text-status-info',
    muted: 'text-text-muted',
  },

  bg: {
    success: 'bg-status-success',
    warning: 'bg-status-warning',
    error: 'bg-status-error',
    info: 'bg-status-info',
    inactive: 'bg-surface-border',

    successStrong: 'bg-status-success/20',
    warningStrong: 'bg-status-warning/20',
    errorStrong: 'bg-status-error/20',
    infoStrong: 'bg-status-info/20',

    successSoft: 'bg-status-success/10',
    warningSoft: 'bg-status-warning/10',
    errorSoft: 'bg-status-error/10',
    infoSoft: 'bg-status-info/10',

    successSubtle: 'bg-status-success/5',
    warningSubtle: 'bg-status-warning/5',
    errorSubtle: 'bg-status-error/5',
    infoSubtle: 'bg-status-info/5',
  },

  border: {
    success: 'border-status-success',
    warning: 'border-status-warning',
    error: 'border-status-error',
    info: 'border-status-info',

    successSoft: 'border-status-success/20',
    warningSoft: 'border-status-warning/20',
    errorSoft: 'border-status-error/20',
    infoSoft: 'border-status-info/20',
  },

  badge: {
    success: 'bg-status-success/10 text-status-success',
    warning: 'bg-status-warning/10 text-status-warning',
    error: 'bg-status-error/10 text-status-error',
    info: 'bg-status-info/10 text-status-info',

    successStrong: 'bg-status-success/20 text-status-success',
    warningStrong: 'bg-status-warning/20 text-status-warning',
    errorStrong: 'bg-status-error/20 text-status-error',
    infoStrong: 'bg-status-info/20 text-status-info',
  },

  // Legacy alias retained for existing call sites that use status.color.X.
  // Mirrors status.bg.* for the four solid colors.
  color: {
    success: 'bg-status-success',
    warning: 'bg-status-warning',
    error: 'bg-status-error',
    info: 'bg-status-info',
    inactive: 'bg-surface-border',
  },
} as const;

/**
 * Severity colors - for CVE/vulnerability ratings (industry standard)
 * Critical = Red, High = Orange, Medium = Yellow, Low = Green
 */
export const severity = {
  critical: {
    bg: 'bg-status-error/15',
    text: 'text-status-error',
    border: 'border-status-error/30',
    dot: 'bg-status-error',
  },
  high: {
    bg: 'bg-severity-high/15',
    text: 'text-severity-high',
    border: 'border-severity-high/30',
    dot: 'bg-severity-high',
  },
  medium: {
    bg: 'bg-status-warning/15',
    text: 'text-status-warning',
    border: 'border-status-warning/30',
    dot: 'bg-status-warning',
  },
  low: {
    bg: 'bg-status-success/15',
    text: 'text-status-success',
    border: 'border-status-success/30',
    dot: 'bg-status-success',
  },
  info: {
    bg: 'bg-status-info/15',
    text: 'text-status-info',
    border: 'border-status-info/30',
    dot: 'bg-status-info',
  },
} as const;

/**
 * Timing/phase colors - for HTTP timing bars, performance metrics
 * Following industry conventions for network timing visualization
 */
export const timing = {
  dns: {
    bg: 'bg-cat-1',
    text: 'text-cat-1',
  },
  tcp: {
    bg: 'bg-cat-2',
    text: 'text-cat-2',
  },
  tls: {
    bg: 'bg-cat-6',
    text: 'text-cat-6',
  },
  wait: {
    bg: 'bg-cat-5',
    text: 'text-cat-5',
  },
  download: {
    bg: 'bg-cat-4',
    text: 'text-cat-4',
  },
} as const;

/**
 * Category colors - for device types, network segments
 */
export const category = {
  router: 'text-cat-1',
  server: 'text-cat-6',
  workstation: 'text-cat-4',
  printer: 'text-cat-5',
  mobile: 'text-cat-2',
  network: 'text-cat-7',
  unknown: 'text-text-muted',
} as const;

/**
 * Module colors - accent colors for The Seed's feature modules
 *
 * IMPORTANT: Use these for icons and small badges only, NOT for card backgrounds.
 * Cards should remain consistent (surface-raised) across all modules.
 *
 * Usage:
 * <RouteIcon className={moduleColor.path.icon} />
 * <span className={cn(moduleColor.wifi.badge, "px-2 py-1")}>Wi-Fi</span>
 */
export const moduleColor = {
  // Path - path analysis, traceroute, deep connectivity
  path: {
    icon: 'text-module-path', // Uses CSS variable
    badge: 'bg-module-path/20 text-module-path',
    border: 'border-module-path/30',
  },
  // Wi-Fi - visibility, surveys, coverage
  wifi: {
    icon: 'text-module-wifi', // Matches brand primary
    badge: 'bg-module-wifi/20 text-module-wifi',
    border: 'border-module-wifi/30',
  },
  // Security - posture, hardening
  security: {
    icon: 'text-module-security',
    badge: 'bg-module-security/20 text-module-security',
    border: 'border-module-security/30',
  },
  // Telemetry - live monitoring, data flow
  telemetry: {
    icon: 'text-module-telemetry',
    badge: 'bg-module-telemetry/20 text-module-telemetry',
    border: 'border-module-telemetry/30',
  },
  // Reporting - reports, compliance, exports
  reporting: {
    icon: 'text-module-reporting', // Matches brand gold
    badge: 'bg-module-reporting/20 text-module-reporting',
    border: 'border-module-reporting/30',
  },
} as const;

/**
 * Brand colors - for special brand elements
 *
 * Usage:
 * <span className={brand.gold.text}>Premium Feature</span>
 * <div className={brand.gold.badge}>PRO</div>
 */
export const brand = {
  // Mustard Gold - for premium/special highlights
  gold: {
    text: 'text-brand-gold',
    bg: 'bg-brand-gold',
    badge: 'bg-brand-gold/20 text-brand-gold',
    border: 'border-brand-gold/30',
  },
} as const;

/**
 * Gauge colors - for speed gauges, progress indicators
 * Returns CSS variable-compatible color based on percentage
 */
export const gauge = {
  getColor: (percentage: number): string => {
    if (percentage < 25) {
      return 'var(--color-status-error)';
    }
    if (percentage < 50) {
      return 'var(--color-status-warning)';
    }
    if (percentage < 75) {
      return 'var(--gauge-amber)';
    }
    return 'var(--color-status-success)';
  },
  // Tailwind class equivalents for non-SVG usage
  className: {
    critical: 'text-status-error',
    warning: 'text-status-warning',
    caution: 'text-cat-5',
    good: 'text-status-success',
  },
} as const;
