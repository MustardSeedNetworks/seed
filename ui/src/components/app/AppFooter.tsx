/**
 * AppFooter - product info, contact, website, legal links, and copyright.
 *
 * Rendered at the bottom of the App dashboard. Pure presentation - takes
 * the current app version and the translator from the parent.
 */

import { useTranslation } from 'react-i18next';
import { cn, radius, spacing } from '../../styles/theme';

interface AppFooterProps {
  appVersion: string;
}

export function AppFooter({ appVersion }: AppFooterProps): JSX.Element {
  const { t } = useTranslation('common');

  return (
    <footer
      class={cn(
        spacing.margin.top.section,
        radius.lg,
        'border border-surface-border bg-surface-raised',
        spacing.pad.lg,
      )}
    >
      <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {/* Product Info */}
        <div>
          <h3 class="heading-4 text-text-primary mb-2">{t('app.title')}</h3>
          <p class="body-small text-text-muted mb-1">
            {t('footer.byCompany', 'by Mustard Seed Networks')}
          </p>
          <p class="caption text-text-muted">
            {t('footer.version', 'Version')} {appVersion}
          </p>
        </div>

        {/* Contact */}
        <div>
          <h4 class="body-small font-medium text-text-primary mb-2">
            {t('footer.contact', 'Contact')}
          </h4>
          <div class="space-y-1">
            <a
              href="mailto:support@mustardseednetworks.com"
              class="body-small text-brand-primary hover:underline block"
            >
              support@mustardseednetworks.com
            </a>
            <a
              href="tel:+17194403079"
              class="body-small text-text-muted hover:text-text-primary block"
            >
              719.440.3079
            </a>
          </div>
        </div>

        {/* Website */}
        <div>
          <h4 class="body-small font-medium text-text-primary mb-2">
            {t('footer.website', 'Website')}
          </h4>
          <a
            href="https://www.mustardseednetworks.com"
            target="_blank"
            rel="noopener noreferrer"
            class="body-small text-brand-primary hover:underline"
          >
            www.mustardseednetworks.com
          </a>
        </div>

        {/* Legal */}
        <div>
          <h4 class="body-small font-medium text-text-primary mb-2">
            {t('footer.legal', 'Legal')}
          </h4>
          <div class="flex flex-wrap gap-x-3 gap-y-1">
            <a href="/terms" class="body-small text-text-muted hover:text-brand-primary">
              {t('footer.tos', 'Terms of Service')}
            </a>
            <a href="/privacy" class="body-small text-text-muted hover:text-brand-primary">
              {t('footer.privacy', 'Privacy')}
            </a>
            <a href="/license" class="body-small text-text-muted hover:text-brand-primary">
              {t('footer.license', 'License')}
            </a>
          </div>
        </div>
      </div>

      {/* Copyright */}
      <div class="mt-6 pt-4 border-t border-surface-border text-center">
        <p class="caption text-text-muted">
          &copy; {new Date().getFullYear()}{' '}
          {t('footer.copyright', 'Mustard Seed Networks. All rights reserved.')}
        </p>
      </div>
    </footer>
  );
}
