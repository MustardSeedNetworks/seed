import type { ReactNode } from 'react';
import { Link } from 'wouter';
import type { BreadcrumbLinkProps } from '../shared/Breadcrumbs';

/** Points the shared PageHeader's breadcrumb links at wouter. */
export function renderBreadcrumbLink({
  href,
  className,
  children,
}: BreadcrumbLinkProps): ReactNode {
  return (
    <Link to={href} className={className}>
      {children}
    </Link>
  );
}
