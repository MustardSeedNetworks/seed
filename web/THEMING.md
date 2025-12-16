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

LuminetIQ uses a semantic typography system with responsive sizing. Use these CSS utility classes
instead of raw Tailwind size classes.

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
import { cn, buttonClass, cardClass, badgeClass, inputClass, button, input } from '../styles/theme';

// Button classes - use buttonClass() helper
<button className={buttonClass('primary', 'md')}>Primary Button</button>
<button className={buttonClass('secondary', 'sm')}>Secondary</button>
<button className={buttonClass('danger', 'lg')}>Danger</button>
<button className={buttonClass('ghost', 'xs')}>Tiny Action</button>

// Or use button tokens directly
<button className={cn(button.base, button.variant.primary, button.size.md)}>
  Primary
</button>

// Input classes - use inputClass() helper
<input className={inputClass('default', 'md')} />  // Most common
<input className={inputClass('error', 'sm')} />    // Compact with error state

// Card classes
<div className={cardClass('default', 'md')}>Card content</div>

// Badge classes
<span className={badgeClass('success')}>Success</span>
<span className={badgeClass('error')}>Error</span>
```

### Button & Input Sizes

Use consistent sizes for buttons and inputs:

| Size | Button Padding | Input Padding | Usage                            |
| ---- | -------------- | ------------- | -------------------------------- |
| `xs` | `px-2 py-1`    | -             | Tiny buttons, icon actions       |
| `sm` | `px-3 py-1.5`  | `px-2 py-1.5` | Compact forms, secondary actions |
| `md` | `px-4 py-2`    | `px-2.5 py-2` | Default size (most common)       |
| `lg` | `px-6 py-3`    | `px-3 py-2.5` | Large CTAs, prominent inputs     |

**Important**: Always use `buttonClass()` or `inputClass()` helpers instead of raw Tailwind padding
classes to ensure consistency.

## Spacing & Layout

### Spacing Scale

Use these consistent spacing values (based on 4px grid). **Prefer semantic CSS utility classes**
over raw Tailwind values:

| Raw Value | Pixels | Semantic Class   | Usage                       |
| --------- | ------ | ---------------- | --------------------------- |
| `1`       | 4px    | `stack-xs`       | Tight inline elements       |
| `2`       | 8px    | `stack-sm`       | Compact layouts, small gaps |
| `3`       | 12px   | `stack`          | Default vertical spacing    |
| `4`       | 16px   | `stack-lg`       | Comfortable spacing         |
| `6`       | 24px   | `stack-xl`       | Major section separation    |
| `8`       | 32px   | `section-gap-lg` | Page-level separation       |

### Semantic Spacing Classes

```tsx
// Vertical stacking (use instead of space-y-*)
<div className="stack-sm">      // 8px vertical gap
<div className="stack">         // 12px vertical gap (default)
<div className="stack-lg">      // 16px vertical gap
<div className="stack-xl">      // 24px vertical gap

// Section separators
<div className="section-gap">   // 24px between major sections
<div className="section-gap-lg"> // 32px for page-level sections

// Flex/grid gaps (use instead of gap-*)
<div className="gap-tight">      // 4px
<div className="gap-compact">    // 8px
<div className="gap-default">    // 12px
<div className="gap-comfortable"> // 16px
<div className="gap-spacious">   // 24px

// Container padding (use instead of p-*)
<div className="pad-sm">         // 12px padding
<div className="pad">            // 16px padding (default)
<div className="pad-lg">         // 24px padding

// Inline gaps (for buttons, badges, icon groups)
<div className="inline-gap-xs">  // 4px
<div className="inline-gap-sm">  // 6px
<div className="inline-gap">     // 8px (default)
<div className="inline-gap-lg">  // 12px
```

### Spacing TypeScript Imports

```tsx
import { spacing } from '../styles/theme';

// Access class names programmatically
<div className={spacing.stack.default}>...</div>
<div className={spacing.gap.comfortable}>...</div>
<div className={spacing.pad.lg}>...</div>
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
<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-comfortable">
```

### Sizing Tokens

For consistent heights and widths, use the sizing tokens instead of arbitrary values:

```tsx
import { sizing } from '../styles/theme';

// Modal/drawer heights
<div className={sizing.height.modal}>   // max-h-[85vh]
<div className={sizing.height.panel}>   // max-h-[70vh]

// Drawer/panel widths
<div className={sizing.width.drawer}>     // w-80 (320px)
<div className={sizing.width.drawerWide}> // w-96 (384px)
<div className={sizing.width.dropdown}>   // w-64 (256px)
```

### Anti-Patterns for Spacing

```tsx
// Bad - arbitrary pixel values
<div className="h-125">       // Use sizing tokens or vh units
<div className="w-md">        // Use sizing.width.* for consistency
<div className="mb-4 mt-2">   // Inconsistent margins

// Good - use semantic classes
<div className="max-h-modal">  // Defined in design system (85vh)
<div className="max-h-[70vh]"> // Viewport-relative for panels
<div className="w-96">         // Standard drawer width (384px)
<div className="stack-lg">     // Consistent 16px spacing
```

## Alert/Banner Components

For inline error/warning/info messages, use the `alert` tokens:

```tsx
import { alert, cn } from '../styles/theme';

// Error alert
<div className={cn(alert.base, alert.variant.error)}>
  Something went wrong!
</div>

// Warning alert
<div className={cn(alert.base, alert.variant.warning)}>
  Please review before continuing.
</div>

// Success alert
<div className={cn(alert.base, alert.variant.success)}>
  Operation completed successfully!
</div>

// Info alert
<div className={cn(alert.base, alert.variant.info)}>
  Here's some helpful information.
</div>
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
    <button onClick={() => setTheme(theme === "dark" ? "light" : "dark")}>Toggle Theme</button>
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

### Canvas API

The HTML Canvas API (`<canvas>`) cannot use CSS variables directly. Colors must be hardcoded:

```tsx
// Canvas API limitation - must use direct color values
ctx.fillStyle = "rgba(37, 99, 235, 0.8)"; // brand-primary blue
ctx.strokeStyle = "#ffffff"; // white border

// For SVG, CSS variables DO work:
<svg>
  <circle fill="var(--color-brand-primary)" />
</svg>;
```

When using Canvas, document the color mapping in comments (e.g., `// brand-primary (#2563eb)`).

### Discovery Method Colors

Network discovery methods use distinct colors for visual identification. These use Tailwind colors
with dark: variants:

```tsx
// Allowed - semantic colored badges with dark mode support
const methodColors = {
  arp: "bg-blue-500/20 text-blue-600 dark:text-blue-400",
  lldp: "bg-green-500/20 text-green-600 dark:text-green-400",
  cdp: "bg-orange-500/20 text-orange-600 dark:text-orange-400",
};
```

These are intentionally colored to help users quickly distinguish protocol types.

## Icon Sizing

Use consistent icon sizes across the application. Import from the design system:

```tsx
import { icon } from '../styles/theme';

// Standard sizes
<Settings className={icon.size.xs} />   // 12px - with caption text
<Settings className={icon.size.sm} />   // 16px - most common (body text)
<Settings className={icon.size.md} />   // 20px - buttons, list items
<Settings className={icon.size.lg} />   // 24px - card headers
<Settings className={icon.size.xl} />   // 32px - empty states
<Settings className={icon.size["2xl"]} /> // 48px - hero sections

// Icon with text patterns
<div className={icon.inline}>           // Icon + inline text
  <Info className={icon.size.sm} /> Info
</div>

<div className={icon.leading}>          // Icon leading text block
  <AlertCircle className={icon.size.md} />
  <span>Warning message</span>
</div>
```

### Icon Size Reference

| Token           | Size | Tailwind    | Usage                         |
| --------------- | ---- | ----------- | ----------------------------- |
| `icon.size.xs`  | 12px | `w-3 h-3`   | Caption text, tiny indicators |
| `icon.size.sm`  | 16px | `w-4 h-4`   | Body text, most common        |
| `icon.size.md`  | 20px | `w-5 h-5`   | Buttons, list item icons      |
| `icon.size.lg`  | 24px | `w-6 h-6`   | Card headers, prominent icons |
| `icon.size.xl`  | 32px | `w-8 h-8`   | Empty states, feature icons   |
| `icon.size.2xl` | 48px | `w-12 h-12` | Hero sections, large features |

**Important**: Always use `icon.size.*` instead of raw `w-4 h-4`, `w-5 h-5`, etc.

## Border & Radius

### Border Radius

Use consistent border radius tokens:

```tsx
import { radius } from '../styles/theme';

<div className={radius.sm}>      // 2px - subtle rounding
<div className={radius.default}> // 4px - inputs, small elements
<div className={radius.md}>      // 6px - buttons
<div className={radius.lg}>      // 8px - cards, panels
<div className={radius.xl}>      // 12px - modals
<div className={radius.full}>    // Pills, badges, avatars
```

### Border Tokens

```tsx
import { border } from '../styles/theme';

// Border widths
<div className={border.width.default}> // 1px border
<div className={border.width.thick}>   // 2px border

// Border colors (use with border width)
<div className={cn(border.width.default, border.color.default)}> // Standard border
<div className={cn(border.width.default, border.color.error)}>   // Error state

// Common combinations
<div className={border.card}>     // border border-surface-border
<div className={border.divider}>  // border-t border-surface-border
```

### Border Reference

| Token            | Tailwind Classes                 | Usage            |
| ---------------- | -------------------------------- | ---------------- |
| `radius.sm`      | `rounded-sm`                     | Subtle rounding  |
| `radius.default` | `rounded`                        | Inputs, badges   |
| `radius.md`      | `rounded-md`                     | Buttons          |
| `radius.lg`      | `rounded-lg`                     | Cards, panels    |
| `radius.xl`      | `rounded-xl`                     | Modals, drawers  |
| `radius.full`    | `rounded-full`                   | Pills, avatars   |
| `border.card`    | `border border-surface-border`   | Card borders     |
| `border.divider` | `border-t border-surface-border` | Section dividers |

## Layout Patterns

Use semantic layout tokens instead of repeating flex/grid patterns:

```tsx
import { layout } from '../styles/theme';

// Flex utilities
<div className={layout.flex.center}>   // flex items-center justify-center
<div className={layout.flex.between}>  // flex items-center justify-between
<div className={layout.flex.col}>      // flex flex-col
<div className={layout.flex.colCenter}> // flex flex-col items-center justify-center

// Grid layouts
<div className={layout.grid.cards}>     // Responsive 1-2-3-4 col grid
<div className={layout.grid.cardsWide}> // 1-3 col grid
<div className={layout.grid.form2col}>  // 2 column form grid
<div className={layout.grid.form4col}>  // 2-4 column responsive form grid
<div className={layout.grid.data2col}>  // Data display (labels + values)
<div className={layout.grid.data3col}>  // 3 column data grid

// Inline patterns (horizontal items)
<div className={layout.inline.tight}>      // gap-1
<div className={layout.inline.default}>    // gap-2
<div className={layout.inline.comfortable}> // gap-3
<div className={layout.inline.spacious}>   // gap-4
<div className={layout.inline.wrap}>       // flex-wrap gap-2

// Stack patterns (vertical items)
<div className={layout.stack.tight}>      // flex-col gap-1
<div className={layout.stack.default}>    // flex-col gap-2
<div className={layout.stack.comfortable}> // flex-col gap-3
<div className={layout.stack.spacious}>   // flex-col gap-4
```

### Layout Reference

| Token                   | Tailwind Classes                                                      | Usage                 |
| ----------------------- | --------------------------------------------------------------------- | --------------------- |
| `layout.flex.center`    | `flex items-center justify-center`                                    | Centering content     |
| `layout.flex.between`   | `flex items-center justify-between`                                   | Header with actions   |
| `layout.grid.cards`     | `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4` | Dashboard cards       |
| `layout.grid.form2col`  | `grid grid-cols-2 gap-2`                                              | Threshold inputs      |
| `layout.inline.default` | `flex items-center gap-2`                                             | Button groups, badges |
| `layout.stack.default`  | `flex flex-col gap-2`                                                 | Form fields           |

## Complete Token Quick Reference

### When to Use What

| Pattern          | Instead of                   | Use Token                     |
| ---------------- | ---------------------------- | ----------------------------- |
| Text color       | `text-white`, `text-gray-*`  | `text-text-*`                 |
| Background       | `bg-white`, `bg-gray-*`      | `bg-surface-*`                |
| Status colors    | `text-red-*`, `text-green-*` | `text-status-*`               |
| Icon dimensions  | `w-4 h-4`, `w-5 h-5`         | `icon.size.*`                 |
| Border radius    | `rounded-lg`, `rounded-md`   | `radius.*`                    |
| Vertical spacing | `space-y-2`, `space-y-4`     | `stack-*` or `layout.stack.*` |
| Horizontal gaps  | `gap-2`, `gap-3`             | `layout.inline.*`             |
| Responsive grid  | `grid grid-cols-1 sm:...`    | `layout.grid.*`               |
| Buttons          | Raw `px-*` `py-*` classes    | `buttonClass()` or `button.*` |
| Inputs           | Raw `px-*` `py-*` classes    | `inputClass()` or `input.*`   |
| Card wrapper     | `rounded-lg border bg-*`     | `cardClass()` or `card.*`     |
| Modal overlay    | `fixed inset-0 bg-black/50`  | `modal.overlay`               |

### ESLint Enforcement

The following patterns are flagged by ESLint:

- `text-white` → Use `text-text-inverse`
- `text-black` → Use `text-text-primary`
- `bg-white` (not opacity) → Use `bg-surface-raised`
- `bg-black` (not opacity) → Use design tokens
- `text-gray-*` → Use `text-text-*` tokens
- `bg-gray-*` → Use `bg-surface-*` tokens

### Migration Checklist

When refactoring existing components:

1. ✅ Replace hardcoded colors with design tokens
2. ✅ Replace `w-* h-*` icon sizes with `icon.size.*`
3. ✅ Replace `rounded-*` with `radius.*` tokens
4. ✅ Replace `flex items-center gap-*` with `layout.inline.*`
5. ✅ Replace `flex flex-col gap-*` with `layout.stack.*`
6. ✅ Replace `space-y-*` with `stack-*` or `section-gap-*`
7. ✅ Replace raw `px-* py-*` on buttons/inputs with helper functions
8. ✅ Replace responsive grid patterns with `layout.grid.*`
