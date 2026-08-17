import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin';
import { playwright } from '@vitest/browser-playwright';
import { defineConfig } from 'vitest/config';

const currentDir = dirname(fileURLToPath(import.meta.url));

/**
 * Story files that do not yet pass the interaction/a11y run, excluded by path
 * so every other story is gated by default.
 *
 * This is deliberately a short deny-list rather than the `tags: { include:
 * ['test-ready'] }` allow-list it replaces. Under the allow-list exactly one
 * of 88 story files carried the tag — the synthetic StorybookGate story — so
 * the job verified that the harness works while covering no real component,
 * and every story added since was ungated by default. A deny-list inverts
 * that: new stories are gated the moment they are written, and anything
 * skipped is visible here rather than invisible by omission.
 *
 * Shrink this list; do not grow it. Tracked in seed#1916.
 */
const NOT_YET_PASSING = [
  // Render crash inside the section components under the Storybook runtime.
  '**/src/components/settings/sections/DiscoverySettings.stories.tsx',
  '**/src/components/settings/sections/DnsSettings.stories.tsx',
  '**/src/components/settings/sections/HealthChecksSettings.stories.tsx',
  '**/src/components/settings/sections/PerformanceSettings.stories.tsx',
  // TypeError raised before the story mounts.
  '**/src/components/ui/SpeedGauge.stories.tsx',
];

export default defineConfig({
  test: {
    projects: [
      {
        extends: true,
        plugins: [
          storybookTest({
            configDir: resolve(currentDir, '.storybook'),
          }),
        ],
        test: {
          name: 'storybook',
          exclude: NOT_YET_PASSING,
          browser: {
            enabled: true,
            headless: true,
            provider: playwright({}),
            instances: [{ browser: 'chromium' }],
          },
        },
      },
    ],
  },
});
