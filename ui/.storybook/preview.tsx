/**
 * Storybook Preview Configuration
 *
 * Global decorators that wrap all stories with required providers:
 * - I18nextProvider: For translation support (useTranslation)
 * - ProfileProvider: For profile and settings context (useSettings, useProfileContext)
 * - RoleProvider: For the active stem role (useRole)
 * - LicenseProvider: For tier gating (useLicense)
 * - Theme wrapper: For dark/light mode support
 *
 * This ensures all components work correctly in isolation without
 * needing to manually wrap each story with providers.
 */

import type { DecoratorFunction, StoryContext } from '@storybook/csf';
import type { Preview, ReactRenderer } from '@storybook/react-vite';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { setupWorker } from 'msw/browser';
import { type JSX, type ReactNode, Suspense, useEffect, useState } from 'react';
import { I18nextProvider } from 'react-i18next';
import { LicenseProvider } from '../src/contexts/LicenseContext';
import { ProfileProvider } from '../src/contexts/profileContext';
import { RoleProvider } from '../src/contexts/RoleContext';
import i18n from '../src/i18n';
import { handlers } from './msw/handlers';
import '../src/index.css';

// msw intercepts the provider bootstrap calls every story makes. Storybook has
// no daemon behind it, so without this they hit the dev server's HTML fallback
// and React Query parses `<!doctype` as JSON -- 60 error lines in a passing
// run, in which a real regression is indistinguishable from the noise (#2203).
const worker = setupWorker(...handlers);
const workerReady = worker.start({
  // Unhandled requests pass through, unannounced. `warn` was tried first and
  // is worse than the disease: it logs a line per unhandled call, which took
  // the run from 60 noisy lines to 397. Endpoints beyond the provider
  // bootstrap still reach the dev server's HTML fallback exactly as they did
  // before this change; silencing those needs their own handlers, and the
  // console-error gate cannot land until it does (#2203).
  onUnhandledRequest: 'bypass',
  quiet: true,
  serviceWorker: { url: './mockServiceWorker.js' },
});

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

function StoryProviders({ children, profile }: { children: ReactNode; profile: boolean }) {
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  );

  useEffect(
    () => (): void => {
      queryClient.clear();
    },
    [queryClient],
  );

  const content = <Suspense fallback={<LoadingFallback />}>{children}</Suspense>;
  return (
    <I18nextProvider i18n={i18n}>
      {profile ? (
        <QueryClientProvider client={queryClient}>
          <ProfileProvider>
            <RoleProvider>
              <LicenseProvider>{content}</LicenseProvider>
            </RoleProvider>
          </ProfileProvider>
        </QueryClientProvider>
      ) : (
        content
      )}
    </I18nextProvider>
  );
}

const preview: Preview = {
  // Awaited before any story renders, so the first story is mocked like every
  // other one rather than racing the worker's registration.
  loaders: [async (): Promise<void> => void (await workerReady)],
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
    // Wave 5 / seed-W5-3: the axe-core a11y addon gates the build.
    //
    // 'error' since #2099. It sat on 'todo' because flipping it surfaced ten
    // WCAG violations across nine components; those are now fixed, the suite
    // is 443/443 with the gate on, and a new violation fails CI rather than
    // being catalogued and ignored. Do not lower this to catalogue a
    // regression -- fix the story or the component.
    a11y: {
      test: 'error',
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
        <StoryProviders profile={context.parameters.seedProfile !== false}>
          <ThemeWrapper dark={isDark}>
            <div className="p-4">
              <Story />
            </div>
          </ThemeWrapper>
        </StoryProviders>
      );
    }) as DecoratorFunction<ReactRenderer>,
  ],
};

export default preview;
