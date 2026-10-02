/**
 * Alert primitive — ported from niac UI kit (Phase B).
 */
import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
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
    containerClass: 'border-status-success/30 bg-status-success/10 text-status-success-strong',
    iconClass: 'text-status-success',
  },
  error: {
    icon: AlertCircle,
    containerClass: 'border-status-error/30 bg-status-error/10 text-status-error-strong',
    iconClass: 'text-status-error',
  },
  warning: {
    icon: AlertTriangle,
    containerClass: 'border-status-warning/30 bg-status-warning/10 text-status-warning-strong',
    iconClass: 'text-status-warning',
  },
  info: {
    icon: Info,
    containerClass: 'border-status-info/30 bg-status-info/10 text-status-info-strong',
    iconClass: 'text-status-info',
  },
};

export const Alert: FC<AlertProps> = ({ status, children, onDismiss, className = '' }) => {
  const { t } = useTranslation();
  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div
      className={`flex items-center gap-compact rounded-lg border pad-sm ${config.containerClass} ${className}`}
      role="alert"
    >
      <Icon className={`${iconSizes.md} flex-shrink-0 ${config.iconClass}`} />
      <span className="flex-1">{children}</span>
      {onDismiss ? (
        <button
          type="button"
          onClick={onDismiss}
          className="ml-auto text-current hover:opacity-70 transition-opacity"
          aria-label={t('accessibility.dismissAlert')}
        >
          <X className={iconSizes.md} />
        </button>
      ) : null}
    </div>
  );
};
