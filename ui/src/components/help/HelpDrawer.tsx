/**
 * HelpDrawer Component
 *
 * Right-side help drawer for The Seed. Replaces the old ImprovedHelpModal,
 * which rendered no body content (every section had `content: null`). This
 * version is data-driven: section metadata + typed content live in
 * `helpDrawerContent.tsx` and are rendered generically by `HelpSectionBody`.
 *
 * Architecture mirrors niac's HelpDrawer (drawer chrome + section nav +
 * search filter) and Seed's SettingsDrawer chrome (overlay/backdrop,
 * useFocusTrap, animate-slide-in), using Seed theme tokens + the `help`
 * i18n namespace throughout.
 *
 * Accessibility:
 * - role="dialog" + aria-modal + aria-labelledby on an id'd <h2>
 * - useFocusTrap provides ESC-close, Tab trapping, and focus restore
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import type React from 'react';
import { type ReactElement, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useFocusTrap } from '../../hooks/useFocusTrap';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import { Search, X } from '../ui/icons';
import { HelpSectionBody } from './HelpSectionBody';
import { helpSections, sectionSearchText } from './helpDrawerContent';

interface HelpDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  /** Application version from backend, shown in the header. */
  version?: string;
}

export function HelpDrawer({ isOpen, onClose, version }: HelpDrawerProps): ReactElement | null {
  const { t } = useTranslation('help');
  const [activeSection, setActiveSection] = useState<string>('about');
  const [searchQuery, setSearchQuery] = useState('');

  // Focus trap: ESC to close, Tab cycling, and focus restore on close.
  const drawerRef = useFocusTrap<HTMLDivElement>({
    isActive: isOpen,
    onEscape: onClose,
  });

  const query = searchQuery.trim().toLowerCase();

  // Filter the TOC: a section matches on its translated title, its id, or any
  // of its rendered body text / keywords.
  const filteredSections = useMemo(() => {
    if (!query) {
      return helpSections;
    }
    return helpSections.filter((section) => {
      const title = t(section.titleKey).toLowerCase();
      return (
        title.includes(query) ||
        section.id.toLowerCase().includes(query) ||
        sectionSearchText(section).includes(query)
      );
    });
  }, [query, t]);

  // Keep the active section valid as the filter narrows the list.
  const currentSection =
    helpSections.find((s) => s.id === activeSection) ?? filteredSections[0] ?? helpSections[0];

  if (!isOpen) {
    return null;
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-scrim/50 backdrop-blur-sm z-40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Drawer */}
      <div
        ref={drawerRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-drawer-title"
        data-testid="help-drawer"
        className="fixed right-0 top-0 h-full w-full sm:w-lg lg:w-2xl bg-surface-raised border-l border-surface-border z-50 flex flex-col shadow-xl animate-slide-in"
      >
        {/* Header */}
        <div
          className={cn(
            layout.flex.between,
            'pad sm:pad-lg border-b border-surface-border bg-surface-raised shrink-0',
          )}
        >
          <div className="stack-xs">
            <h2 id="help-drawer-title" className="heading-3">
              {t('modal.title')}
            </h2>
            {version ? <p className="caption">v{version}</p> : null}
          </div>
          <button
            type="button"
            onClick={onClose}
            data-testid="help-drawer-close"
            className={cn(
              button.size.md,
              radius.md,
              'hover:bg-surface-hover active:bg-surface-hover text-text-muted touch-manipulation focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-raised',
            )}
            aria-label={t('modal.closeHelp')}
          >
            <X className={iconTokens.size.lg} aria-hidden="true" />
          </button>
        </div>

        {/* Body: sidebar (search + TOC) + content pane */}
        <div className="flex flex-1 overflow-hidden">
          {/* Sidebar / table of contents */}
          <aside className="w-64 border-r border-surface-border overflow-y-auto shrink-0 hidden sm:block">
            <div className={cn(spacing.pad.sm, 'border-b border-surface-border')}>
              <div className="relative">
                <Search
                  className={cn(
                    'absolute left-3 top-1/2 -translate-y-1/2',
                    iconTokens.size.sm,
                    'text-text-muted',
                  )}
                  aria-hidden="true"
                />
                <input
                  type="text"
                  placeholder={t('modal.searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    setSearchQuery(e.target.value)
                  }
                  className={cn(
                    'w-full pl-9',
                    spacing.chip.lg,
                    'body-small',
                    radius.default,
                    'border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-brand-primary',
                  )}
                />
              </div>
            </div>

            <nav className={cn(spacing.pad.xs, 'stack-xs')} aria-label={t('modal.contents')}>
              <p className={cn('section-title', spacing.chip.lg)}>{t('modal.contents')}</p>
              {filteredSections.length === 0 ? (
                <p className={cn('caption', spacing.chip.lg)}>{t('modal.searchPlaceholder')}</p>
              ) : (
                filteredSections.map((section) => (
                  <button
                    type="button"
                    key={section.id}
                    onClick={(): void => setActiveSection(section.id)}
                    className={cn(
                      'w-full flex items-center',
                      spacing.gap.default,
                      spacing.tab,
                      radius.default,
                      'body-small transition-colors text-left',
                      currentSection.id === section.id
                        ? 'bg-brand-primary/10 text-brand-primary font-medium'
                        : 'text-text-secondary hover:bg-surface-hover hover:text-text-primary',
                    )}
                    aria-current={currentSection.id === section.id ? 'true' : undefined}
                  >
                    {section.icon}
                    <span>{t(section.titleKey)}</span>
                  </button>
                ))
              )}
            </nav>
          </aside>

          {/* Content pane */}
          <main
            // Reset scroll on reopen: keying the pane to the open state + active
            // section remounts it, so it always starts at the top.
            key={`${currentSection.id}`}
            className={cn('flex-1 overflow-y-auto', spacing.pad.lg)}
            data-testid="help-drawer-content"
          >
            <h3 className="heading-3 mb-content">{t(currentSection.titleKey)}</h3>
            <HelpSectionBody blocks={currentSection.blocks} />
          </main>
        </div>
      </div>
    </>
  );
}

export default HelpDrawer;
