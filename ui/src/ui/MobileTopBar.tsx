/**
 * The phone top bar: the product mark and name, and the button that opens the
 * rail as a drawer. Below the `lg` breakpoint it stands in for the rail.
 */
import { Menu, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { SeedLogo } from '../components/app/SeedLogo';
import { Tooltip } from '../components/ui/Tooltip';
import { iconSizes } from '../constants/sizes';
import { ProductTitle } from './ProductTitle';

interface MobileTopBarProps {
  mobileOpen: boolean;
  toggleMobile: () => void;
  deviceName?: string;
}

export const MobileTopBar: FC<MobileTopBarProps> = ({ mobileOpen, toggleMobile, deviceName }) => {
  const { t } = useTranslation();
  return (
    <header className="lg:hidden fixed top-0 left-0 right-0 z-50 flex-between px-4 py-row-lg bg-surface-raised/95 backdrop-blur-xl border-b border-surface-border">
      <div className="flex min-w-0 items-center gap-compact">
        <SeedLogo badge badgeClassName="h-8 w-8" glyphClassName={iconSizes.md} />
        <ProductTitle
          bar="topbar"
          deviceName={deviceName}
          className="font-display font-bold text-text-primary"
        />
      </div>
      <Tooltip text={mobileOpen ? t('accessibility.closeMenu') : t('accessibility.openMenu')}>
        <button
          type="button"
          onClick={toggleMobile}
          data-testid="mobile-menu-toggle"
          className="pad-xs rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
          aria-label={mobileOpen ? t('accessibility.closeMenu') : t('accessibility.openMenu')}
        >
          {mobileOpen ? <X className={iconSizes.lg} /> : <Menu className={iconSizes.lg} />}
        </button>
      </Tooltip>
    </header>
  );
};
