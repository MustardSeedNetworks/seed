/**
 * ImprovedHelpModal Component (~681 lines)
 *
 * Purpose: Comprehensive application help modal providing user guidance across multiple topics.
 * Features tabbed navigation, search functionality, and rich content for all major features.
 *
 * Key Features:
 * - Multi-section help: About, Network Discovery, WiFi, Cable/Link, Performance, etc.
 * - Search functionality: Filter help content by keyword
 * - Icon-based navigation: Visual section selector with icons
 * - Rich content: Markdown-like formatting for help text
 * - Modal overlay: Centered help dialog with close button
 * - Responsive design: Adapts to different screen sizes
 * - Keyboard support: ESC key closes modal
 * - Scrollable sections: Long help content in scrollable containers
 *
 * Usage:
 * ```typescript
 * <ImprovedHelpModal isOpen={showHelp} onClose={() => setShowHelp(false)} />
 * ```
 *
 * Dependencies: Icons, theme utilities, useState for tab/search state management
 * State: activeSection (current tab), searchQuery (help search text)
 */

import type React from 'react';
import { type ReactNode, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, layout, modal, radius, spacing } from '../../styles/theme';
import {
  Activity,
  AlertTriangle,
  BookOpen,
  Cable,
  Heart,
  HeartPulse,
  Info,
  LayoutDashboard,
  Lightbulb,
  Monitor,
  Network,
  Search,
  Server,
  Shield,
  Signal,
  SlidersHorizontal,
  Wifi,
  Zap,
} from '../ui/icons';

interface HelpModalProps {
  isOpen: boolean;
  onClose: () => void;
  /** Application version from backend */
  version?: string;
}

interface HelpSection {
  id: string;
  title: string;
  icon: ReactNode;
  content: ReactNode;
}

/**
 * ImprovedHelpModal Component
 * Renders a modal dialog with tabbed help content and search functionality
 */
export function ImprovedHelpModal({ isOpen, onClose }: HelpModalProps): React.JSX.Element | null {
  const { t } = useTranslation('help');
  // Track which help section is currently active
  const [activeSection, setActiveSection] = useState<string>('about');
  // Track search query for filtering help content
  const [searchQuery, setSearchQuery] = useState('');

  if (!isOpen) {
    return null;
  }

  const sections: HelpSection[] = [
    {
      id: 'about',
      title: t('sections.about'),
      icon: <Info className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'getting-started',
      title: t('sections.gettingStarted'),
      icon: <LayoutDashboard className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'link',
      title: t('sections.link'),
      icon: <Activity className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'cable',
      title: t('sections.cable'),
      icon: <Cable className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'wifi',
      title: t('sections.wifi'),
      icon: <Wifi className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'network',
      title: t('sections.network'),
      icon: <Network className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'gateway',
      title: t('sections.gateway'),
      icon: <Server className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'dns',
      title: t('sections.dns'),
      icon: <Search className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'performance',
      title: t('sections.performance'),
      icon: <Zap className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'discovery',
      title: t('sections.discovery'),
      icon: <Search className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'healthChecks',
      title: t('sections.healthChecks'),
      icon: <Heart className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'security',
      title: t('sections.security'),
      icon: <Shield className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'troubleshooting',
      title: t('sections.troubleshooting'),
      icon: <AlertTriangle className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'profiles',
      title: t('sections.profiles'),
      icon: <SlidersHorizontal className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'wifiSurvey',
      title: t('sections.wifiSurvey'),
      icon: <Signal className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'rtspChecks',
      title: t('sections.rtspChecks'),
      icon: <Monitor className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'dicomChecks',
      title: t('sections.dicomChecks'),
      icon: <HeartPulse className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'howTo',
      title: t('sections.howTo'),
      icon: <Lightbulb className={iconTokens.size.sm} />,
      content: null,
    },
    {
      id: 'glossary',
      title: t('sections.glossary'),
      icon: <BookOpen className={iconTokens.size.sm} />,
      content: null,
    },
  ];

  const filteredSections = sections.filter(
    (section) =>
      section.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      section.id.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const currentSection = sections.find((s) => s.id === activeSection);

  return (
    <div className={modal.overlay}>
      {/* Backdrop */}
      <div className={modal.backdrop} onClick={onClose} aria-hidden="true" />
      {/* Modal */}
      <div
        className={cn(
          'relative',
          modal.content,
          modal.size.xl,
          radius.lg,
          'flex flex-col overflow-hidden',
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-modal-title"
      >
        {/* Header */}
        <div
          className={cn(
            layout.flex.between,
            spacing.pad.default,
            'border-b border-surface-border shrink-0',
          )}
        >
          <h2 id="help-modal-title" className="heading-3">
            {t('modal.title')}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className={cn(
              spacing.pad.xs,
              'text-text-muted hover:text-text-primary transition-colors',
              radius.default,
              'hover:bg-surface-hover',
            )}
            aria-label={t('modal.closeHelp')}
          >
            <svg
              className={iconTokens.size.md}
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Content area with sidebar */}
        <div className="flex flex-1 overflow-hidden">
          {/* Sidebar / TOC */}
          <aside className="w-64 border-r border-surface-border bg-surface-base overflow-y-auto shrink-0">
            {/* Search */}
            <div className={cn(spacing.pad.sm, 'border-b border-surface-border')}>
              <div className="relative">
                <Search
                  className={cn(
                    'absolute left-3 top-1/2 -translate-y-1/2',
                    iconTokens.size.sm,
                    'text-text-muted',
                  )}
                />
                <input
                  type="text"
                  placeholder={t('modal.searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
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

            {/* Table of Contents */}
            <nav className={cn(spacing.pad.xs, 'stack-xs')}>
              <p className={cn('caption', spacing.chip.lg, 'uppercase tracking-wider')}>
                {t('modal.contents')}
              </p>
              {filteredSections.map((section) => (
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
                    activeSection === section.id
                      ? 'bg-brand-primary/10 text-brand-primary font-medium'
                      : 'text-text-secondary hover:bg-surface-hover hover:text-text-primary',
                  )}
                >
                  {section.icon}
                  <span>{section.title}</span>
                </button>
              ))}
            </nav>
          </aside>

          {/* Main content */}
          <main className={cn('flex-1 overflow-y-auto', spacing.pad.lg)}>
            {currentSection && <div>{currentSection.content}</div>}
          </main>
        </div>
      </div>
    </div>
  );
}
