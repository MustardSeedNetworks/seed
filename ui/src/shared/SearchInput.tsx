/**
 * SearchInput: a filter box for a list or table. A search-typed input with a
 * magnifier and a clear button, named by `label` (there is no visible label, so
 * it is required). Controlled; `onValueChange` gets the text. Debounce in the
 * caller if the filter is expensive.
 */
import { Search, X } from 'lucide-react';
import type { Ref } from 'react';
import { cx } from './cx';
import { controlClass } from './Input';

export interface SearchInputProps {
  value: string;
  onValueChange: (value: string) => void;
  /** Accessible name, e.g. "Search devices". */
  label: string;
  /** Accessible name of the clear button, e.g. "Clear search". */
  clearLabel: string;
  placeholder?: string;
  className?: string;
  ref?: Ref<HTMLInputElement>;
}

export function SearchInput({
  value,
  onValueChange,
  label,
  clearLabel,
  placeholder,
  className,
  ref,
}: SearchInputProps) {
  return (
    <div className={cx('relative w-full max-w-md', className)}>
      <Search
        className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted"
        aria-hidden="true"
      />
      <input
        ref={ref}
        type="search"
        value={value}
        placeholder={placeholder}
        aria-label={label}
        onChange={(event) => onValueChange(event.target.value)}
        className={cx(controlClass, 'pl-9 pr-9 [&::-webkit-search-cancel-button]:appearance-none')}
      />
      {value === '' ? null : (
        <button
          type="button"
          onClick={() => onValueChange('')}
          aria-label={clearLabel}
          className="absolute right-1.5 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-md text-text-muted hover:bg-surface-sunken hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      )}
    </div>
  );
}
