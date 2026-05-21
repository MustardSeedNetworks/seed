/**
 * theme_colors.ts — color tokens for status indicators, severity ratings,
 * timing/perf phases, device categories, module accent colors, brand
 * highlights, gauge thresholds, discovery-method badges, and progress bars.
 * Re-exported through theme.ts.
 */

/**
 * Discovery method colors - for network discovery badges
 * Use dark: variants for proper dark mode support
 */
export const discoveryMethod = {
  arp: 'bg-blue-500/20 text-blue-600 dark:text-blue-400',
  ping: 'bg-cyan-500/20 text-cyan-600 dark:text-cyan-400',
  ndp: 'bg-indigo-500/20 text-indigo-600 dark:text-indigo-400',
  lldp: 'bg-green-500/20 text-green-600 dark:text-green-400',
  cdp: 'bg-orange-500/20 text-orange-600 dark:text-orange-400',
  snmp: 'bg-purple-500/20 text-purple-600 dark:text-purple-400',
  edp: 'bg-teal-500/20 text-teal-600 dark:text-teal-400',
  mdns: 'bg-rose-500/20 text-rose-600 dark:text-rose-400',
} as const;

/**
 * Progress bar colors - for timing/performance visualization
 */
export const progressBar = {
  http: 'bg-blue-500 dark:bg-blue-400',
  tcp: 'bg-amber-500 dark:bg-amber-400',
  success: 'bg-green-500 dark:bg-green-400',
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
    bg: 'bg-orange-500/15 dark:bg-orange-400/15',
    text: 'text-orange-600 dark:text-orange-400',
    border: 'border-orange-500/30 dark:border-orange-400/30',
    dot: 'bg-orange-500 dark:bg-orange-400',
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
    bg: 'bg-blue-500 dark:bg-blue-400',
    text: 'text-blue-600 dark:text-blue-400',
  },
  tcp: {
    bg: 'bg-cyan-500 dark:bg-cyan-400',
    text: 'text-cyan-600 dark:text-cyan-400',
  },
  tls: {
    bg: 'bg-purple-500 dark:bg-purple-400',
    text: 'text-purple-600 dark:text-purple-400',
  },
  wait: {
    bg: 'bg-amber-500 dark:bg-amber-400',
    text: 'text-amber-600 dark:text-amber-400',
  },
  download: {
    bg: 'bg-green-500 dark:bg-green-400',
    text: 'text-green-600 dark:text-green-400',
  },
} as const;

/**
 * Category colors - for device types, network segments
 */
export const category = {
  router: 'text-blue-500 dark:text-blue-400',
  server: 'text-purple-500 dark:text-purple-400',
  workstation: 'text-green-500 dark:text-green-400',
  printer: 'text-orange-500 dark:text-orange-400',
  mobile: 'text-cyan-500 dark:text-cyan-400',
  network: 'text-teal-500 dark:text-teal-400',
  unknown: 'text-text-muted',
} as const;

/**
 * Module colors - accent colors for The Seed's feature modules
 *
 * IMPORTANT: Use these for icons and small badges only, NOT for card backgrounds.
 * Cards should remain consistent (surface-raised) across all modules.
 *
 * Usage:
 * <RootIcon className={moduleColor.roots.icon} />
 * <span className={cn(moduleColor.canopy.badge, "px-2 py-1")}>WiFi</span>
 */
export const moduleColor = {
  // Roots - Path Analysis, Traceroute, Deep Connectivity
  roots: {
    icon: 'text-module-roots', // Uses CSS variable
    badge: 'bg-module-roots/20 text-module-roots',
    border: 'border-module-roots/30',
  },
  // Canopy - WiFi Planning, Surveys, Coverage
  canopy: {
    icon: 'text-module-canopy', // Matches brand primary
    badge: 'bg-module-canopy/20 text-module-canopy',
    border: 'border-module-canopy/30',
  },
  // Shell - Security Posture, Hardening
  shell: {
    icon: 'text-module-shell',
    badge: 'bg-module-shell/20 text-module-shell',
    border: 'border-module-shell/30',
  },
  // Sap - Live Telemetry, Monitoring, Data Flow
  sap: {
    icon: 'text-module-sap',
    badge: 'bg-module-sap/20 text-module-sap',
    border: 'border-module-sap/30',
  },
  // Harvest - Reports, Compliance, Exports
  harvest: {
    icon: 'text-module-harvest', // Matches brand gold
    badge: 'bg-module-harvest/20 text-module-harvest',
    border: 'border-module-harvest/30',
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
      return 'var(--gauge-amber, #eab308)';
    }
    return 'var(--color-status-success)';
  },
  // Tailwind class equivalents for non-SVG usage
  className: {
    critical: 'text-status-error',
    warning: 'text-status-warning',
    caution: 'text-amber-500 dark:text-amber-400',
    good: 'text-status-success',
  },
} as const;
