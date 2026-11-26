import { useTranslation } from 'react-i18next';
import type { CardStatus } from '../shared/Card';

export type { CardStatus };

/**
 * The accessible name the shared Card, StatusIndicator and CollapsibleSection
 * need for a status: the shared set takes translated strings, so the call site
 * supplies them. One literal key per status, so the locale gate sees each one.
 */
export function useCardStatusLabel(): (status: CardStatus) => string {
  const { t } = useTranslation('common');
  return (status) => {
    switch (status) {
      case 'success':
        return t('accessibility.status.success');
      case 'warning':
        return t('accessibility.status.warning');
      case 'error':
        return t('accessibility.status.error');
      case 'info':
        return t('accessibility.status.info');
      case 'loading':
        return t('accessibility.status.loading');
      case 'unknown':
        return t('accessibility.status.unknown');
    }
  };
}
