/**
 * Breadcrumbs: the trail above a page header. Every page has one.
 *
 * The caller passes the trail already translated, and a `renderLink` that
 * wraps its router's Link (seed uses wouter, the others react-router), so
 * nothing here knows which router is in play. The last item is the current
 * page: it is text, not a link, and carries `aria-current="page"`.
 */
import { ChevronRight } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from './cx';

export interface BreadcrumbItem {
  label: string;
  /** Omit on the current page. */
  href?: string;
}

export interface BreadcrumbLinkProps {
  href: string;
  className: string;
  children: ReactNode;
}

export interface BreadcrumbsProps {
  items: readonly BreadcrumbItem[];
  /** Accessible name of the nav landmark, e.g. "Breadcrumb". */
  label: string;
  renderLink?: (props: BreadcrumbLinkProps) => ReactNode;
  className?: string;
}

const linkClass =
  'inline-flex min-h-6 items-center rounded text-text-muted hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary';

function defaultLink({ href, className, children }: BreadcrumbLinkProps): ReactNode {
  return (
    <a href={href} className={className}>
      {children}
    </a>
  );
}

export function Breadcrumbs({
  items,
  label,
  renderLink = defaultLink,
  className,
}: BreadcrumbsProps) {
  const lastIndex = items.length - 1;
  return (
    <nav aria-label={label} className={className} data-testid="breadcrumbs">
      <ol className="flex flex-wrap items-center gap-x-1 gap-y-0.5 text-sm">
        {items.map((item, index) => (
          <li key={`${item.href ?? ''}|${item.label}`} className="flex min-w-0 items-center gap-1">
            {index > 0 ? (
              <ChevronRight className="h-3.5 w-3.5 shrink-0 text-text-muted" aria-hidden="true" />
            ) : null}
            {index === lastIndex || item.href === undefined ? (
              <span
                className="min-w-0 truncate font-medium text-text-secondary"
                aria-current={index === lastIndex ? 'page' : undefined}
              >
                {item.label}
              </span>
            ) : (
              renderLink({
                href: item.href,
                className: cx(linkClass, 'min-w-0 truncate'),
                children: item.label,
              })
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
}
