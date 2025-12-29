import { twMerge } from "tailwind-merge";

/**
 * =============================================================================
 * THE SEED DESIGN SYSTEM - Mustard Seed Networks
 * =============================================================================
 *
 * Centralized design tokens and utilities for consistent UI across the app.
 *
 * ARCHITECTURE:
 * 1. CSS Variables (index.css) - Core color tokens for light/dark modes
 * 2. This file (theme.ts) - TypeScript tokens and utility functions
 * 3. Tailwind Classes - CSS-first configuration using @theme directive
 *
 * BRAND COLORS:
 * - Primary: Seed Green (#2d7a3e / #81c784 dark) - Actions, links, focus states
 * - Accent: Lighter Seed Green (#4caf50 / #a5d6a7 dark) - Hover states
 * - Gold: Mustard Gold (#d4a017 / #fbbf24 dark) - Special highlights, premium
 *
 * STATUS COLORS (Industry Standard - DO NOT CHANGE):
 * - Success: Green (#28a745) - Positive states
 * - Warning: Amber (#ffc107) - Caution states
 * - Error: Red (#dc3545) - Error/danger states
 * - Info: Cyan (#17a2b8) - Informational states
 *
 * MODULE COLORS (for icons/badges only, not backgrounds):
 * - Roots: Amber/Brown - Path analysis, foundation
 * - Canopy: Green - WiFi planning, coverage
 * - Shell: Orange - Security, protection
 * - Sap: Cyan - Telemetry, data flow
 * - Harvest: Gold - Reports, results
 *
 * USAGE:
 * import { spacing, button, cn, moduleColor } from '../styles/theme';
 * <button className={cn(button.base, button.variant.primary)}>Action</button>
 *
 * =============================================================================
 */

// ============================================================================
// SPACING SCALE
// ============================================================================
// Use Tailwind's spacing scale: 1 unit = 0.25rem (4px)
// Common values:
// - 0.5 = 2px (tight spacing)
// - 1 = 4px (minimal)
// - 2 = 8px (compact)
// - 3 = 12px (default)
// - 4 = 16px (comfortable)
// - 6 = 24px (spacious)
// - 8 = 32px (section separation)
// - 12 = 48px (major sections)

/**
 * Spacing scale - based on 4px grid
 * Use these semantic spacing utilities for consistency.
 * CSS utility classes are defined in index.css (@layer components)
 */
export const spacing = {
  // Raw Tailwind values (for reference)
  values: {
    xs: "1", // 4px
    sm: "2", // 8px
    default: "3", // 12px
    md: "4", // 16px
    lg: "6", // 24px
    xl: "8", // 32px
    "2xl": "12", // 48px
  },

  // Semantic CSS utility classes (preferred - use these)
  stack: {
    xs: "stack-xs", // 4px vertical
    sm: "stack-sm", // 8px vertical
    default: "stack", // 12px vertical
    lg: "stack-lg", // 16px vertical
    xl: "stack-xl", // 24px vertical
  },

  section: {
    default: "section-gap", // 24px between sections
    lg: "section-gap-lg", // 32px for page-level
  },

  gap: {
    tight: "gap-tight", // 4px
    compact: "gap-compact", // 8px
    default: "gap-default", // 12px
    comfortable: "gap-comfortable", // 16px
    spacious: "gap-spacious", // 24px
  },

  inline: {
    xs: "inline-gap-xs", // 4px
    sm: "inline-gap-sm", // 6px
    default: "inline-gap", // 8px
    lg: "inline-gap-lg", // 12px
  },

  pad: {
    xs: "pad-xs", // 8px
    sm: "pad-sm", // 12px
    default: "pad", // 16px
    lg: "pad-lg", // 24px
    xl: "pad-xl", // 32px
  },

  // Chip/pill padding (for tags, badges, small interactive elements)
  chip: {
    sm: "chip-pad", // px-3 py-1
    md: "chip-pad-md", // px-3 py-1.5
    lg: "chip-pad-lg", // px-3 py-2
  },

  // Tab button padding
  tab: "tab-pad", // py-2.5 px-3

  // Semantic margin utilities (CSS classes from index.css)
  margin: {
    bottom: {
      section: "mb-section", // 24px - between major sections
      sectionLg: "mb-section-lg", // 32px - large section gaps
      heading: "mb-heading", // 12px - after headings
      content: "mb-content", // 16px - after content blocks
      inline: "mb-2", // 8px - inline content (small bottom margins)
      tight: "mb-tight", // 4px - very tight, for labels
    },
    top: {
      section: "mt-section", // 32px - before major sections
      content: "mt-content", // 16px - content separation
      heading: "mt-heading", // 12px - before headings/after inline
      inline: "mt-inline", // 8px - inline content
      tight: "mt-tight", // 4px - very tight, for form fields
    },
    left: {
      tight: "ml-tight", // 4px - minimal left margin
      inline: "ml-inline", // 8px - inline content
      content: "ml-content", // 16px - content indentation
      spacious: "ml-spacious", // 24px - large indentation (lists)
    },
  },

  // Padding utilities for dividers/sections
  padding: {
    top: {
      heading: "pt-heading", // 12px - section divider top
      section: "pt-section", // 16px - section divider with more space
      tight: "pt-tight", // 4px - minimal top padding for subtle spacing
    },
    bottom: {
      inline: "pb-inline", // 8px - inline bottom padding
      tight: "pb-tight", // 4px - minimal bottom padding
    },
    right: {
      icon: "pr-icon", // 40px - space for right-positioned icon
      tight: "pr-tight", // 32px - slightly smaller right padding
    },
  },

  // Centered content padding (for loading states, empty states)
  centered: "py-centered", // 48px vertical

  // Compact action button padding (remove, delete buttons)
  actionBtn: "action-btn-pad", // 4px horizontal

  // Micro spacing (for fine-grained adjustments)
  micro: {
    gap: "gap-micro", // 2px - very tight badge/icon spacing
    mt: "mt-micro", // 2px - minimal top margin for icon alignment
    mtNeg: "-mt-tight", // -4px - negative margin for tight visual alignment
    pb: "pb-micro", // 2px - minimal bottom padding
    pbCompact: "pb-compact", // 4px - compact bottom padding
    pbCompactMd: "pb-compact-md", // 6px - medium compact bottom padding
    mtCompactMd: "mt-compact-md", // 6px - medium compact top margin
  },

  // Badge/status indicator padding
  badge: {
    xs: "p-badge-xs", // 2px - extra small badge padding
    sm: "p-badge-sm", // 4px - small badge padding
    padXs: "badge-pad-xs", // px-2 py-0.5 - compact status badge
  },

  // Keyboard key styling
  kbd: "kbd-pad", // px-1 py-0.5 - keyboard key padding

  // Left padding for indentation
  indent: "pl-indent", // 20px - nested content indentation

  // Compact vertical padding (for list items, table cells)
  compact: {
    py: "py-compact", // 4px - compact list item
    pyMd: "py-compact-md", // 6px - slightly larger compact
  },

  // Row/cell padding
  row: {
    py: "py-row", // 8px - table cell/row padding
    pyLg: "py-row-lg", // 12px - larger row padding
  },

  // Cell horizontal padding
  cell: {
    px: "px-cell", // 8px - table cell horizontal padding
  },

  // Icon button padding
  iconBtn: {
    sm: "p-icon-btn", // 4px - compact icon button
    md: "p-icon-btn-md", // 6px - medium icon button
  },

  // Main content layout padding
  mainPadding: {
    y: "main-padding-y", // py-4 sm:py-6
    x: "content-padding-x", // px-4 sm:px-6 lg:px-8
  },

  // Header padding (responsive)
  headerPadding: {
    y: "header-padding-y", // py-2 sm:py-3
  },

  // Drawer/panel content padding
  drawerPad: "drawer-content-pad", // px-4 sm:px-5 pb-10 pt-4

  // Table cell padding
  tableCell: {
    empty: "table-cell-empty", // px-2 py-4 - empty state cells
  },
} as const;

// ============================================================================
// TYPOGRAPHY
// ============================================================================
// CSS utility classes are defined in index.css (@layer components)
// Use these TypeScript constants for programmatic styling or documentation

export const typography = {
  // Semantic heading classes (match CSS utilities in index.css)
  // These are the preferred way to style headings
  heading: {
    h1: "heading-1", // Page titles: 24px/30px bold
    h2: "heading-2", // Section/modal titles: 20px/24px semibold
    h3: "heading-3", // Card titles: 18px/20px semibold
    h4: "heading-4", // Subsections: 16px/18px medium
    section: "section-title", // Category labels: 12px uppercase muted
  },

  // Body text variants
  body: {
    large: "body-large", // 18px primary
    default: "body", // 16px primary (most common)
    small: "body-small", // 14px secondary
    caption: "caption", // 12px muted (metadata)
  },

  // Utility classes
  label: "label", // Form labels: 14px medium
  code: "code", // Monospace with background

  // Raw size classes (use sparingly - prefer semantic variants above)
  size: {
    xs: "text-xs", // 12px
    sm: "text-sm", // 14px
    base: "text-base", // 16px
    lg: "text-lg", // 18px
    xl: "text-xl", // 20px
    "2xl": "text-2xl", // 24px
    "3xl": "text-3xl", // 30px
  },

  // Font weights
  weight: {
    normal: "font-normal", // 400
    medium: "font-medium", // 500
    semibold: "font-semibold", // 600
    bold: "font-bold", // 700
  },

  // Font families
  family: {
    body: "font-body", // Inter
    display: "font-display", // Inter (display variant)
    mono: "font-mono", // JetBrains Mono
  },

  // Line heights
  leading: {
    tight: "leading-tight", // 1.25 - headings
    snug: "leading-snug", // 1.375 - subheadings
    normal: "leading-normal", // 1.5 - default
    relaxed: "leading-relaxed", // 1.625 - body text
  },
} as const;

// ============================================================================
// COMPONENT VARIANTS
// ============================================================================

/**
 * Button variants - consistent button styling across the app
 */
export const button = {
  base: "inline-flex items-center justify-center gap-2 rounded font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed",

  variant: {
    primary: "bg-brand-primary text-text-inverse hover:bg-brand-accent",
    secondary: "border border-surface-border bg-surface-raised hover:bg-surface-hover",
    ghost: "hover:bg-surface-hover",
    danger: "bg-status-error text-text-inverse hover:opacity-90",
    success: "bg-status-success text-text-inverse hover:opacity-90",
  },

  size: {
    xs: "px-2 py-1 text-xs", // Tiny buttons, badges
    sm: "px-3 py-1.5 text-sm", // Small buttons
    md: "px-4 py-2 text-base", // Default buttons
    lg: "px-6 py-3 text-lg", // Large CTAs
  },
} as const;

/**
 * Input variants - consistent form input styling
 */
export const input = {
  base: "w-full rounded border bg-surface-raised text-text-primary transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed",

  state: {
    default: "border-surface-border",
    error: "border-status-error",
    success: "border-status-success",
  },

  size: {
    sm: "px-2 py-1.5 text-sm", // Compact inputs
    md: "px-2.5 py-2 text-sm", // Default inputs (most common)
    lg: "px-3 py-2.5 text-base", // Large inputs
  },
} as const;

/**
 * Card variants - consistent card styling
 */
export const card = {
  base: "rounded-lg border bg-surface-raised",

  variant: {
    default: "border-surface-border",
    elevated: "border-surface-border shadow-lg",
    interactive:
      "border-surface-border hover:border-brand-primary cursor-pointer transition-colors",
  },

  padding: {
    none: "",
    sm: "p-3",
    md: "p-4",
    lg: "p-6",
  },
} as const;

/**
 * Badge/Chip variants - for status indicators
 */
export const badge = {
  base: "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium",

  variant: {
    default: "bg-surface-hover text-text-primary",
    success: "bg-status-success/10 text-status-success",
    warning: "bg-status-warning/10 text-status-warning",
    error: "bg-status-error/10 text-status-error",
    info: "bg-status-info/10 text-status-info",
    primary: "bg-brand-primary/10 text-brand-primary",
  },
} as const;

/**
 * Toast/Notification variants - for non-modal notifications
 */
export const toast = {
  container: "px-4 py-3 shadow-lg",
  animation: "animate-slide-in",
} as const;

/**
 * Alert/Banner variants - for inline messages
 * Use these for error/warning/info banners instead of hardcoded colors
 */
export const alert = {
  base: "px-4 py-3 rounded-lg border",

  variant: {
    error:
      "bg-status-error/10 border-status-error/20 text-status-error dark:bg-status-error/15 dark:border-status-error/30",
    warning:
      "bg-status-warning/10 border-status-warning/20 text-status-warning dark:bg-status-warning/15 dark:border-status-warning/30",
    success:
      "bg-status-success/10 border-status-success/20 text-status-success dark:bg-status-success/15 dark:border-status-success/30",
    info: "bg-status-info/10 border-status-info/20 text-status-info dark:bg-status-info/15 dark:border-status-info/30",
  },
} as const;

/**
 * Discovery method colors - for network discovery badges
 * Use dark: variants for proper dark mode support
 */
export const discoveryMethod = {
  arp: "bg-blue-500/20 text-blue-600 dark:text-blue-400",
  icmp: "bg-cyan-500/20 text-cyan-600 dark:text-cyan-400",
  ndp: "bg-indigo-500/20 text-indigo-600 dark:text-indigo-400",
  lldp: "bg-green-500/20 text-green-600 dark:text-green-400",
  cdp: "bg-orange-500/20 text-orange-600 dark:text-orange-400",
  snmp: "bg-purple-500/20 text-purple-600 dark:text-purple-400",
  edp: "bg-teal-500/20 text-teal-600 dark:text-teal-400",
  mdns: "bg-rose-500/20 text-rose-600 dark:text-rose-400",
} as const;

/**
 * Progress bar colors - for timing/performance visualization
 */
export const progressBar = {
  http: "bg-blue-500 dark:bg-blue-400",
  tcp: "bg-amber-500 dark:bg-amber-400",
  success: "bg-green-500 dark:bg-green-400",
} as const;

/**
 * Sizing tokens - for consistent heights/widths
 * Avoids arbitrary values like h-[500px] or w-[28rem]
 */
export const sizing = {
  // Modal/drawer heights (viewport-relative)
  height: {
    modal: "max-h-modal", // 85vh - defined in @theme
    drawer: "h-full", // Full height for side drawers
    panel: "max-h-[70vh]", // Shorter panels
  },

  // Fixed widths for drawers, panels
  width: {
    drawer: "w-80", // 320px - standard drawer
    drawerWide: "w-96", // 384px - wide drawer
    panel: "w-72", // 288px - side panels
    dropdown: "w-64", // 256px - dropdown menus
  },

  // Min/max constraints
  minHeight: {
    card: "min-h-[120px]", // Minimum card height
    section: "min-h-[200px]", // Minimum section height
  },
} as const;

/**
 * Modal/Dialog variants
 */
export const modal = {
  overlay: "fixed inset-0 z-50 flex items-center justify-center p-4",
  backdrop: "absolute inset-0 bg-black/50 backdrop-blur-sm",
  content:
    "bg-surface-raised border border-surface-border rounded-lg shadow-xl max-h-modal overflow-y-auto",

  size: {
    sm: "max-w-md w-full",
    md: "max-w-2xl w-full",
    lg: "max-w-4xl w-full",
    xl: "max-w-6xl w-full",
    full: "max-w-7xl w-full",
  },

  padding: {
    sm: "pad", // 16px - uses semantic token
    md: "pad-lg", // 24px - uses semantic token
    lg: "pad-xl", // 32px - uses semantic token
  },
} as const;

/**
 * Section/Container variants
 */
export const section = {
  container: "mx-auto px-4",

  width: {
    sm: "max-w-3xl",
    md: "max-w-5xl",
    lg: "max-w-7xl",
    xl: "max-w-8xl",
    full: "max-w-full",
  },

  spacing: {
    tight: "space-y-2",
    default: "space-y-4",
    comfortable: "space-y-6",
    spacious: "space-y-8",
  },
} as const;

/**
 * Status indicator variants - for connection status, health, etc.
 */
export const status = {
  dot: "inline-block w-2 h-2 rounded-full",

  color: {
    success: "bg-status-success",
    warning: "bg-status-warning",
    error: "bg-status-error",
    info: "bg-status-info",
    inactive: "bg-surface-border",
  },

  withLabel: "inline-flex items-center gap-2",
} as const;

/**
 * Severity colors - for CVE/vulnerability ratings (industry standard)
 * Critical = Red, High = Orange, Medium = Yellow, Low = Green
 */
export const severity = {
  critical: {
    bg: "bg-status-error/15",
    text: "text-status-error",
    border: "border-status-error/30",
    dot: "bg-status-error",
  },
  high: {
    bg: "bg-orange-500/15 dark:bg-orange-400/15",
    text: "text-orange-600 dark:text-orange-400",
    border: "border-orange-500/30 dark:border-orange-400/30",
    dot: "bg-orange-500 dark:bg-orange-400",
  },
  medium: {
    bg: "bg-status-warning/15",
    text: "text-status-warning",
    border: "border-status-warning/30",
    dot: "bg-status-warning",
  },
  low: {
    bg: "bg-status-success/15",
    text: "text-status-success",
    border: "border-status-success/30",
    dot: "bg-status-success",
  },
  info: {
    bg: "bg-status-info/15",
    text: "text-status-info",
    border: "border-status-info/30",
    dot: "bg-status-info",
  },
} as const;

/**
 * Timing/phase colors - for HTTP timing bars, performance metrics
 * Following industry conventions for network timing visualization
 */
export const timing = {
  dns: {
    bg: "bg-blue-500 dark:bg-blue-400",
    text: "text-blue-600 dark:text-blue-400",
  },
  tcp: {
    bg: "bg-cyan-500 dark:bg-cyan-400",
    text: "text-cyan-600 dark:text-cyan-400",
  },
  tls: {
    bg: "bg-purple-500 dark:bg-purple-400",
    text: "text-purple-600 dark:text-purple-400",
  },
  wait: {
    bg: "bg-amber-500 dark:bg-amber-400",
    text: "text-amber-600 dark:text-amber-400",
  },
  download: {
    bg: "bg-green-500 dark:bg-green-400",
    text: "text-green-600 dark:text-green-400",
  },
} as const;

/**
 * Category colors - for device types, network segments
 */
export const category = {
  router: "text-blue-500 dark:text-blue-400",
  server: "text-purple-500 dark:text-purple-400",
  workstation: "text-green-500 dark:text-green-400",
  printer: "text-orange-500 dark:text-orange-400",
  mobile: "text-cyan-500 dark:text-cyan-400",
  network: "text-teal-500 dark:text-teal-400",
  unknown: "text-text-muted",
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
    icon: "text-module-roots", // Uses CSS variable
    badge: "bg-module-roots/20 text-module-roots",
    border: "border-module-roots/30",
  },
  // Canopy - WiFi Planning, Surveys, Coverage
  canopy: {
    icon: "text-module-canopy", // Matches brand primary
    badge: "bg-module-canopy/20 text-module-canopy",
    border: "border-module-canopy/30",
  },
  // Shell - Security Posture, Hardening
  shell: {
    icon: "text-module-shell",
    badge: "bg-module-shell/20 text-module-shell",
    border: "border-module-shell/30",
  },
  // Sap - Live Telemetry, Monitoring, Data Flow
  sap: {
    icon: "text-module-sap",
    badge: "bg-module-sap/20 text-module-sap",
    border: "border-module-sap/30",
  },
  // Harvest - Reports, Compliance, Exports
  harvest: {
    icon: "text-module-harvest", // Matches brand gold
    badge: "bg-module-harvest/20 text-module-harvest",
    border: "border-module-harvest/30",
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
    text: "text-brand-gold",
    bg: "bg-brand-gold",
    badge: "bg-brand-gold/20 text-brand-gold",
    border: "border-brand-gold/30",
  },
} as const;

/**
 * Gauge colors - for speed gauges, progress indicators
 * Returns CSS variable-compatible color based on percentage
 */
export const gauge = {
  getColor: (percentage: number): string => {
    if (percentage < 25) return "var(--color-status-error)";
    if (percentage < 50) return "var(--color-status-warning)";
    if (percentage < 75) return "var(--gauge-amber, #eab308)";
    return "var(--color-status-success)";
  },
  // Tailwind class equivalents for non-SVG usage
  class: {
    critical: "text-status-error",
    warning: "text-status-warning",
    caution: "text-amber-500 dark:text-amber-400",
    good: "text-status-success",
  },
} as const;

// ============================================================================
// ICON SIZING
// ============================================================================
/**
 * Icon sizes - standardized icon dimensions
 * Use these instead of arbitrary w-4 h-4, w-5 h-5, etc.
 */
export const icon = {
  // Size tokens (width and height combined)
  size: {
    xs: "w-3 h-3", // 12px - inline with small text
    sm: "w-4 h-4", // 16px - most common, inline with body text
    md: "w-5 h-5", // 20px - buttons, list items
    lg: "w-6 h-6", // 24px - card headers, prominent icons
    xl: "w-8 h-8", // 32px - empty states, feature icons
    "2xl": "w-12 h-12", // 48px - hero sections, large features
  },

  // Common icon + text patterns
  inline: "inline-flex items-center gap-1.5", // Icon with inline text
  button: "inline-flex items-center gap-2", // Icon in button
  leading: "flex items-center gap-2", // Icon leading text
} as const;

// ============================================================================
// BORDER & RADIUS
// ============================================================================
/**
 * Border radius tokens - consistent rounding across components
 */
export const radius = {
  none: "rounded-none",
  sm: "rounded-sm", // 2px - subtle rounding
  default: "rounded", // 4px - inputs, small elements
  md: "rounded-md", // 6px - buttons
  lg: "rounded-lg", // 8px - cards, panels
  xl: "rounded-xl", // 12px - modals, large containers
  full: "rounded-full", // Pills, badges, avatars
} as const;

/**
 * Border tokens - consistent border styling
 */
export const border = {
  // Border widths
  width: {
    none: "border-0",
    default: "border", // 1px
    thick: "border-2", // 2px
  },

  // Border colors (using design tokens)
  color: {
    default: "border-surface-border",
    focus: "border-brand-primary",
    error: "border-status-error",
    success: "border-status-success",
    warning: "border-status-warning",
  },

  // Common combinations
  card: "border border-surface-border",
  input: "border border-surface-border focus:border-brand-primary",
  divider: "border-t border-surface-border",
} as const;

// ============================================================================
// LAYOUT PATTERNS
// ============================================================================
/**
 * Layout tokens - common flex/grid patterns
 * Use these instead of repeating flex items-center gap-2 everywhere
 */
export const layout = {
  // Flex utilities
  flex: {
    center: "flex items-center justify-center",
    between: "flex items-center justify-between",
    start: "flex items-center justify-start",
    end: "flex items-center justify-end",
    col: "flex flex-col",
    colCenter: "flex flex-col items-center justify-center",
    wrap: "flex flex-wrap",
  },

  // Grid layouts
  grid: {
    // Responsive card grids - cards stretch to fill available space
    cards: "grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6",
    cardsWide: "grid grid-cols-1 lg:grid-cols-3 gap-6",
    // Form layouts
    form2col: "grid grid-cols-2 gap-2",
    form4col: "grid grid-cols-2 md:grid-cols-4 gap-4",
    // Settings/data layouts
    data2col: "grid grid-cols-2 gap-x-4 gap-y-2",
    data3col: "grid grid-cols-3 gap-2",
  },

  // Inline patterns (horizontal lists, tags, buttons)
  inline: {
    tight: "flex items-center gap-1",
    default: "flex items-center gap-2",
    comfortable: "flex items-center gap-3",
    spacious: "flex items-center gap-4",
    wrap: "flex flex-wrap items-center gap-2",
  },

  // Stack patterns (vertical lists)
  stack: {
    tight: "flex flex-col gap-1",
    default: "flex flex-col gap-2",
    comfortable: "flex flex-col gap-3",
    spacious: "flex flex-col gap-4",
  },
} as const;

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Combine class names with Tailwind class conflict resolution.
 * Uses tailwind-merge to properly handle conflicting Tailwind classes
 * (e.g., z-50 vs z-20, p-4 vs p-2 will resolve to the last value).
 */
export function cn(...classes: (string | boolean | undefined | null)[]): string {
  return twMerge(classes.filter(Boolean).join(" "));
}

// Type-safe Maps for dynamic lookups
const buttonVariantMap = new Map<keyof typeof button.variant, string>(
  Object.entries(button.variant) as [keyof typeof button.variant, string][],
);
const buttonSizeMap = new Map<keyof typeof button.size, string>(
  Object.entries(button.size) as [keyof typeof button.size, string][],
);

/**
 * Build a button class string
 */
export function buttonClass(
  variant: keyof typeof button.variant = "primary",
  size: keyof typeof button.size = "md",
  className?: string,
): string {
  return cn(button.base, buttonVariantMap.get(variant), buttonSizeMap.get(size), className);
}

// Type-safe Maps for input lookups
const inputStateMap = new Map<keyof typeof input.state, string>(
  Object.entries(input.state) as [keyof typeof input.state, string][],
);
const inputSizeMap = new Map<keyof typeof input.size, string>(
  Object.entries(input.size) as [keyof typeof input.size, string][],
);

/**
 * Build an input class string
 */
export function inputClass(
  state: keyof typeof input.state = "default",
  size: keyof typeof input.size = "md",
  className?: string,
): string {
  return cn(input.base, inputStateMap.get(state), inputSizeMap.get(size), className);
}

// Type-safe Maps for card lookups
const cardVariantMap = new Map<keyof typeof card.variant, string>(
  Object.entries(card.variant) as [keyof typeof card.variant, string][],
);
const cardPaddingMap = new Map<keyof typeof card.padding, string>(
  Object.entries(card.padding) as [keyof typeof card.padding, string][],
);

/**
 * Build a card class string
 */
export function cardClass(
  variant: keyof typeof card.variant = "default",
  padding: keyof typeof card.padding = "md",
  className?: string,
): string {
  return cn(card.base, cardVariantMap.get(variant), cardPaddingMap.get(padding), className);
}

// Type-safe Map for badge lookups
const badgeVariantMap = new Map<keyof typeof badge.variant, string>(
  Object.entries(badge.variant) as [keyof typeof badge.variant, string][],
);

/**
 * Build a badge class string
 */
export function badgeClass(
  variant: keyof typeof badge.variant = "default",
  className?: string,
): string {
  return cn(badge.base, badgeVariantMap.get(variant), className);
}

// Type-safe Maps for modal lookups
const modalSizeMap = new Map<keyof typeof modal.size, string>(
  Object.entries(modal.size) as [keyof typeof modal.size, string][],
);
const modalPaddingMap = new Map<keyof typeof modal.padding, string>(
  Object.entries(modal.padding) as [keyof typeof modal.padding, string][],
);

/**
 * Build a modal class string
 */
export function modalClass(
  size: keyof typeof modal.size = "md",
  padding: keyof typeof modal.padding = "md",
  className?: string,
): string {
  return cn(modal.content, modalSizeMap.get(size), modalPaddingMap.get(padding), className);
}
