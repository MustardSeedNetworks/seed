/**
 * Typography primitives — ported from niac UI kit (Phase B).
 *
 * One canonical class set per visual level so the app doesn't drift into
 * a dozen "almost h2" inline classNames.
 */
import type { HTMLAttributes, ReactNode } from 'react';
import { Link } from 'wouter';

interface TypographyProps extends HTMLAttributes<HTMLElement> {
  children: ReactNode;
  className?: string;
}

export function H1({ children, className = '', ...props }: TypographyProps) {
  return (
    <h1 className={`heading-1 text-text-primary ${className}`} {...props}>
      {children}
    </h1>
  );
}

export function H2({ children, className = '', ...props }: TypographyProps) {
  return (
    <h2 className={`heading-2 text-text-primary ${className}`} {...props}>
      {children}
    </h2>
  );
}

export function H3({ children, className = '', ...props }: TypographyProps) {
  return (
    <h3 className={`heading-3 text-text-primary ${className}`} {...props}>
      {children}
    </h3>
  );
}

export function H4({ children, className = '', ...props }: TypographyProps) {
  return (
    <h4 className={`text-sm font-semibold text-text-primary ${className}`} {...props}>
      {children}
    </h4>
  );
}

export function P({ children, className = '', ...props }: TypographyProps) {
  return (
    <p className={`text-text-secondary leading-relaxed ${className}`} {...props}>
      {children}
    </p>
  );
}

export function SmallText({ children, className = '', ...props }: TypographyProps) {
  return (
    <span className={`text-sm text-text-muted ${className}`} {...props}>
      {children}
    </span>
  );
}

export function Caption({ children, className = '', ...props }: TypographyProps) {
  return (
    <span className={`text-xs text-text-muted ${className}`} {...props}>
      {children}
    </span>
  );
}

interface AccentLinkProps extends TypographyProps {
  href?: string;
  to?: string;
  onClick?: () => void;
}

export function AccentLink({
  children,
  className = '',
  href,
  to,
  onClick,
  ...props
}: AccentLinkProps) {
  const linkClass = `text-brand-accent hover:text-brand-accent underline underline-offset-2 transition-colors ${className}`;

  if (to) {
    return (
      <Link to={to} className={linkClass} {...props}>
        {children}
      </Link>
    );
  }
  if (href) {
    return (
      <a href={href} className={linkClass} {...props}>
        {children}
      </a>
    );
  }
  return (
    <button type="button" onClick={onClick} className={linkClass} {...props}>
      {children}
    </button>
  );
}
