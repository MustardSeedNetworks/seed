# The Seed Design System

This design system ensures consistent styling across the application. Instead of scattered utility classes, use the
centralized theme tokens and component utilities.

> **State management** (React Query vs Zustand vs Context vs local) lives in
> [`../STATE.md`](../STATE.md) — read it before adding a new store or context.

## Token Architecture (read first)

Three tiers, **one** source of truth for values, **one** derivation direction:

```
Primitive   Tailwind's built-in palette (green-500 = #4caf50)   ← never referenced directly in app code
   ↓ alias
Semantic    index.css @theme + :root/.dark                      ← THE source of truth for VALUES
            brand-*, status-*, surface-*, text-*, cat-1..8,
            log-*-bg/-fg, on-brand, on-danger, z-overlay/z-max
   ↓ alias
Component   components/ui/* (<Button>, <Card>, <Input> …)        ← consumes semantic tokens
            + TS class-token objects in styles/theme (status.*, layout.* …)
```

**Two invariants (enforced by lint):**

1. **Values flow one direction** — defined once in `index.css`, everything else
   references them. Never hand-copy a hex sideways into a `.ts`/`.tsx` file.
2. **App code names only semantic / component tokens** — never a primitive
   palette utility (`bg-green-500`) and never a raw hex.

**When you can't use a class** (`<canvas>` drawing, generated HTML/PDF reports),
read the value at runtime from `styles/tokens.ts` (`token('brandPrimary')`),
which reads the CSS variable — so there's still a single source of truth.

**Legitimate exceptions** (allowlisted): `styles/tokens.ts`,
`utils/reportRenderer.ts`, `components/survey/FloorPlanCanvas.tsx`, and named
domain-palette maps (e.g. T568B Ethernet wire colors), which represent physical
reality and are intentionally outside the brand palette.

## Quick Start

````tsx
import { Button } from '../components/ui/Button';

// ❌ Bad - scattered utilities, hard to maintain, raw palette
<button className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
  Click me
</button>

// ✅ Good - the component is the source of truth for button styling
<Button>Click me</Button>
```python

## Color System

Colors are defined as CSS variables in `index.css` (`:root` / `.dark`) and exposed to Tailwind via the `@theme` block in the same file (Tailwind v4 is CSS-first — there is no `tailwind.config.js`).

### Brand Colors

- `bg-brand-primary` - Primary brand color
- `bg-brand-accent` - Accent/hover state
- `text-brand-primary` - Brand text color

### Surface Colors

- `bg-surface-base` - Page background
- `bg-surface-raised` - Card/modal backgrounds
- `bg-surface-hover` - Hover states
- `border-surface-border` - Border color

### Text Colors

- `text-text-primary` - Primary text
- `text-text-secondary` - Secondary text
- `text-text-muted` - Muted/disabled text
- `text-text-accent` - Accent text
- `text-text-inverse` - Light text on dark backgrounds

### Status Colors

- `text-status-success` / `bg-status-success`
- `text-status-warning` / `bg-status-warning`
- `text-status-error` / `bg-status-error`
- `text-status-info` / `bg-status-info`

## Spacing Scale

Use Tailwind's spacing scale (1 unit = 4px):

```tsx
import { spacing } from '../styles/theme';

// Predefined spacing values
spacing.tight      // 0.5 (2px)
spacing.compact    // 2 (8px)
spacing.default    // 3 (12px)
spacing.comfortable // 4 (16px)
spacing.spacious   // 6 (24px)
spacing.section    // 8 (32px)
spacing.major      // 12 (48px)

// Usage
<div className={`mb-${spacing.default} gap-${spacing.comfortable}`}>
```python

### Common Patterns

- **Card padding**: `p-4` or `p-6`
- **Button spacing**: `px-4 py-2`
- **Section spacing**: `space-y-4` or `space-y-6`
- **Grid gaps**: `gap-4` or `gap-6`

## Typography

### Font Sizes

```tsx
import { typography } from '../styles/theme';

<p className={typography.size.base}>     // 16px - body text
<h3 className={typography.size.xl}>      // 20px - card titles
<h2 className={typography.size['2xl']}>  // 24px - section headings
<h1 className={typography.size['3xl']}>  // 30px - page titles
```text

### Font Weights

```tsx
<p className={typography.weight.normal}>    // 400 - body
<p className={typography.weight.medium}>    // 500 - emphasis
<h3 className={typography.weight.semibold}> // 600 - headings
<h1 className={typography.weight.bold}>     // 700 - major headings
```text

### Font Families

```tsx
<p className={typography.family.body}>     // Inter - body text
<h1 className={typography.family.display}> // Inter - headings
<code className={typography.family.mono}>  // JetBrains Mono - code
```python

## Component Variants

> The `<Button>`, `<Card>`, `<Input>`, `<Modal>`, and `<StatusBadge>` components in
> `components/ui/*` are the source of truth for component styling. Prefer them.
> The `button` / `card` / `input` / `badge` / `modal` token objects in
> `styles/theme` remain for ad-hoc composition with `cn()` when a component
> can't apply (e.g. styling a `<form>` like a card).

### Buttons

```tsx
import { Button } from '../components/ui/Button';

<Button>Save Changes</Button>                          // solid / violet (brand)
<Button variant="secondary">Cancel</Button>
<Button variant="ghost" size="sm">View Details</Button>
<Button tone="red">Delete</Button>
<Button className="w-full">Full Width Button</Button>
```python

**variant**: `solid` | `outline` | `ghost` | `secondary` · **tone**: `violet` | `red` | `green` | `blue` | `gray` · **size**: `xs` | `sm` | `md` | `lg`

Ad-hoc (non-`<button>` element): `cn(button.base, button.variant.primary, button.size.md)`.

### Inputs

```tsx
import { Input } from '../components/ui/Input';

<Input label="Email" placeholder="you@example.com" />
<Input label="Email" error="Required" />
<Input label="Host" hint="IP or hostname" rightIcon={<Search />} />
```python

For bespoke inputs (custom label/affordance layout) compose the token object:
`cn(input.base, input.state.default, input.size.md)` — states `default` | `error` | `success`.

### Cards

```tsx
import { Card } from '../components/ui/card';

<Card>Card content</Card>
```python

For a non-`<div>` element that should look like a card (e.g. a `<form>`):
`cn(card.base, card.variant.default, card.padding.lg)` — variants `default` | `elevated` | `interactive`, padding `none` | `sm` | `md` | `lg`.

### Badges

```tsx
import { StatusBadge } from '../components/ui/StatusBadge';

<StatusBadge status="success">Active</StatusBadge>
```python

Or compose the token object: `cn(badge.base, badge.variant.success)` — variants
`default` | `success` | `warning` | `error` | `info` | `primary`.

### Modals

```tsx
import { Modal } from '../components/ui/Modal';

<Modal open={open} onClose={close} title="Modal Title" size="md">
  <p>Modal content</p>
</Modal>
```python

Sizes `sm` | `md` | `lg` | `xl` | `full`. The `modal` token object (overlay /
backdrop / content) remains for fully custom dialogs.

### Status Indicators

```tsx
import { status, cn } from '../styles/theme';

// Status dot
<span className={cn(status.dot, status.color.success)} />

// Status with label
<div className={status.withLabel}>
  <span className={cn(status.dot, status.color.success)} />
  <span>Connected</span>
</div>
```python

**Colors**: `success` | `warning` | `error` | `info` | `inactive`

### Sections/Containers

```tsx
import { section } from "../styles/theme";

// Page container
<div className={cn(section.container, section.width.lg)}>
  <div className={section.spacing.default}>{/* Content with consistent spacing */}</div>
</div>;
```python

**Widths**: `sm` | `md` | `lg` | `xl` | `full`

**Spacing**: `tight` | `default` | `comfortable` | `spacious`

## Utility Function

### `cn()` - Conditional Classes

Safely combine class names, automatically filtering out falsy values:

```tsx
import { cn } from '../styles/theme';

<div className={cn(
  'base-class',
  isActive && 'active-class',
  isDisabled && 'disabled-class',
  customClass
)}>
```text

## Migration Guide

### Before (scattered utilities)

```tsx
<button className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 font-medium">
  Save
</button>

<div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
  <h3 className="text-xl font-semibold mb-2">Title</h3>
</div>
```python

### After (design system)

```tsx
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/card';

<Button>Save</Button>

<Card>
  <h3 className="heading-3 mb-heading">Title</h3>
</Card>
```text

## Benefits

✅ **Consistency**: All components use the same design tokens ✅ **Maintainability**: Change once, update everywhere ✅
**Type Safety**: TypeScript autocomplete for variants ✅ **Accessibility**: Built-in focus states, contrast ratios ✅
**Dark Mode**: Automatic theme switching via CSS variables ✅ **Performance**: No runtime CSS-in-JS overhead

## Best Practices

1. **Always use design system utilities** for new components
2. **Migrate existing components** gradually
3. **Use `cn()` function** for conditional classes
4. **Add custom classes** as the third parameter when needed
5. **Document new patterns** if they're reused 3+ times
6. **Keep colors semantic** - use status colors for meaning, not decoration
````
