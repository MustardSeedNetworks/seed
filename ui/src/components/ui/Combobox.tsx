/**
 * Combobox — generic typeahead select built on cmdk for consistency
 * with the command palette.
 *
 * Keyboard accessible (arrow keys, enter, escape). Lightweight wrapper
 * around `cmdk` so the same look-and-feel applies wherever the user
 * picks from a list.
 *
 * Example:
 *   <Combobox<Iface>
 *     value={selected}
 *     onChange={setSelected}
 *     options={interfaces}
 *     getKey={(o) => o.name}
 *     getLabel={(o) => `${o.name} (${o.speed} Mbps)`}
 *     placeholder="Select interface…"
 *   />
 */
import { Command } from 'cmdk';
import { Check, ChevronsUpDown } from 'lucide-react';
import { type ReactNode, useEffect, useRef, useState } from 'react';

export interface ComboboxProps<T> {
  value: T | null;
  onChange: (next: T) => void;
  options: T[];
  getKey: (option: T) => string;
  getLabel: (option: T) => string;
  renderItem?: (option: T, isSelected: boolean) => ReactNode;
  placeholder?: string;
  emptyText?: string;
  className?: string;
  /** ARIA label so screen-reader users know what the combobox controls. */
  ariaLabel?: string;
  disabled?: boolean;
}

export function Combobox<T>({
  value,
  onChange,
  options,
  getKey,
  getLabel,
  renderItem,
  placeholder = 'Select…',
  emptyText = 'No matches.',
  className = '',
  ariaLabel = 'Combobox',
  disabled = false,
}: ComboboxProps<T>): React.JSX.Element {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const containerRef = useRef<HTMLDivElement>(null);

  // Close on outside click.
  useEffect(() => {
    if (!open) {
      return;
    }
    const onClick = (e: MouseEvent): void => {
      if (!containerRef.current?.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, [open]);

  useEffect(() => {
    if (!open) {
      setQuery('');
    }
  }, [open]);

  const selectedLabel = value ? getLabel(value) : '';

  const handleSelect = (option: T): void => {
    onChange(option);
    setOpen(false);
  };

  return (
    <div ref={containerRef} class={`relative inline-block w-full ${className}`}>
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={ariaLabel}
        disabled={disabled}
        onClick={() => setOpen((v) => !v)}
        class="flex w-full items-center justify-between gap-2 rounded-lg border border-surface-border bg-bg-base/60 px-3 py-2 text-sm text-text-primary transition-colors hover:border-surface-border focus:outline-none focus:ring-2 focus:ring-brand-primary/30 disabled:cursor-not-allowed disabled:opacity-50"
      >
        <span class={selectedLabel ? '' : 'text-text-muted'}>{selectedLabel || placeholder}</span>
        <ChevronsUpDown class="h-4 w-4 text-text-muted" aria-hidden="true" />
      </button>

      {open ? (
        <div class="absolute left-0 right-0 z-40 mt-1 rounded-lg border border-surface-border bg-bg-surface shadow-xl">
          <Command shouldFilter={true} class="overflow-hidden rounded-lg">
            <Command.Input
              autoFocus={true}
              value={query}
              onValueChange={setQuery}
              placeholder="Search…"
              class="w-full bg-transparent px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none border-b border-surface-border"
            />
            <Command.List class="max-h-60 overflow-y-auto py-1 text-sm">
              <Command.Empty class="px-3 py-4 text-center text-text-muted">
                {emptyText}
              </Command.Empty>
              {options.map((option) => {
                const key = getKey(option);
                const label = getLabel(option);
                const isSelected = value !== null && getKey(value) === key;
                return (
                  <Command.Item
                    key={key}
                    value={`${label} ${key}`}
                    onSelect={() => handleSelect(option)}
                    class="flex cursor-pointer items-center gap-2 px-3 py-2 text-text-primary aria-selected:bg-surface-hover"
                  >
                    <span class="flex-1">
                      {renderItem !== undefined
                        ? (renderItem(option, isSelected) ?? null)
                        : label || null}
                    </span>
                    {isSelected ? (
                      <Check class="h-4 w-4 text-brand-accent" aria-hidden="true" />
                    ) : null}
                  </Command.Item>
                );
              })}
            </Command.List>
          </Command>
        </div>
      ) : null}
    </div>
  );
}
