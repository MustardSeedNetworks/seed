import { ReactNode } from 'react';

export type Status = 'success' | 'warning' | 'error' | 'unknown' | 'loading';

interface CardProps {
  title: string;
  status: Status;
  children: ReactNode;
  className?: string;
  onClick?: () => void;
}

const statusConfig: Record<Status, { icon: string; color: string; bgColor: string }> = {
  success: {
    icon: '●',
    color: 'text-status-success',
    bgColor: 'bg-status-success/10',
  },
  warning: {
    icon: '●',
    color: 'text-status-warning',
    bgColor: 'bg-status-warning/10',
  },
  error: {
    icon: '●',
    color: 'text-status-error',
    bgColor: 'bg-status-error/10',
  },
  unknown: {
    icon: '○',
    color: 'text-text-muted',
    bgColor: 'bg-surface-hover',
  },
  loading: {
    icon: '◐',
    color: 'text-status-info animate-spin',
    bgColor: 'bg-status-info/10',
  },
};

export function Card({ title, status, children, className = '', onClick }: CardProps) {
  const config = statusConfig[status];

  return (
    <div
      className={`rounded-lg border border-surface-border bg-surface-raised p-3 sm:p-4 transition-all hover:border-brand-primary/50 touch-manipulation ${
        onClick ? 'cursor-pointer active:scale-[0.98]' : ''
      } ${className}`}
      onClick={onClick}
    >
      <div className="flex items-center justify-between">
        <h3 className="font-semibold text-text-primary text-base sm:text-lg">{title}</h3>
        <span className={`text-lg ${config.color}`}>{config.icon}</span>
      </div>
      <div className="mt-2 sm:mt-3">{children}</div>
    </div>
  );
}

interface CardValueProps {
  label?: string;
  value: string | number;
  unit?: string;
  size?: 'sm' | 'md' | 'lg';
  status?: Status;
}

export function CardValue({ label, value, unit, size = 'md', status }: CardValueProps) {
  const sizeClasses = {
    sm: 'text-sm',
    md: 'text-base font-medium',
    lg: 'text-lg font-semibold',
  };

  return (
    <div>
      {label && <p className="text-xs text-text-muted mb-1">{label}</p>}
      <p className={`${sizeClasses[size]} ${status ? statusConfig[status].color : 'text-text-primary'}`}>
        {value}
        {unit && <span className="text-sm font-normal text-text-muted ml-1">{unit}</span>}
      </p>
    </div>
  );
}

interface CardRowProps {
  label: string;
  value: string | number;
  status?: Status;
}

export function CardRow({ label, value, status }: CardRowProps) {
  return (
    <div className="flex items-center justify-between gap-2 py-1">
      <span className="text-sm text-text-muted shrink-0">{label}</span>
      <span
        className={`text-sm font-medium text-right truncate ${status ? statusConfig[status].color : 'text-text-primary'}`}
        title={String(value)}
      >
        {value}
      </span>
    </div>
  );
}

interface CardDividerProps {
  className?: string;
}

export function CardDivider({ className = '' }: CardDividerProps) {
  return <hr className={`border-surface-border my-3 ${className}`} />;
}
