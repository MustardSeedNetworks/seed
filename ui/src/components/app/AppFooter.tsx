/**
 * AppFooter — the quiet line under every page: copyright, how to reach us,
 * and the legal links.
 *
 * It used to be a four-column panel 192px tall, carrying the product name,
 * the company and the version — all three of which the rail already shows a
 * few hundred pixels to the left — and it rendered on all twelve routes, not
 * only the dashboard its docstring claimed. On /wifi that made a contact
 * block taller than the page's own content (UI-SEED-20, seed#2709). What is
 * left is what only the footer can say.
 *
 * The section headings survive as group labels rather than visual headings:
 * one line reads as one line, and a screen reader still hears which links are
 * contact and which are legal.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, spacing } from '../../styles/theme';

const LINK = 'caption text-text-muted hover:text-text-primary link-target';

export function AppFooter(): JSX.Element {
  const { t } = useTranslation('common');
  const company = t('footer.company');

  return (
    <footer
      data-testid="app-footer"
      className={cn(
        spacing.margin.top.content,
        spacing.padding.top.section,
        'border-t border-surface-border',
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
        <p className="caption text-text-muted">
          {t('footer.copyright', { company, year: String(new Date().getFullYear()) })}
        </p>

        <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
          <nav aria-label={t('footer.contact')} className="flex flex-wrap items-center gap-x-4">
            <a
              href="mailto:support@mustardseednetworks.com"
              className={cn(LINK, 'text-brand-primary hover:underline break-words')}
            >
              support@mustardseednetworks.com
            </a>
            <a href="tel:+17194403079" className={LINK}>
              719.440.3079
            </a>
            <a
              href="https://www.mustardseednetworks.com"
              target="_blank"
              rel="noopener noreferrer"
              aria-label={t('footer.website')}
              className={LINK}
            >
              www.mustardseednetworks.com
            </a>
          </nav>

          <nav aria-label={t('footer.legal')} className="flex flex-wrap items-center gap-x-3">
            <a href="/terms" className={LINK}>
              {t('footer.tos')}
            </a>
            <a href="/privacy" className={LINK}>
              {t('footer.privacy')}
            </a>
            <a href="/license" className={LINK}>
              {t('footer.license')}
            </a>
          </nav>
        </div>
      </div>
    </footer>
  );
}
