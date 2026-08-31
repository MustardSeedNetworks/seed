import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import type { StorybookConfig } from '@storybook/react-vite';
import type { UserConfig } from 'vite';

const currentDir: string = dirname(fileURLToPath(import.meta.url));

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
  addons: [
    '@chromatic-com/storybook',
    '@storybook/addon-vitest',
    '@storybook/addon-a11y',
    '@storybook/addon-docs',
    '@storybook/addon-onboarding',
  ],
  // Serves .storybook/public, which holds msw's generated service worker.
  //
  // It lived in ui/public and Vite copies that directory verbatim into
  // internal/api/ui, so the mock worker was being embedded in the seed binary
  // and served to operators. CodeQL caught it: the worker has a postMessage
  // handler with no origin check, which is fine for a test double and not
  // something to ship. Storybook is the only thing that should ever see it.
  staticDirs: ['./public'],
  framework: '@storybook/react-vite',
  viteFinal: (viteConfig: UserConfig): UserConfig => {
    // OVERRIDE — don't merge aliases. The main vite.config.ts declares
    // aliases as an array of `{find: RegExp, replacement: string}` entries
    // (Vite's "advanced" form), but Storybook's rolldown-based builder
    // expects the simpler object form. Spread-merging the two produces a
    // malformed config that crashes with "StringExpected on
    // BindingViteAliasPluginAlias.replacement". The object form below
    // satisfies Storybook; runtime Vite still uses the regex form.
    return {
      ...viteConfig,
      resolve: {
        ...viteConfig.resolve,
        alias: {
          '@': resolve(currentDir, '../src'),
          '@locales': resolve(currentDir, '../../internal/i18n/locales'),
        },
      },
    };
  },
};
export default config;
