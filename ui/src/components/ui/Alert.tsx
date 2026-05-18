/**
 * Alert primitive — ported from niac UI kit (Phase B).
 */
import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { iconSizes } from '../../constants/sizes';

export type AlertStatus = 'success' | 'error' | 'warning' | 'info';

export interface AlertProps {
  status: AlertStatus;
  children: ReactNode;
  onDismiss?: () => void;
  className?: string;
}

const statusConfig: Record<
  AlertStatus,
  {
    icon: typeof AlertCircle;
    containerClass: string;
    iconClass: string;
  }
> = {
  success: {
    icon: CheckCircle,
    containerClass: 'border-status-success/30 bg-status-success/10 text-status-success',
    iconClass: 'text-status-success',
  },
  error: {
    icon: AlertCircle,
    containerClass: 'border-status-error/30 bg-status-error/10 text-status-error',
    iconClass: 'text-status-error',
  },
  warning: {
    icon: AlertTriangle,
    containerClass: 'border-status-warning/30 bg-status-warning/10 text-status-warning',
    iconClass: 'text-status-warning',
  },
  info: {
    icon: Info,
    containerClass: 'border-status-info/30 bg-status-info/10 text-status-info',
    iconClass: 'text-status-info',
  },
};

export const Alert: FC<AlertProps> = ({ status, children, onDismiss, className = '' }) => {
  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div
      class={`flex items-center gap-2 rounded-lg border p-3 ${config.containerClass} ${className}`}
      role="alert"
    >
      <Icon class={`${iconSizes.md} flex-shrink-0 ${config.iconClass}`} />
      <span class="flex-1">{children}</span>
      {onDismiss ? (
        <button
          type="button"
          onClick={onDismiss}
          class="ml-auto text-current hover:opacity-70 transition-opacity"
          aria-label="Dismiss alert"
        >
          <X class={iconSizes.md} />
        </button>
      ) : null}
    </div>
  );
};
