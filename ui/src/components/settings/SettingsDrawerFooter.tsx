/**
 * SettingsDrawer footer: Logs (debug) viewer, Export download link,
 * About blurb. Pulled out so SettingsDrawer.tsx can shrink.
 */

import { useTranslation } from 'react-i18next';
import { button, cn, icon as iconTokens, radius, spacing } from '../../styles/theme';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

interface SettingsDrawerFooterProps {
  version: string;
  fetchLogPreview: () => Promise<void> | void;
  logLoading: boolean;
  logError: string | null;
  logPreview: string[];
}

export function SettingsDrawerFooter({
  version,
  fetchLogPreview,
  logLoading,
  logError,
  logPreview,
}: SettingsDrawerFooterProps): JSX.Element {
  const { t } = useTranslation('settings');

  return (
    <>
      {/* Logs (debug) */}
      <section class={cn(spacing.padding.top.section, 'border-t border-surface-border')}>
        <div class="flex items-start justify-between">
          <div>
            <h3 class="body-small font-medium text-text-muted">{t('logs.title')}</h3>
            <p class="caption text-text-muted">{t('logs.description')}</p>
          </div>
          <button
            type="button"
            onClick={fetchLogPreview}
            class={cn(
              'caption',
              spacing.chip.sm,
              'border border-surface-border',
              radius.md,
              'text-text-muted hover:text-text-primary hover:border-text-muted transition-colors',
            )}
          >
            {logLoading ? t('logs.loading') : t('logs.view')}
          </button>
        </div>
        {logError ? (
          <p class={cn('caption text-status-error', spacing.margin.top.inline)}>{logError}</p>
        ) : null}
        {!logError && logPreview.length > 0 ? (
          <pre
            class={cn(
              spacing.margin.top.inline,
              'max-h-48 overflow-y-auto text-2xs leading-5 bg-surface-base border border-surface-border',
              radius.md,
              spacing.chip.lg,
              'text-text-primary whitespace-pre-wrap',
            )}
          >
            {logPreview.join('\n')}
          </pre>
        ) : null}
      </section>

      {/* Export Section */}
      <section class={cn(spacing.padding.top.section, 'border-t border-surface-border')}>
        <h3 class={cn('body-small font-medium text-text-muted', spacing.margin.bottom.heading)}>
          {t('export.title')}
        </h3>
        <a
          href={`${API_BASE}/api/harvest/export`}
          download="seed-export.json"
          class={cn(
            'w-full',
            button.size.md,
            'bg-surface-base border border-surface-border text-text-primary',
            radius.md,
            'font-medium hover:bg-surface-hover transition-colors flex items-center justify-center',
            spacing.gap.compact,
            'touch-manipulation',
          )}
        >
          <svg
            class={iconTokens.size.sm}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
            />
          </svg>
          {t('export.download')}
        </a>
        <p class={cn('caption text-text-muted', spacing.margin.top.inline)}>
          {t('export.description')}
        </p>
      </section>

      {/* About Section */}
      <section class={cn(spacing.padding.top.section, 'border-t border-surface-border')}>
        <h3 class={cn('body-small font-medium text-text-muted', spacing.margin.bottom.inline)}>
          {t('about.title')}
        </h3>
        <p class="caption text-text-muted">
          {t('about.appName')} {version}
          <br />
          {t('about.description')}
        </p>
      </section>
    </>
  );
}
