/**
 * Storybook Preview Configuration
 *
 * Global decorators that wrap all stories with required providers:
 * - I18nextProvider: For translation support (useTranslation)
 * - ProfileProvider: For profile and settings context (useSettings, useProfileContext)
 * - Theme wrapper: For dark/light mode support
 *
 * This ensures all components work correctly in isolation without
 * needing to manually wrap each story with providers.
 */

import type { DecoratorFunction, StoryContext } from '@storybook/csf';
import type { Preview, ReactRenderer } from '@storybook/react-vite';
import { type JSX, type ReactNode, Suspense, useEffect } from 'react';
import { I18nextProvider } from 'react-i18next';
import { ProfileProvider } from '../src/contexts/ProfileContext';
import i18n from '../src/i18n';
import '../src/index.css';

/**
 * Theme wrapper that applies dark/light class to document.
 * Storybook background parameter controls the visual background,
 * while this applies the Tailwind theme class.
 */
function ThemeWrapper({
  children,
  dark = true,
}: {
  children: ReactNode;
  dark?: boolean;
}): JSX.Element {
  useEffect((): (() => void) => {
    if (dark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
    return (): void => {
      document.documentElement.classList.remove('dark');
    };
  }, [dark]);
  return <>{children}</>;
}

/**
 * Loading fallback for Suspense during i18n initialization
 */
function LoadingFallback(): JSX.Element {
  return <div className="flex items-center justify-center p-4 text-text-muted">Loading...</div>;
}

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    backgrounds: {
      default: 'dark',
      values: [
        { name: 'dark', value: 'var(--color-surface-base, #0f172a)' },
        { name: 'light', value: 'var(--color-surface-base-light, #f8fafc)' },
      ],
    },
    layout: 'centered',
    // Wave 5 / seed-W5-3: activate the axe-core a11y addon. 'todo'
    // surfaces WCAG 2.1 AA violations in the Storybook UI as a
    // catalog without failing CI yet. After a baseline pass through
    // existing stories, raise to 'error' to gate new violations.
    a11y: {
      test: 'todo',
      config: {
        rules: [
          // color-contrast can flake on Tailwind tokens whose runtime
          // values depend on theme-context. Leave enabled but bias
          // toward report-only for now.
        ],
      },
    },
  },
  decorators: [
    // Global decorator: wraps all stories with providers. The Story
    // argument is the rendered story; rendering `<Story />` (capital
    // S — JSX component, not the lowercase HTML element placeholder
    // a prior version had) is what makes the wrapper actually show
    // the story content.
    ((Story: () => ReactNode, context: StoryContext<ReactRenderer>): JSX.Element => {
      const isDark =
        context.globals.backgrounds?.value !== 'var(--color-surface-base-light, #f8fafc)';

      return (
        <I18nextProvider i18n={i18n}>
          <Suspense fallback={<LoadingFallback />}>
            <ProfileProvider>
              <ThemeWrapper dark={isDark}>
                <div className="p-4">
                  <Story />
                </div>
              </ThemeWrapper>
            </ProfileProvider>
          </Suspense>
        </I18nextProvider>
      );
    }) as DecoratorFunction<ReactRenderer>,
  ],
};

export default preview;
