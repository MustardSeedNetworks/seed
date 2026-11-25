/**
 * Button: four variants, a loading state, an icon-only form and one disabled
 * look.
 *
 * - primary: the page's one most-likely action. At most one per page; a page
 *   with no single likely action has none.
 * - secondary: everything else.
 * - ghost: borderless, for toolbar and low-emphasis actions (used by seed,
 *   stem and niac today). Never the only way to reach something important.
 * - destructive: anything that deletes or cannot be undone. Filled, never an
 *   outlined or text-only red: a destructive action is styled one way across
 *   the fleet. Pair it with a ConfirmDialog.
 *
 * `loading` shows a spinner, disables the button and sets `aria-busy`, so a
 * slow action cannot be double-submitted. Keep the label: it names the action.
 *
 * `iconOnly` renders a square button with just an icon. The type then requires
 * an `aria-label`, because an icon is not a name.
 *
 * `data-variant` is rendered so "one primary per page" is a queryable fact
 * (`[data-variant="primary"]`) rather than a source grep.
 *
 * Disabled is visibly disabled: a flat sunken face with disabled-text colour
 * and a not-allowed cursor, not just a fade. A fade makes a primary button
 * read as a secondary one. Say why in a `title` or the surrounding copy.
 *
 * There is no permission hook. A product decides whether to render or disable
 * a button at the call site, where the permission is known.
 */
import { LoaderCircle } from 'lucide-react';
import type { ButtonHTMLAttributes, ReactNode, Ref } from 'react';
import { cx } from './cx';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'destructive';
export type ButtonSize = 'sm' | 'md';

type NativeProps = Omit<
  ButtonHTMLAttributes<HTMLButtonElement>,
  'className' | 'children' | 'aria-label'
>;

interface CommonProps extends NativeProps {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Spinner, disabled and `aria-busy`. */
  loading?: boolean;
  className?: string;
  ref?: Ref<HTMLButtonElement>;
}

interface LabelledButtonProps extends CommonProps {
  iconOnly?: false;
  /** A decorative icon before the label. */
  icon?: ReactNode;
  children: ReactNode;
  'aria-label'?: string;
}

interface IconOnlyButtonProps extends CommonProps {
  iconOnly: true;
  icon: ReactNode;
  /** Required: an icon is not an accessible name. */
  'aria-label': string;
  children?: never;
}

export type ButtonProps = LabelledButtonProps | IconOnlyButtonProps;

const base =
  'inline-flex items-center justify-center gap-2 rounded-lg border font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:border-hairline disabled:bg-surface-sunken disabled:text-text-disabled aria-disabled:cursor-not-allowed aria-disabled:border-hairline aria-disabled:bg-surface-sunken aria-disabled:text-text-disabled';

const variants: Record<ButtonVariant, string> = {
  primary:
    'border-transparent bg-brand-primary text-on-brand hover:bg-brand-primary/90 disabled:hover:bg-surface-sunken aria-disabled:hover:bg-surface-sunken',
  secondary:
    'border-surface-border bg-surface-raised text-text-primary hover:bg-surface-hover disabled:hover:bg-surface-sunken aria-disabled:hover:bg-surface-sunken',
  ghost:
    'border-transparent bg-transparent text-text-secondary hover:bg-surface-hover hover:text-text-primary disabled:hover:bg-surface-sunken aria-disabled:hover:bg-surface-sunken',
  destructive:
    'border-transparent bg-status-error text-on-danger hover:bg-status-error/90 disabled:hover:bg-surface-sunken aria-disabled:hover:bg-surface-sunken',
};

const sizes: Record<ButtonSize, string> = {
  sm: 'min-h-8 px-3 text-xs',
  md: 'min-h-10 px-4 text-sm',
};

const iconOnlySizes: Record<ButtonSize, string> = {
  sm: 'h-8 w-8',
  md: 'h-10 w-10',
};

export function Button(props: ButtonProps) {
  const {
    variant = 'secondary',
    size = 'md',
    loading = false,
    icon,
    iconOnly,
    className,
    type = 'button',
    disabled,
    children: _children,
    ref,
    ...rest
  } = props;
  return (
    <button
      ref={ref}
      type={type}
      data-variant={variant}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cx(
        base,
        variants[variant],
        iconOnly ? iconOnlySizes[size] : sizes[size],
        className,
      )}
      {...rest}
    >
      {loading ? (
        <LoaderCircle className="h-4 w-4 motion-safe:animate-spin" aria-hidden="true" />
      ) : icon === undefined ? null : (
        <span aria-hidden="true">{icon}</span>
      )}
      {props.iconOnly ? undefined : props.children}
    </button>
  );
}
