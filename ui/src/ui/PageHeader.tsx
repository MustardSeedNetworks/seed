import type { LucideIcon } from 'lucide-react';
import { createElement, type FC, type ReactNode } from 'react';

interface PageHeaderProps {
  title: string;
  description?: string;
  icon?: LucideIcon;
  actions?: ReactNode;
  iconColorClass?: string;
}

/**
 * Page-level header (title + description + optional icon and trailing actions).
 * Sits below the breadcrumbs at the top of every routed page.
 */
export const PageHeader: FC<PageHeaderProps> = ({
  title,
  description,
  icon,
  actions,
  iconColorClass = 'text-brand-primary',
}) => (
  <div className="mb-section animate-fade-in">
    <div className="flex flex-wrap items-start justify-between gap-comfortable">
      <div className="flex items-center gap-default">
        {icon ? createElement(icon, { className: `h-8 w-8 ${iconColorClass}` }) : null}
        <div>
          <h1 data-testid="page-header-title" className="heading-1 text-text-primary font-display">
            {title}
          </h1>
          {description ? (
            <p className="text-sm text-text-muted mt-tight max-w-2xl">{description}</p>
          ) : null}
        </div>
      </div>
      {actions ? <div className="flex items-center gap-default">{actions}</div> : null}
    </div>
  </div>
);
