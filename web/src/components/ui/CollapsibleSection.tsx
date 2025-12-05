import { useState, ReactNode } from 'react';
import { Status } from './Card';

interface CollapsibleSectionProps {
  title: ReactNode;
  defaultOpen?: boolean;
  children: ReactNode;
  /** Number of items to display in header, e.g., "Server Results (2)" */
  count?: number;
  /** Status indicator to show next to title */
  status?: Status;
  /** Use compact styling for inside cards */
  variant?: 'default' | 'compact';
}

const statusColors: Record<Status, string> = {
  success: 'bg-status-success',
  warning: 'bg-status-warning',
  error: 'bg-status-error',
  loading: 'bg-text-muted animate-pulse',
  unknown: 'bg-text-muted',
};

export function CollapsibleSection({
  title,
  defaultOpen = false,
  children,
  count,
  status,
  variant = 'default',
}: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  const isCompact = variant === 'compact';

  return (
    <section className={isCompact ? '' : 'border border-surface-border rounded-lg overflow-hidden'}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`w-full flex items-center justify-between transition-colors ${
          isCompact
            ? 'py-1.5 hover:bg-surface-hover/50 rounded'
            : 'p-3 bg-surface-base hover:bg-surface-hover'
        }`}
      >
        <div className="flex items-center gap-2">
          <svg
            className={`w-3 h-3 text-text-muted transition-transform duration-200 ${isOpen ? 'rotate-90' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          {status && (
            <span className={`w-2 h-2 rounded-full ${statusColors[status]}`} />
          )}
          <span className={`font-medium text-text-primary ${isCompact ? 'text-xs' : 'text-sm'}`}>
            {title}
            {count !== undefined && (
              <span className="text-text-muted ml-1">({count})</span>
            )}
          </span>
        </div>
      </button>
      {isOpen && (
        <div className={
          isCompact
            ? 'pl-5 pb-2 space-y-1'
            : 'p-3 border-t border-surface-border bg-surface-raised space-y-3'
        }>
          {children}
        </div>
      )}
    </section>
  );
}
