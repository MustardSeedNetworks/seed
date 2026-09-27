import { mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, type FullConfig } from '@playwright/test';

import { AUTH_STORAGE_STATE } from './helpers/auth';
import { signInAndPersist } from './helpers/sign-in';

// ESM equivalent of __dirname; Playwright executes this file as ESM so
// the CommonJS globals are not available.
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 * One real login at suite start; every spec shares the resulting
 * storageState (cookies + localStorage) via use.storageState in
 * playwright.config.ts. This collapses ~436 login attempts down to
 * exactly 1 — the per-IP login limiter
 * (internal/api/ratelimit.go: defaultMaxAttempts = 5 / 15 min) is no
 * longer a suite-wide cliff. auth.spec.ts and auth-complete.spec.ts
 * opt back into a clean unauthenticated context with
 * `test.use({ storageState: { cookies: [], origins: [] } })` so they
 * still exercise the real form.
 *
 * In CI the seed binary starts fresh each run, so there is no admin user
 * until signInAndPersist completes the first-run setup; see
 * helpers/sign-in.ts for why it all happens in a single browser context.
 */
async function globalSetup(config: FullConfig): Promise<void> {
  const [project] = config.projects;
  if (project === undefined) {
    throw new Error('global-setup: no Playwright project configured');
  }
  const baseURL = project.use.baseURL ?? process.env.E2E_BASE_URL ?? 'http://localhost:5173';
  const outPath = resolve(__dirname, '..', AUTH_STORAGE_STATE);

  await mkdir(dirname(outPath), { recursive: true });

  const browser = await chromium.launch();
  try {
    const context = await browser.newContext({
      baseURL,
      ignoreHTTPSErrors: true,
    });
    try {
      await signInAndPersist(context, outPath);
    } finally {
      await context.close();
    }
  } finally {
    await browser.close();
  }
}

export default globalSetup;
