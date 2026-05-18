import { ChevronRight, Home } from 'lucide-react';
import type { FC } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { iconSizes } from '../constants/sizes';

interface BreadcrumbItem {
  label: string;
  path: string;
}

const ROUTE_LABELS: Record<string, string> = {
  '/': 'Link',
  '/link': 'Link',
  '/network': 'Network',
  '/path': 'Path Analysis',
  '/wifi': 'Wi-Fi',
  '/security': 'Security',
  '/performance': 'Performance',
  '/reports': 'Reports',
  '/logs': 'Logs',
};

export const Breadcrumbs: FC = () => {
  const location = useLocation();
  const pathSegments = location.pathname.split('/').filter(Boolean);

  if (pathSegments.length === 0) {
    return null;
  }

  const items: BreadcrumbItem[] = [];
  let currentPath = '';
  for (const segment of pathSegments) {
    currentPath += `/${segment}`;
    const label = ROUTE_LABELS[currentPath] ?? segment.replace(/-/g, ' ');
    items.push({ label, path: currentPath });
  }

  return (
    <nav aria-label="Breadcrumb" class="flex items-center gap-1 text-sm text-text-muted mb-4">
      <Link
        to="/"
        class="flex items-center gap-1 hover:text-text-primary transition-colors"
        aria-label="Home"
      >
        <Home class={iconSizes.sm} />
      </Link>
      {items.map((item, index) => (
        <span key={item.path} class="flex items-center gap-1">
          <ChevronRight class={`${iconSizes.xs} text-text-muted`} />
          {index === items.length - 1 ? (
            <span class="text-text-primary font-medium capitalize" aria-current="page">
              {item.label}
            </span>
          ) : (
            <Link to={item.path} class="hover:text-text-primary transition-colors capitalize">
              {item.label}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
};
