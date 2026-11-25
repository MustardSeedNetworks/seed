import { useTranslation } from 'react-i18next';
import type { CardStatus } from '../shared/Card';

/** The statuses that have a translated name: every card status except `info`. */
export type LabelledStatus = Exclude<CardStatus, 'info'>;

/**
 * The translated accessible name of a status indicator ("Status: success"),
 * which the shared Card and StatusIndicator take as `statusLabel` / `label`.
 */
export function useStatusLabel(): (status: LabelledStatus) => string {
  const { t } = useTranslation();
  return (status) => {
    switch (status) {
      case 'success':
        return t('accessibility.status.success');
      case 'warning':
        return t('accessibility.status.warning');
      case 'error':
        return t('accessibility.status.error');
      case 'unknown':
        return t('accessibility.status.unknown');
      case 'loading':
        return t('accessibility.status.loading');
    }
  };
}
