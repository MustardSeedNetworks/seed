/**
 * The phone-width gate's setup-command (MustardSeedNetworks/.github
 * phone-width.yml): sign the fresh E2E daemon in and write BASE_URL's session
 * to STORAGE_STATE, so the gate visits each route signed in rather than
 * judging the login screen.
 *
 * The browser comes from the gate's own Playwright, not ui/'s: the gate
 * installs a build for its pinned driver only, and puts that driver on
 * NODE_PATH, which only require() consults. (helpers/auth.ts still loads ui/'s
 * @playwright/test for its `expect`; nothing here launches from it.) Run under
 * plain `node`, so imports carry `.ts`.
 */
import { createRequire } from 'node:module';
import process from 'node:process';

import { signInAndPersist } from './helpers/sign-in.ts';

const { chromium } = createRequire(import.meta.url)('playwright') as typeof import('playwright');
const { BASE_URL: baseURL, STORAGE_STATE: outPath } = process.env;
if (!baseURL || !outPath) {
  process.stderr.write('BASE_URL and STORAGE_STATE are required\n');
  process.exit(2);
}

const browser = await chromium.launch();
try {
  const context = await browser.newContext({ baseURL, ignoreHTTPSErrors: true });
  await signInAndPersist(context, outPath);
} finally {
  await browser.close();
}
