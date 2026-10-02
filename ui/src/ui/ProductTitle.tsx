import type { FC } from 'react';
import { useTranslation } from 'react-i18next';

interface ProductTitleProps {
  /** The bar's own type treatment for the product name. */
  className: string;
  /** The operator's name for this device (#195); empty or absent shows nothing. */
  deviceName?: string;
  /** Which bar this is, for the E2E: `rail` or `topbar`. */
  bar: 'rail' | 'topbar';
}

/**
 * The product name with the device's own name under it, as both the rail and
 * the phone top bar show it, so several open tabs can be told apart.
 */
export const ProductTitle: FC<ProductTitleProps> = ({ className, deviceName, bar }) => {
  const { t } = useTranslation();
  return (
    <span className="flex min-w-0 flex-col">
      <span className={className}>{t('app.title')}</span>
      {deviceName ? (
        <span data-testid={`${bar}-device-name`} className="caption truncate text-text-muted">
          {deviceName}
        </span>
      ) : null}
    </span>
  );
};
