import { ChevronRight, Home } from 'lucide-react';
import type { FC } from 'react';
import { Link, useLocation } from 'wouter';
import { iconSizes } from '../constants/sizes';
import { usePages } from '../pageRegistry';

interface BreadcrumbItem {
  label: string;
  path: string;
}

/**
 * Labels come from the route table, never from a list kept here. The hard-coded
 * label map this replaced knew nine of the eleven routes and none of the
 * three newest, so `/polling-targets` fell through to a de-slugged segment and
 * read "Polling Targets" over an H1 of "Polling targets" — a breadcrumb
 * disagreeing with the heading directly beneath it (#2645). It was also
 * untranslated, so every breadcrumb stayed English under `es`.
 */
export const Breadcrumbs: FC = () => {
  const [location] = useLocation();
  const pages = usePages();
  const pathSegments = location.split('/').filter(Boolean);

  if (pathSegments.length === 0) {
    return null;
  }

  const items: BreadcrumbItem[] = [];
  let currentPath = '';
  for (const segment of pathSegments) {
    currentPath += `/${segment}`;
    // A segment with no page of its own (an intermediate path, or a route
    // added without a registry entry) still has to render something; the
    // de-slugged segment is the fallback, as before.
    const label =
      pages.find((page) => page.path === currentPath)?.label ?? segment.replace(/-/g, ' ');
    items.push({ label, path: currentPath });
  }

  return (
    <nav
      aria-label="Breadcrumb"
      className="flex items-center gap-tight text-sm text-text-muted mb-content"
    >
      <Link
        to="/"
        // The icon is 14px; without a floor the whole link is a 14x14 target,
        // which fails WCAG 2.5.8 at any width and is simply hard to hit (#244).
        className="target flex-center hover:text-text-primary transition-colors"
        aria-label="Home"
      >
        <Home className={iconSizes.sm} />
      </Link>
      {items.map((item, index) => (
        <span key={item.path} className="flex items-center gap-tight">
          <ChevronRight className={`${iconSizes.xs} text-text-muted`} />
          {index === items.length - 1 ? (
            <span className="text-text-primary font-medium capitalize" aria-current="page">
              {item.label}
            </span>
          ) : (
            <Link to={item.path} className="hover:text-text-primary transition-colors capitalize">
              {item.label}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
};
