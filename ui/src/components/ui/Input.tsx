/**
 * Input / Textarea / Select / Checkbox / Toggle / SearchInput / FormGroup / FormSection
 * primitives — ported from niac UI kit (Phase B).
 *
 * React 19: refs are regular props.
 */
import type { FC, InputHTMLAttributes, ReactNode, Ref, TextareaHTMLAttributes } from 'react';

const inputBaseStyles =
  'w-full rounded-lg border bg-bg-base/60 text-text-primary placeholder:text-text-muted transition-all focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed';
const inputFocusStyles = 'focus:border-brand-primary focus:ring-2 focus:ring-brand-primary/20';
const inputBorderStyles = 'border-surface-border hover:border-surface-border';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
  leftIcon?: ReactNode;
  rightIcon?: ReactNode;
  containerClassName?: string;
  ref?: Ref<HTMLInputElement>;
}

export const Input: FC<InputProps> = ({
  label,
  error,
  hint,
  leftIcon,
  rightIcon,
  className = '',
  containerClassName = '',
  id,
  ref,
  ...props
}) => {
  const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
  const hasError = !!error;

  return (
    <div class={containerClassName}>
      {label ? (
        <label for={inputId} class="block text-sm font-medium text-text-secondary mb-2">
          {label}
        </label>
      ) : null}
      <div class="relative">
        {leftIcon ? (
          <div class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted">{leftIcon}</div>
        ) : null}
        <input
          ref={ref}
          id={inputId}
          class={`
            ${inputBaseStyles}
            ${hasError ? 'border-status-error focus:border-status-error focus:ring-status-error/20' : `${inputBorderStyles} ${inputFocusStyles}`}
            ${leftIcon ? 'pl-10' : 'px-4'}
            ${rightIcon ? 'pr-10' : 'px-4'}
            py-2.5
            ${className}
          `}
          {...props}
        />
        {rightIcon ? (
          <div class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted">{rightIcon}</div>
        ) : null}
      </div>
      {error || hint ? (
        <p class={`mt-1.5 text-sm ${hasError ? 'text-status-error' : 'text-text-muted'}`}>
          {error || hint}
        </p>
      ) : null}
    </div>
  );
};

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
  hint?: string;
  containerClassName?: string;
  ref?: Ref<HTMLTextAreaElement>;
}

export const Textarea: FC<TextareaProps> = ({
  label,
  error,
  hint,
  className = '',
  containerClassName = '',
  id,
  ref,
  ...props
}) => {
  const textareaId = id || label?.toLowerCase().replace(/\s+/g, '-');
  const hasError = !!error;

  return (
    <div class={containerClassName}>
      {label ? (
        <label for={textareaId} class="block text-sm font-medium text-text-secondary mb-2">
          {label}
        </label>
      ) : null}
      <textarea
        ref={ref}
        id={textareaId}
        class={`
          ${inputBaseStyles}
          ${hasError ? 'border-status-error focus:border-status-error focus:ring-status-error/20' : `${inputBorderStyles} ${inputFocusStyles}`}
          px-4 py-2.5 min-h-[100px] resize-y
          ${className}
        `}
        {...props}
      />
      {error || hint ? (
        <p class={`mt-1.5 text-sm ${hasError ? 'text-status-error' : 'text-text-muted'}`}>
          {error || hint}
        </p>
      ) : null}
    </div>
  );
};

interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface SelectProps extends Omit<InputHTMLAttributes<HTMLSelectElement>, 'onChange'> {
  label?: string;
  error?: string;
  hint?: string;
  options: SelectOption[];
  placeholder?: string;
  containerClassName?: string;
  onChange?: (value: string) => void;
  ref?: Ref<HTMLSelectElement>;
}

export const Select: FC<SelectProps> = ({
  label,
  error,
  hint,
  options,
  placeholder,
  className = '',
  containerClassName = '',
  id,
  onChange,
  ref,
  ...props
}) => {
  const selectId = id || label?.toLowerCase().replace(/\s+/g, '-');
  const hasError = !!error;

  return (
    <div class={containerClassName}>
      {label ? (
        <label for={selectId} class="block text-sm font-medium text-text-secondary mb-2">
          {label}
        </label>
      ) : null}
      <select
        ref={ref}
        id={selectId}
        class={`
          ${inputBaseStyles}
          ${hasError ? 'border-status-error focus:border-status-error focus:ring-status-error/20' : `${inputBorderStyles} ${inputFocusStyles}`}
          px-4 py-2.5 appearance-none cursor-pointer
          bg-[url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20fill%3D%22none%22%20viewBox%3D%220%200%2024%2024%22%20stroke%3D%22%239ca3af%22%3E%3Cpath%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%20stroke-width%3D%222%22%20d%3D%22M19%209l-7%207-7-7%22%2F%3E%3C%2Fsvg%3E')]
          bg-[length:1.25rem] bg-[right_0.75rem_center] bg-no-repeat pr-10
          ${className}
        `}
        onChange={(e) => onChange?.(e.target.value)}
        {...props}
      >
        {placeholder ? (
          <option value="" disabled={true}>
            {placeholder}
          </option>
        ) : null}
        {options.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>
            {option.label}
          </option>
        ))}
      </select>
      {error || hint ? (
        <p class={`mt-1.5 text-sm ${hasError ? 'text-status-error' : 'text-text-muted'}`}>
          {error || hint}
        </p>
      ) : null}
    </div>
  );
};

interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: string;
  description?: string;
  containerClassName?: string;
  ref?: Ref<HTMLInputElement>;
}

export const Checkbox: FC<CheckboxProps> = ({
  label,
  description,
  className = '',
  containerClassName = '',
  id,
  ref,
  ...props
}) => {
  const checkboxId = id || label.toLowerCase().replace(/\s+/g, '-');

  return (
    <div class={`flex items-start gap-3 ${containerClassName}`}>
      <input
        ref={ref}
        type="checkbox"
        id={checkboxId}
        class={`
          mt-0.5 h-4 w-4 rounded border-border-muted bg-bg-elevated text-brand-primary
          focus:ring-2 focus:ring-brand-primary/50 focus:ring-offset-0
          transition-colors cursor-pointer
          ${className}
        `}
        {...props}
      />
      <div>
        <label for={checkboxId} class="text-sm font-medium text-text-primary cursor-pointer">
          {label}
        </label>
        {description ? <p class="text-sm text-text-muted mt-0.5">{description}</p> : null}
      </div>
    </div>
  );
};

interface ToggleProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: string;
  description?: string;
  containerClassName?: string;
  ref?: Ref<HTMLInputElement>;
}

export const Toggle: FC<ToggleProps> = ({
  label,
  description,
  className = '',
  containerClassName = '',
  id,
  checked,
  ref,
  ...props
}) => {
  const toggleId = id || label.toLowerCase().replace(/\s+/g, '-');

  return (
    <div class={`flex items-center justify-between gap-4 ${containerClassName}`}>
      <div>
        <label for={toggleId} class="text-sm font-medium text-text-primary cursor-pointer">
          {label}
        </label>
        {description ? <p class="text-sm text-text-muted mt-0.5">{description}</p> : null}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => {
          const input = document.getElementById(toggleId) as HTMLInputElement;
          if (input) {
            input.click();
          }
        }}
        class={`
          relative inline-flex h-6 w-11 items-center rounded-full transition-colors
          focus:outline-none focus:ring-2 focus:ring-brand-primary/50 focus:ring-offset-2 focus:ring-offset-gray-900
          ${checked ? 'bg-brand-primary' : 'bg-bg-elevated'}
          ${className}
        `}
      >
        <span
          class={`
            inline-block h-4 w-4 transform rounded-full bg-white shadow-lg transition-transform
            ${checked ? 'translate-x-6' : 'translate-x-1'}
          `}
        />
      </button>
      <input ref={ref} type="checkbox" id={toggleId} checked={checked} class="sr-only" {...props} />
    </div>
  );
};

interface SearchInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label?: string;
  onClear?: () => void;
  containerClassName?: string;
  ref?: Ref<HTMLInputElement>;
}

export const SearchInput: FC<SearchInputProps> = ({
  label,
  onClear,
  className = '',
  containerClassName = '',
  id,
  value,
  onChange,
  ref,
  ...props
}) => {
  const inputId = id || label?.toLowerCase().replace(/\s+/g, '-') || 'search-input';
  const hasValue = value !== undefined && value !== '';

  const handleClear = () => {
    if (onClear) {
      onClear();
    } else if (onChange) {
      const syntheticEvent = {
        target: { value: '' },
        currentTarget: { value: '' },
      } as React.ChangeEvent<HTMLInputElement>;
      onChange(syntheticEvent);
    }
  };

  return (
    <div class={containerClassName}>
      {label ? (
        <label for={inputId} class="block text-sm font-medium text-text-secondary mb-2">
          {label}
        </label>
      ) : null}
      <div class="relative">
        <div class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-4 w-4"
            aria-hidden="true"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
        </div>
        <input
          ref={ref}
          id={inputId}
          type="search"
          value={value}
          onChange={onChange}
          placeholder={props.placeholder || 'Search...'}
          class={`
            ${inputBaseStyles}
            ${inputBorderStyles}
            ${inputFocusStyles}
            pl-10
            ${hasValue ? 'pr-10' : 'pr-4'}
            py-2.5
            ${className}
          `}
          {...props}
        />
        {hasValue ? (
          <button
            type="button"
            onClick={handleClear}
            aria-label="Clear search"
            class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary/50 rounded"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-4 w-4"
              aria-hidden="true"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        ) : null}
      </div>
    </div>
  );
};

interface FormGroupProps {
  children: ReactNode;
  className?: string;
}

export const FormGroup: FC<FormGroupProps> = ({ children, className = '' }) => (
  <div class={`space-y-4 ${className}`}>{children}</div>
);

interface FormSectionProps {
  title: string;
  description?: string;
  children: ReactNode;
  className?: string;
}

export const FormSection: FC<FormSectionProps> = ({
  title,
  description,
  children,
  className = '',
}) => (
  <div class={`space-y-4 ${className}`}>
    <div>
      <h3 class="text-lg font-semibold text-text-primary">{title}</h3>
      {description ? <p class="text-sm text-text-muted mt-1">{description}</p> : null}
    </div>
    <div class="space-y-4">{children}</div>
  </div>
);
