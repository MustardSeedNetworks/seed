# The Seed Design System

This design system ensures consistent styling across the application. Instead of scattered utility classes, use the
centralized theme tokens and component utilities.

## Quick Start

````tsx
import { buttonClass, cardClass, cn } from '../styles/theme';

// ❌ Bad - scattered utilities, hard to maintain
<button className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
  Click me
</button>

// ✅ Good - uses design system
<button className={buttonClass('primary', 'md')}>
  Click me
</button>
```python

## Color System

Colors are defined as CSS variables in `index.css` and mapped in `tailwind.config.js`.

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

### Buttons

```tsx
import { buttonClass } from '../styles/theme';

// Primary action button
<button className={buttonClass('primary', 'md')}>
  Save Changes
</button>

// Secondary button
<button className={buttonClass('secondary', 'md')}>
  Cancel
</button>

// Ghost button (no background)
<button className={buttonClass('ghost', 'sm')}>
  View Details
</button>

// Danger button
<button className={buttonClass('danger', 'md')}>
  Delete
</button>

// Custom additions
<button className={buttonClass('primary', 'lg', 'w-full')}>
  Full Width Button
</button>
```python

**Sizes**: `sm` | `md` | `lg`

**Variants**: `primary` | `secondary` | `ghost` | `danger` | `success`

### Inputs

```tsx
import { inputClass } from '../styles/theme';

// Default input
<input className={inputClass('default', 'md')} />

// Error state
<input className={inputClass('error', 'md')} />

// Success state
<input className={inputClass('success', 'md')} />

// Custom additions
<input className={inputClass('default', 'md', 'font-mono')} />
```python

**Sizes**: `sm` | `md` | `lg`

**States**: `default` | `error` | `success`

### Cards

```tsx
import { cardClass } from '../styles/theme';

// Default card
<div className={cardClass('default', 'md')}>
  <h3 className="font-semibold mb-2">Card Title</h3>
  <p>Card content</p>
</div>

// Elevated card (with shadow)
<div className={cardClass('elevated', 'lg')}>
  Content
</div>

// Interactive card (hover effect)
<div className={cardClass('interactive', 'md')}>
  Clickable card
</div>
```python

**Variants**: `default` | `elevated` | `interactive`

**Padding**: `none` | `sm` | `md` | `lg`

### Badges

```tsx
import { badgeClass } from '../styles/theme';

<span className={badgeClass('success')}>Active</span>
<span className={badgeClass('warning')}>Pending</span>
<span className={badgeClass('error')}>Failed</span>
<span className={badgeClass('info')}>Info</span>
<span className={badgeClass('primary')}>New</span>
```python

**Variants**: `default` | `success` | `warning` | `error` | `info` | `primary`

### Modals

```tsx
import { modal, modalClass } from "../styles/theme";

<div className={modal.overlay}>
  <div className={modalClass("md", "md")}>
    <h2 className="text-xl font-semibold mb-4">Modal Title</h2>
    <p className="mb-6">Modal content</p>
    <div className="flex justify-end gap-3">
      <button className={buttonClass("secondary", "md")}>Cancel</button>
      <button className={buttonClass("primary", "md")}>Confirm</button>
    </div>
  </div>
</div>;
```python

**Sizes**: `sm` | `md` | `lg` | `xl` | `full`

**Padding**: `sm` | `md` | `lg`

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
import { buttonClass, cardClass } from '../styles/theme';

<button className={buttonClass('primary', 'md')}>
  Save
</button>

<div className={cardClass('default', 'lg')}>
  <h3 className="text-xl font-semibold mb-2">Title</h3>
</div>
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
