/**
 * SegmentedControl: choose one of two to five short options that change the
 * view in place (My configs / Upload, Graph / Table). Native radio inputs under
 * a labelled fieldset, so the arrow keys, the single Tab stop and the screen
 * reader's "1 of 3" come from the platform. For switching between page
 * sections use SettingsTabs or navigation, not this.
 */
import { type ReactNode, useId } from 'react';
import { cx } from './cx';

export interface SegmentedOption {
  value: string;
  label: string;
  icon?: ReactNode;
}

export interface SegmentedControlProps {
  options: readonly SegmentedOption[];
  value: string;
  onChange: (value: string) => void;
  /** Accessible name of the group, e.g. "View". */
  label: string;
  className?: string;
}

export function SegmentedControl({
  options,
  value,
  onChange,
  label,
  className,
}: SegmentedControlProps) {
  const name = useId();
  return (
    <fieldset
      className={cx(
        'inline-flex min-w-0 max-w-full rounded-lg border border-surface-border bg-surface-sunken p-0.5',
        className,
      )}
    >
      <legend className="sr-only">{label}</legend>
      {options.map((option) => (
        <label
          key={option.value}
          className="relative inline-flex min-h-8 min-w-0 cursor-pointer items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium has-[:checked]:bg-surface-raised has-[:checked]:text-text-primary has-[:checked]:shadow-sm has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-brand-primary text-text-muted hover:text-text-primary"
        >
          <input
            type="radio"
            name={name}
            value={option.value}
            checked={option.value === value}
            onChange={() => onChange(option.value)}
            className="sr-only"
          />
          {option.icon ? <span aria-hidden="true">{option.icon}</span> : null}
          <span className="truncate">{option.label}</span>
        </label>
      ))}
    </fieldset>
  );
}
