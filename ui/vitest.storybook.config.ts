import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin';
import { playwright } from '@vitest/browser-playwright';
import { defineConfig } from 'vitest/config';
import type { Reporter } from 'vitest/node';

const currentDir = dirname(fileURLToPath(import.meta.url));

// Outside a terminal the default reporter prints a file only once it has
// finished, so a browser run that stalls mid-file (#3283) ends at the job
// timeout without naming the story it was in. Announcing each test as it
// starts means a hung CI log names the test that started and never finished.
// A terminal already shows the running tests live.
const announceTestStart: Reporter = {
  onTestCaseReady(testCase) {
    process.stdout.write(`started ${testCase.module.relativeModuleId} > ${testCase.fullName}\n`);
  },
};

export default defineConfig({
  test: {
    reporters: process.env.CI ? ['default', announceTestStart] : ['default'],
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
