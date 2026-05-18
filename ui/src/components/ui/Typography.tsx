/**
 * Typography primitives — ported from niac UI kit (Phase B).
 *
 * One canonical class set per visual level so the app doesn't drift into
 * a dozen "almost h2" inline classNames.
 */
import type { FC, HTMLAttributes, ReactNode } from 'react';
import { Link } from 'react-router-dom';

interface TypographyProps extends HTMLAttributes<HTMLElement> {
  children: ReactNode;
  className?: string;
}

export const H1: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h1 class={`text-2xl font-bold text-text-primary ${className}`} {...props}>
    {children}
  </h1>
);

export const H2: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h2 class={`text-xl font-semibold text-text-primary ${className}`} {...props}>
    {children}
  </h2>
);

export const H3: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h3 class={`text-lg font-semibold text-text-primary ${className}`} {...props}>
    {children}
  </h3>
);

export const H4: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h4 class={`text-sm font-semibold text-text-primary ${className}`} {...props}>
    {children}
  </h4>
);

export const P: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <p class={`text-text-secondary leading-relaxed ${className}`} {...props}>
    {children}
  </p>
);

export const SmallText: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <span class={`text-sm text-text-muted ${className}`} {...props}>
    {children}
  </span>
);

export const Caption: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <span class={`text-xs text-text-muted ${className}`} {...props}>
    {children}
  </span>
);

interface AccentLinkProps extends TypographyProps {
  href?: string;
  to?: string;
  onClick?: () => void;
}

export const AccentLink: FC<AccentLinkProps> = ({
  children,
  className = '',
  href,
  to,
  onClick,
  ...props
}) => {
  const linkClass = `text-brand-accent hover:text-brand-accent underline underline-offset-2 transition-colors ${className}`;

  if (to) {
    return (
      <Link to={to} class={linkClass} {...props}>
        {children}
      </Link>
    );
  }
  if (href) {
    return (
      <a href={href} class={linkClass} {...props}>
        {children}
      </a>
    );
  }
  return (
    <button type="button" onClick={onClick} class={linkClass} {...props}>
      {children}
    </button>
  );
};
