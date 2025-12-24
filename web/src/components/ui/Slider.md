# Slider Component

A reusable, accessible range slider component for numeric input with visual feedback and custom formatting.

## Location

`/web/src/components/ui/Slider.tsx`

## Features

- **Visual Feedback**: Filled track shows current value position
- **Custom Formatting**: Support for ms, seconds, minutes, percentages, counts, etc.
- **End Labels**: Optional contextual labels (e.g., "Slower ◄────► Faster")
- **Keyboard Navigation**: Full keyboard support with shortcuts
- **Accessibility**: Proper ARIA attributes for screen readers
- **Touch Support**: Mobile-friendly dragging
- **Theme Integration**: Uses Seed design system tokens
- **Disabled State**: Visual and functional disabled state

## Basic Usage

```tsx
import { Slider } from "@/components/ui/Slider";

function MyComponent() {
  const [value, setValue] = useState(50);

  return (
    <Slider
      value={value}
      onChange={setValue}
      min={0}
      max={100}
      step={10}
      label="Volume"
      formatValue={(v) => `${v}%`}
    />
  );
}
```

## Props

| Prop          | Type                        | Required | Default         | Description                         |
| ------------- | --------------------------- | -------- | --------------- | ----------------------------------- |
| `value`       | `number`                    | Yes      | -               | Current slider value                |
| `onChange`    | `(value: number) => void`   | Yes      | -               | Callback when value changes         |
| `min`         | `number`                    | Yes      | -               | Minimum value                       |
| `max`         | `number`                    | Yes      | -               | Maximum value                       |
| `step`        | `number`                    | Yes      | -               | Step increment                      |
| `label`       | `string`                    | No       | -               | Label displayed above slider        |
| `leftLabel`   | `string`                    | No       | -               | Label at left end (e.g., "Slower")  |
| `rightLabel`  | `string`                    | No       | -               | Label at right end (e.g., "Faster") |
| `formatValue` | `(value: number) => string` | No       | `String(value)` | Custom formatter for value display  |
| `disabled`    | `boolean`                   | No       | `false`         | Disable slider interaction          |
| `className`   | `string`                    | No       | -               | Additional CSS classes              |

## Scanner Settings (from plan)

The component is designed to support the following scanner settings:

| Setting         | Min   | Default | Max     | Step  | Format                          |
| --------------- | ----- | ------- | ------- | ----- | ------------------------------- |
| Probe Interval  | 25ms  | 75ms    | 500ms   | 25ms  | `${v}ms`                        |
| Scan Timeout    | 500ms | 2000ms  | 10000ms | 500ms | `${v >= 1000 ? v/1000 : v}s/ms` |
| Workers         | 5     | 20      | 100     | 5     | `${v} workers`                  |
| Rescan Interval | 1min  | 10min   | 60min   | 1min  | `${v} min`                      |
| Banner Timeout  | 500ms | 2000ms  | 10000ms | 500ms | `${v}ms`                        |

## Examples

### Simple Numeric Slider

```tsx
<Slider
  value={volume}
  onChange={setVolume}
  min={0}
  max={100}
  step={5}
  label="Volume"
  formatValue={(v) => `${v}%`}
/>
```

### Probe Interval (Milliseconds)

```tsx
<Slider
  value={probeInterval}
  onChange={setProbeInterval}
  min={25}
  max={500}
  step={25}
  label="Probe Interval"
  leftLabel="Faster"
  rightLabel="Slower"
  formatValue={(v) => `${v}ms`}
/>
```

### Scan Timeout (Smart ms/s formatting)

```tsx
<Slider
  value={scanTimeout}
  onChange={setScanTimeout}
  min={500}
  max={10000}
  step={500}
  label="Scan Timeout"
  leftLabel="Quick"
  rightLabel="Patient"
  formatValue={(v) => (v >= 1000 ? `${v / 1000}s` : `${v}ms`)}
/>
```

### Worker Threads (Count)

```tsx
<Slider
  value={workers}
  onChange={setWorkers}
  min={5}
  max={100}
  step={5}
  label="Worker Threads"
  leftLabel="Conservative"
  rightLabel="Aggressive"
  formatValue={(v) => `${v} workers`}
/>
```

### Rescan Interval (Minutes)

```tsx
<Slider
  value={rescanInterval}
  onChange={setRescanInterval}
  min={1}
  max={60}
  step={1}
  label="Rescan Interval"
  leftLabel="Frequent"
  rightLabel="Rare"
  formatValue={(v) => `${v} min`}
/>
```

### Disabled State

```tsx
<Slider
  value={timeout}
  onChange={setTimeout}
  min={500}
  max={10000}
  step={500}
  label="Timeout"
  disabled={!isEnabled}
/>
```

## Keyboard Shortcuts

The slider supports the following keyboard shortcuts for power users:

| Key             | Action                  |
| --------------- | ----------------------- |
| Arrow Left/Down | Decrease by `step`      |
| Arrow Right/Up  | Increase by `step`      |
| Page Down       | Decrease by `step * 10` |
| Page Up         | Increase by `step * 10` |
| Home            | Jump to `min`           |
| End             | Jump to `max`           |

## Accessibility

The component includes proper ARIA attributes:

- `aria-label`: Set to the `label` prop or "Slider"
- `aria-valuemin`: Set to `min`
- `aria-valuemax`: Set to `max`
- `aria-valuenow`: Set to current `value`
- `aria-valuetext`: Set to formatted value (e.g., "500ms")
- Focus ring on keyboard focus
- Proper disabled state announced to screen readers

## Styling

The slider uses Seed design system tokens:

- Track: `bg-surface-hover` (unfilled) / `bg-brand-primary` (filled)
- Thumb: `bg-brand-primary` with `border-surface-base`
- Label: `label` class (14px medium)
- Value: `body-small` in `text-brand-primary`
- End labels: `caption` in `text-text-muted`
- Focus ring: `ring-brand-primary`

## Testing

Run the test suite:

```bash
npm test -- Slider.test.tsx
```

The test suite covers:

- Basic rendering
- Value changes
- Keyboard navigation
- Boundary conditions
- Custom formatting
- Disabled state
- ARIA attributes

## Storybook

View all variants in Storybook:

```bash
npm run storybook
```

Navigate to: UI → Slider

## Browser Support

- Chrome/Edge: ✅ Full support
- Firefox: ✅ Full support
- Safari: ✅ Full support
- Mobile browsers: ✅ Touch support included

## Performance

- Memoized with `React.memo` to prevent unnecessary re-renders
- Uses `useCallback` for event handlers
- Native HTML5 range input for optimal performance

## Related Components

- `Input`: For text-based numeric input
- `Card`: Container for settings panels
- `CollapsibleSection`: For grouping multiple sliders

## Migration Guide

If replacing a custom slider implementation:

```tsx
// Before (custom slider)
<div className="slider-container">
  <label>Timeout: {timeout}ms</label>
  <input
    type="range"
    min={500}
    max={10000}
    step={500}
    value={timeout}
    onChange={(e) => setTimeout(Number(e.target.value))}
  />
</div>

// After (Slider component)
<Slider
  value={timeout}
  onChange={setTimeout}
  min={500}
  max={10000}
  step={500}
  label="Timeout"
  formatValue={(v) => `${v}ms`}
/>
```

## Future Enhancements

Potential improvements for future versions:

- [ ] Dual-handle range slider (min/max selection)
- [ ] Vertical orientation support
- [ ] Tick marks at intervals
- [ ] Value tooltip on hover/drag
- [ ] Preset value markers
- [ ] Log scale support for exponential ranges
