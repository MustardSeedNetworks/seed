# LuminetIQ Design System & Theming Guide

This document describes the design system and theming architecture used in the LuminetIQ frontend.

## Architecture Overview

The theming system consists of three layers:

1. **CSS Variables** (`src/index.css`) - Core color tokens for light/dark modes
2. **Design System** (`src/styles/theme.ts`) - TypeScript tokens and utility functions
3. **Tailwind Classes** - CSS-first configuration using `@theme` directive

## Color Tokens

### Brand Colors

| Token                   | Light     | Dark      | Usage                  |
| ----------------------- | --------- | --------- | ---------------------- |
| `--color-brand-primary` | `#1d4ed8` | `#93c5fd` | Primary actions, links |
| `--color-brand-accent`  | `#2563eb` | `#60a5fa` | Hover states           |

### Surface Colors

| Token                    | Light     | Dark      | Usage             |
| ------------------------ | --------- | --------- | ----------------- |
| `--color-surface-base`   | `#e5edf5` | `#0c1626` | Page background   |
| `--color-surface-raised` | `#f9fbfd` | `#131f32` | Cards, modals     |
| `--color-surface-border` | `#8fa3ba` | `#2a3a52` | Borders           |
| `--color-surface-hover`  | `#d4deea` | `#1c2a40` | Hover backgrounds |

### Text Colors

| Token                    | Light     | Dark      | Usage                       |
| ------------------------ | --------- | --------- | --------------------------- |
| `--color-text-primary`   | `#0b1220` | `#f8fbff` | Main text                   |
| `--color-text-secondary` | `#1b2737` | `#dbe5f3` | Secondary text              |
| `--color-text-muted`     | `#334155` | `#b8c5d9` | Subtle text                 |
| `--color-text-accent`    | `#1d4ed8` | `#93c5fd` | Links, highlights           |
| `--color-text-inverse`   | `#f8fafc` | `#0f172a` | Text on colored backgrounds |

### Status Colors (Industry Standard)

| Token                    | Light     | Dark      | Usage          |
| ------------------------ | --------- | --------- | -------------- |
| `--color-status-success` | `#047857` | `#86efac` | Success states |
| `--color-status-warning` | `#92400e` | `#fcd34d` | Warning states |
| `--color-status-error`   | `#b91c1c` | `#fca5a5` | Error states   |
| `--color-status-info`    | `#1d4ed8` | `#93c5fd` | Informational  |

## Typography Scale

LuminetIQ uses a semantic typography system with responsive sizing. Use these CSS utility classes instead of raw Tailwind size classes.

### Heading Classes

| Class            | Size (mobile/desktop) | Weight   | Usage                                      |
| ---------------- | --------------------- | -------- | ------------------------------------------ |
| `.heading-1`     | 24px / 30px           | Bold     | Page titles (login, major pages)           |
| `.heading-2`     | 20px / 24px           | Semibold | Section titles, modal headers              |
| `.heading-3`     | 18px / 20px           | Semibold | Card titles, subsection headers            |
| `.heading-4`     | 16px / 18px           | Medium   | Form sections, minor headings              |
| `.section-title` | 12px (uppercase)      | Medium   | Category labels (Connectivity, Network...) |

### Body Text Classes

| Class         | Size | Color     | Usage                             |
| ------------- | ---- | --------- | --------------------------------- |
| `.body-large` | 18px | Primary   | Emphasized paragraphs             |
| `.body`       | 16px | Primary   | Default paragraph text            |
| `.body-small` | 14px | Secondary | Secondary content, descriptions   |
| `.caption`    | 12px | Muted     | Metadata, timestamps, badges      |
| `.label`      | 14px | Primary   | Form field labels (medium weight) |
| `.code`       | 14px | Primary   | Inline code with background       |

### Usage Examples

```tsx
// Headings - use semantic HTML with utility classes
<h1 className="heading-1">Welcome to LuminetIQ</h1>
<h2 className="heading-2">Network Overview</h2>
<h2 className="section-title">Connectivity</h2>  // Category label
<h3 className="heading-3">DNS Status</h3>         // Card title

// Body text
<p className="body">Regular paragraph content.</p>
<p className="body-small">Secondary explanation text.</p>
<span className="caption">Last updated: 5 min ago</span>

// Form labels
<label className="label">Server Address</label>

// Inline code
<code className="code">192.168.1.1</code>
```

### TypeScript Imports

```tsx
import { typography } from '../styles/theme';

// Access class names programmatically
<h1 className={typography.heading.h1}>Title</h1>
<p className={typography.body.default}>Text</p>
<span className={typography.body.caption}>Metadata</span>
```

## Using Tailwind Classes

### Text Colors

```tsx
// Good - uses design tokens
<span className="text-text-primary">Main text</span>
<span className="text-text-secondary">Secondary</span>
<span className="text-text-muted">Muted text</span>
<span className="text-text-accent">Link text</span>
<span className="text-text-inverse">On colored bg</span>

// Bad - hardcoded colors
<span className="text-white">Don't do this</span>
<span className="text-gray-500">Or this</span>
```

### Background Colors

```tsx
// Good
<div className="bg-surface-base">Page</div>
<div className="bg-surface-raised">Card</div>
<div className="bg-surface-hover">Hover state</div>

// Bad
<div className="bg-white">Don't do this</div>
<div className="bg-gray-100">Or this</div>
```

### Status Colors

```tsx
// Good - uses design tokens
<span className="text-status-success">Success</span>
<span className="text-status-warning">Warning</span>
<span className="text-status-error">Error</span>
<span className="text-status-info">Info</span>

// Bad
<span className="text-green-500">Don't do this</span>
<span className="text-red-500">Or this</span>
```

## Design System Utilities

Import utilities from `src/styles/theme.ts`:

```tsx
import { cn, buttonClass, cardClass, badgeClass, severity, timing, category } from '../styles/theme';

// Button classes
<button className={buttonClass('primary', 'md')}>Primary</button>
<button className={buttonClass('secondary', 'sm')}>Secondary</button>
<button className={buttonClass('danger', 'lg')}>Danger</button>

// Card classes
<div className={cardClass('default', 'md')}>Card content</div>

// Badge classes
<span className={badgeClass('success')}>Success</span>
<span className={badgeClass('error')}>Error</span>
```

## Spacing & Layout

### Spacing Scale

Use these consistent spacing values (based on 4px grid):

| Token         | Value | Pixels | Usage                         |
| ------------- | ----- | ------ | ----------------------------- |
| `tight`       | 0.5   | 2px    | Inline elements, tight gaps   |
| `compact`     | 2     | 8px    | Compact layouts, small gaps   |
| `default`     | 3     | 12px   | Default spacing, card padding |
| `comfortable` | 4     | 16px   | Comfortable spacing           |
| `spacious`    | 6     | 24px   | Section separation            |
| `section`     | 8     | 32px   | Major section gaps            |
| `major`       | 12    | 48px   | Page-level separation         |

### Common Patterns

```tsx
// Card padding (responsive)
<div className="p-3 sm:p-4">

// Container padding (responsive)
<div className="px-4 sm:px-6 lg:px-8">

// Section gaps
<div className="space-y-6">  // Between sections
<div className="space-y-3">  // Within sections
<div className="gap-4">      // Grid/flex gaps
```

### Layout Utilities

```tsx
// CSS utilities defined in index.css
<div className="content-max">        // max-w-7xl mx-auto px-4
<div className="content-grid">       // Responsive 1-2-3 column grid
<div className="flex-center">        // flex items-center justify-center
<div className="flex-between">       // flex items-center justify-between
```

### Dashboard Grid

The main dashboard uses a responsive grid:

```tsx
// 1 column on mobile, 2 on tablet, 3 on desktop, 4 on large screens
<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
```

## Severity Colors (CVE/Vulnerability)

For vulnerability severity, use the `severity` object from the theme:

```tsx
import { severity } from '../styles/theme';

// Critical - Red
<span className={cn(severity.critical.bg, severity.critical.text, severity.critical.border)}>
  CRITICAL
</span>

// High - Orange
<span className={cn(severity.high.bg, severity.high.text, severity.high.border)}>
  HIGH
</span>

// Medium - Amber/Yellow
<span className={cn(severity.medium.bg, severity.medium.text, severity.medium.border)}>
  MEDIUM
</span>

// Low - Green
<span className={cn(severity.low.bg, severity.low.text, severity.low.border)}>
  LOW
</span>
```

## Timing/Phase Colors (Network)

For HTTP timing bars and performance metrics:

```tsx
import { timing } from '../styles/theme';

// DNS lookup - Blue
<div className={timing.dns.bg}>DNS</div>

// TCP connection - Cyan
<div className={timing.tcp.bg}>TCP</div>

// TLS handshake - Purple
<div className={timing.tls.bg}>TLS</div>

// Wait/TTFB - Amber
<div className={timing.wait.bg}>Wait</div>

// Download - Green
<div className={timing.download.bg}>Download</div>
```

## Device Category Colors

For device type icons in network discovery:

```tsx
import { category } from '../styles/theme';

<RouterIcon className={category.router} />      // Blue
<ServerIcon className={category.server} />      // Purple
<MonitorIcon className={category.workstation} /> // Green
<PrinterIcon className={category.printer} />    // Orange
<PhoneIcon className={category.mobile} />       // Cyan
<WifiIcon className={category.network} />       // Teal
```

## SVG and Canvas Colors

For SVG elements that need dynamic colors, use CSS variables:

```tsx
import { gauge } from "../styles/theme";

// Get color based on percentage
const color = gauge.getColor(percentage);
// Returns: "var(--color-status-error)" for <25%
//          "var(--color-status-warning)" for 25-50%
//          "var(--gauge-amber, #eab308)" for 50-75%
//          "var(--color-status-success)" for >75%

<svg>
  <circle fill={color} />
</svg>;
```

## Dark Mode Implementation

Dark mode is implemented via the `useTheme` hook:

```tsx
import { useTheme } from "../hooks/useTheme";

function Component() {
  const { theme, setTheme, actualTheme } = useTheme();

  // theme: 'light' | 'dark' | 'system'
  // actualTheme: 'light' | 'dark' (resolved value)

  return (
    <button onClick={() => setTheme(theme === "dark" ? "light" : "dark")}>
      Toggle Theme
    </button>
  );
}
```

The `.dark` class is applied to the `<html>` element, and all CSS variables automatically switch.

## Color Accessibility

All color combinations meet WCAG AA contrast requirements:

- Light mode text: 4.5:1 minimum contrast ratio
- Dark mode text: 4.5:1 minimum contrast ratio
- Status colors: Adjusted for readability in both modes

## Adding New Colors

1. Add CSS variable to `src/index.css` in both `:root` and `.dark`
2. Add Tailwind utility mapping in `@theme` block if needed
3. Add TypeScript constant in `src/styles/theme.ts`
4. Document usage in this file

## Anti-Patterns to Avoid

### Don't use hardcoded colors

```tsx
// Bad
<span className="text-white">Text</span>
<span className="text-gray-500">Text</span>
<div style={{ color: '#ffffff' }}>Text</div>

// Good
<span className="text-text-inverse">Text</span>
<span className="text-text-muted">Text</span>
```

### Don't use arbitrary Tailwind colors without dark mode variants

```tsx
// Bad - breaks in dark mode
<span className="text-blue-400">Text</span>

// Good - dark mode aware
<span className="text-blue-600 dark:text-blue-400">Text</span>

// Better - use status token if semantic
<span className="text-status-info">Text</span>
```

### Don't hardcode hex in JavaScript for SVG/Canvas

```tsx
// Bad
const color = "#ef4444";

// Good
const color = gauge.getColor(percentage);
// or
const color = "var(--color-status-error)";
```

## Exceptions: Opacity Variants

These patterns are **allowed** because they work consistently in both light/dark modes:

### Modal/Dialog Overlays

```tsx
// Allowed - semi-transparent black for dimming background
<div className="bg-black/50">Modal overlay</div>;

// Better - use design system
import { modal } from "../styles/theme";
<div className={modal.overlay}>Modal overlay</div>;
```

### Hover Effects on Colored Backgrounds

```tsx
// Allowed - subtle brightening on colored buttons/toasts
<button className="bg-status-success hover:bg-white/20">...</button>
```

These work because:

- `bg-black/50` always darkens regardless of theme
- `bg-white/20` always lightens regardless of theme
- They don't establish foreground/background contrast that would break in dark mode
