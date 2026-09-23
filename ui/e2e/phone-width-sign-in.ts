/**
 * The phone-width gate's setup-command (MustardSeedNetworks/.github
 * phone-width.yml): sign the fresh E2E daemon in and write BASE_URL's session
 * to STORAGE_STATE, so the gate visits each route signed in rather than
 * judging the login screen.
 *
 * The browser comes from the gate's own Playwright, not ui/'s: the gate
 * installs a build for its pinned driver only and passes that driver's
 * node_modules as NODE_PATH. require() is anchored THERE, because resolving
 * from this file finds ui/node_modules/playwright first — a different version
 * whose browser the runner does not have. (helpers/auth.ts still loads ui/'s
 * @playwright/test for its `expect`; nothing here launches from it.) Run under
 * plain `node`, so imports carry `.ts`.
 */
import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import process from 'node:process';

import { signInAndPersist } from './helpers/sign-in.ts';

const { BASE_URL: baseURL, STORAGE_STATE: outPath, NODE_PATH: gateModules } = process.env;
if (!baseURL || !outPath || !gateModules) {
  process.stderr.write('BASE_URL, STORAGE_STATE and NODE_PATH are required\n');
  process.exit(2);
}
const { chromium } = createRequire(resolve(gateModules, '..', 'package.json'))(
  'playwright',
) as typeof import('playwright');

const browser = await chromium.launch();
try {
  const context = await browser.newContext({ baseURL, ignoreHTTPSErrors: true });
  await signInAndPersist(context, outPath);
} finally {
  await browser.close();
}
