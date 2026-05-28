// SYNCED FROM stem@6a5de95ca0a1 — DO NOT EDIT.
// Edits in this repo will be overwritten on next `make sync-shell`.
// To change this file, send a PR to stem then re-sync.
/**
 * PageHeader — page-level title bar with optional breadcrumbs, actions,
 * and a slide-out help panel.
 *
 * CANONICAL SHELL — owned by stem; seed and niac sync this file via
 * scripts/sync-shell.sh. Edits made downstream will be overwritten on
 * next sync. All colors/spacing reference theme tokens.
 *
 * Usage:
 *   <PageHeader
 *     title="Devices"
 *     description="Active simulated devices in this NIAC instance"
 *     icon={ServerIcon}
 *     breadcrumbs={[{ label: 'Home', href: '/' }, { label: 'Devices' }]}
 *     actions={<Button>Add device</Button>}
 *     help={<p>Detailed help content shown in a side panel.</p>}
 *   />
 */
import type { LucideIcon } from 'lucide-react';
import { ChevronRight, HelpCircle, X } from 'lucide-react';
import { createElement, type FC, type ReactNode, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { iconSizes } from '../constants/sizes';

interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface PageHeaderProps {
  title: string;
  description?: string;
  icon?: LucideIcon;
  iconColorClass?: string;
  actions?: ReactNode;
  breadcrumbs?: BreadcrumbItem[];
  /**
   * Rich help content shown in a side-panel when the user clicks the (?) icon.
   * Pass any ReactNode (paragraphs, lists, links). Omit to hide the button.
   */
  help?: ReactNode;
  className?: string;
}

interface BreadcrumbProps {
  items: BreadcrumbItem[];
  className?: string;
}

const Breadcrumb: FC<BreadcrumbProps> = ({ items, className = '' }) => (
  <nav className={`flex items-center gap-tight text-sm ${className}`} aria-label="Breadcrumb">
    {items.map((item, index) => (
      <div key={item.label} className="flex items-center gap-tight">
        {index > 0 && <ChevronRight className={`${iconSizes.md} text-text-disabled`} />}
        {item.href ? (
          <Link
            to={item.href}
            className="text-text-muted hover:text-text-primary transition-colors"
          >
            {item.label}
          </Link>
        ) : (
          <span className="text-text-secondary font-medium">{item.label}</span>
        )}
      </div>
    ))}
  </nav>
);

interface HelpPanelProps {
  title: string;
  children: ReactNode;
  onClose: () => void;
}

/**
 * HelpPanel — fixed side panel triggered by the PageHeader's (?) button.
 * Closes on Escape, overlay click, or X. Content is opaque ReactNode so
 * each page can ship its own help with formatting/links/inline code.
 */
const HelpPanel: FC<HelpPanelProps> = ({ title, children, onClose }) => {
  useEffect(() => {
    const onKey = (e: KeyboardEvent): void => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return (): void => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex justify-end bg-scrim/40 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label={`Help: ${title}`}
    >
      <button
        type="button"
        aria-label="Close help (overlay)"
        className="absolute inset-0 cursor-default"
        onClick={onClose}
      />
      <aside className="relative h-full w-full max-w-md overflow-y-auto bg-surface-raised pad-lg shadow-2xl">
        <div className="mb-content flex-between">
          <h2 className="heading-3">{title}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close help"
            className="rounded p-1 text-text-muted hover:bg-surface-hover hover:text-text-primary"
          >
            <X className={iconSizes.lg} />
          </button>
        </div>
        <div className="text-text-primary">{children}</div>
      </aside>
    </div>
  );
};

export const PageHeader: FC<PageHeaderProps> = ({
  title,
  description,
  icon,
  iconColorClass = 'text-brand-primary',
  actions,
  breadcrumbs,
  help,
  className = '',
}) => {
  const [helpOpen, setHelpOpen] = useState(false);

  return (
    <div className={`mb-section animate-fade-in ${className}`}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumb items={breadcrumbs} className="mb-heading" />
      )}
      <div className="flex flex-wrap items-start justify-between gap-comfortable">
        <div className="flex items-center gap-default">
          {icon ? createElement(icon, { className: `h-8 w-8 ${iconColorClass}` }) : null}
          <div>
            <h1 className="heading-1 font-display" data-testid="page-header-title">
              {title}
            </h1>
            {description ? <p className="body-small mt-tight max-w-2xl">{description}</p> : null}
          </div>
        </div>
        <div className="flex items-center gap-default">
          {actions}
          {help ? (
            <button
              type="button"
              onClick={() => setHelpOpen(true)}
              aria-label={`Open help for ${title}`}
              title={`What is ${title}?`}
              className="rounded-full p-1.5 text-text-muted hover:bg-surface-hover hover:text-text-primary"
            >
              <HelpCircle className={iconSizes.lg} />
            </button>
          ) : null}
        </div>
      </div>
      {help && helpOpen ? (
        <HelpPanel title={title} onClose={() => setHelpOpen(false)}>
          {help}
        </HelpPanel>
      ) : null}
    </div>
  );
};
