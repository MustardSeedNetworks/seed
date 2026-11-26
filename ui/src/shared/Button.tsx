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
 * `disabledReason` is a disabled button that explains itself, replacing seed's
 * GatedButton. With `disabled` and a reason the button is `aria-disabled`, not
 * `disabled`: it stays in the tab order, its click and its Enter/Space
 * activation are swallowed (so a submit button does not submit), it looks
 * disabled, and a Tooltip carries the reason on hover and focus, linked by
 * `aria-describedby`. The reason is also what a keyboard or screen-reader user
 * can reach; touch has no hover, so put it in the surrounding copy too when the
 * button gates a primary flow. Without `disabled` the reason is ignored; with
 * `loading` the button is a plain disabled one.
 *
 * Disabled is visibly disabled: a flat sunken face with disabled-text colour
 * and a not-allowed cursor, not just a fade. A fade makes a primary button
 * read as a secondary one. Say why in a `title` or the surrounding copy.
 *
 * `size` is `xs` (24px, the WCAG 2.5.8 floor, for dense toolbars and table
 * rows), `sm` (32px) or `md` (40px).
 *
 * ButtonLink is the same control as a link: an `<a href>` by default, or a
 * router link through `renderLink`, with the same variants, sizes and icon.
 * Use it where the action navigates; use Button where it acts. It replaces
 * trellis's and niac's LinkButton.
 *
 * There is no permission hook. A product decides whether to render or disable
 * a button at the call site, where the permission is known.
 */
import { LoaderCircle } from 'lucide-react';
import type {
  AnchorHTMLAttributes,
  ButtonHTMLAttributes,
  MouseEvent,
  MouseEventHandler,
  ReactNode,
  Ref,
} from 'react';
import { cx } from './cx';
import { Tooltip } from './Tooltip';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'destructive';
export type ButtonSize = 'xs' | 'sm' | 'md';

type NativeProps = Omit<
  ButtonHTMLAttributes<HTMLButtonElement>,
  'className' | 'children' | 'aria-label'
>;

interface CommonProps extends NativeProps {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Spinner, disabled and `aria-busy`. */
  loading?: boolean;
  /** With `disabled`: stays focusable (`aria-disabled`) and a Tooltip says why. */
  disabledReason?: string;
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
  xs: 'min-h-6 px-2 text-xs',
  sm: 'min-h-8 px-3 text-xs',
  md: 'min-h-10 px-4 text-sm',
};

const iconOnlySizes: Record<ButtonSize, string> = {
  xs: 'h-6 w-6',
  sm: 'h-8 w-8',
  md: 'h-10 w-10',
};

/** The look shared by Button and ButtonLink, so a link reads as the same control. */
export function buttonClass(
  variant: ButtonVariant,
  size: ButtonSize,
  iconOnly: boolean,
  className?: string,
): string {
  return cx(base, variants[variant], iconOnly ? iconOnlySizes[size] : sizes[size], className);
}

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
    disabledReason,
    onClick,
    children: _children,
    ref,
    ...rest
  } = props;
  const explained = disabled === true && !loading && Boolean(disabledReason);
  const button = (
    <button
      ref={ref}
      type={type}
      data-variant={variant}
      disabled={explained ? false : disabled || loading}
      aria-disabled={explained ? true : undefined}
      aria-busy={loading || undefined}
      className={buttonClass(variant, size, iconOnly === true, className)}
      onClick={explained ? (event) => event.preventDefault() : onClick}
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
  return explained ? <Tooltip text={disabledReason}>{button}</Tooltip> : button;
}

/** What `renderLink` receives: spread it onto the router's link component. */
export interface ButtonLinkRenderProps {
  className: string;
  children: ReactNode;
  'data-variant': ButtonVariant;
  'aria-label'?: string;
  'aria-disabled'?: true;
  tabIndex?: -1;
  onClick?: MouseEventHandler<HTMLElement>;
}

interface LinkCommonProps {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Not navigable: rendered `aria-disabled`, out of the tab order, its click swallowed. */
  disabled?: boolean;
  className?: string;
  ref?: Ref<HTMLAnchorElement>;
}

type PlainLinkProps = LinkCommonProps &
  Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'className' | 'children' | 'aria-label'> & {
    /** Renders a router link instead of an `<a>`; omit `href` then, the router owns it. */
    renderLink?: never;
  };

type RoutedLinkProps = LinkCommonProps & {
  renderLink: (props: ButtonLinkRenderProps) => ReactNode;
  href?: never;
};

type LinkContentProps =
  | { iconOnly?: false; icon?: ReactNode; children: ReactNode; 'aria-label'?: string }
  | { iconOnly: true; icon: ReactNode; 'aria-label': string; children?: never };

export type ButtonLinkProps = (PlainLinkProps | RoutedLinkProps) & LinkContentProps;

export function ButtonLink(props: ButtonLinkProps) {
  const {
    variant = 'secondary',
    size = 'md',
    disabled = false,
    className,
    icon,
    iconOnly,
    renderLink,
    children: _children,
    ref,
    ...rest
  } = props;
  const content = (
    <>
      {icon === undefined ? null : <span aria-hidden="true">{icon}</span>}
      {props.iconOnly ? undefined : props.children}
    </>
  );
  const shared = {
    className: buttonClass(variant, size, iconOnly === true, cx('no-underline', className)),
    'data-variant': variant,
    ...(disabled
      ? {
          'aria-disabled': true as const,
          tabIndex: -1 as const,
          onClick: (event: MouseEvent<HTMLElement>) => event.preventDefault(),
        }
      : {}),
  };
  if (renderLink) {
    const label = props['aria-label'];
    return renderLink({
      ...shared,
      ...(label === undefined ? {} : { 'aria-label': label }),
      children: content,
    });
  }
  // Without an href an <a> is inert, so a disabled link needs no click guard.
  const { href, onClick, ...anchor } = rest as AnchorHTMLAttributes<HTMLAnchorElement>;
  const live = disabled ? {} : { href, onClick };
  return (
    <a
      ref={ref}
      {...anchor}
      {...live}
      className={shared.className}
      data-variant={variant}
      {...(disabled ? { role: 'link', 'aria-disabled': true, tabIndex: -1 } : {})}
    >
      {content}
    </a>
  );
}
