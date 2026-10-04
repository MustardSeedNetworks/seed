import { expect, test } from '@playwright/test';

import { TEST_CREDENTIALS } from './helpers/auth';
import { totpCode, wrongTotpCode } from './helpers/totp';

/**
 * TOTP enrolment and removal from the security page (#2725, #2731).
 *
 * This enrols a real second factor on the shared `admin` account, so it runs in
 * the `serial-admin` project, which every browser project depends on: it
 * finishes before the parallel suite starts. Run alongside it, any spec that
 * signs in for real would meet a TOTP prompt for the length of the window.
 * Enrolling a throwaway user instead is not available: `POST /api/v1/users` is
 * licence-gated and answers 402 on the Free daemon the harness runs.
 *
 * #2725: `mutate()` sent no CSRF token to the enrolment routes, so `totp/setup`
 * answered 403 and MFA could not be enabled from the UI at all.
 * #2731: nothing called `totp/disable`, and a wrong factor there (or on
 * `totp/verify`) answered 401, which the client takes for an expired session:
 * it refreshed, replayed the request and signed the user out over a typo.
 */
test.describe('MFA TOTP', () => {
  test('enrols, refuses a wrong code in place, turns off and signs in with the password alone', async ({
    page,
    browser,
  }) => {
    const setup = page.waitForResponse(
      (candidate) =>
        candidate.url().includes('/api/v1/auth/totp/setup') &&
        candidate.request().method() === 'POST',
    );
    await page.goto('/security');
    await page.getByTestId('mfa-setup-totp').click();

    const setupResponse = await setup;
    expect(
      setupResponse.status(),
      'POST /auth/totp/setup was refused — the enrolment mutation left the browser with no CSRF token (#2725)',
    ).toBe(200);
    const { secret } = (await setupResponse.json()) as { secret: string };
    expect(secret).not.toHaveLength(0);

    await page.getByTestId('mfa-totp-code').fill(totpCode(secret));
    await page.getByTestId('mfa-verify-totp').click();
    await expect(page.getByTestId('mfa-disable-totp')).toBeVisible();

    await page.getByTestId('mfa-disable-totp').click();
    await page.getByTestId('mfa-disable-password').fill(TEST_CREDENTIALS.password);
    await page.getByTestId('mfa-disable-code').fill(wrongTotpCode(secret));
    await page.getByTestId('mfa-disable-confirm').click();

    // The refusal is reported on the card and the session survives it.
    await expect(page.getByTestId('mfa-error')).toHaveText(
      'Wrong or expired code. Try the next one.',
    );
    await expect(page.getByTestId('mfa-disable-form')).toBeVisible();
    await expect(page.getByTestId('login-submit')).toHaveCount(0);

    await page.getByTestId('mfa-disable-code').fill(totpCode(secret));
    await page.getByTestId('mfa-disable-confirm').click();
    await expect(page.getByTestId('mfa-setup-totp')).toBeVisible();
    await expect(page.getByTestId('mfa-disable-totp')).toHaveCount(0);

    // A fresh sign-in asks for the password and nothing else.
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    try {
      const fresh = await context.newPage();
      await fresh.goto('/');
      await fresh.getByLabel(/username/i).fill(TEST_CREDENTIALS.username);
      await fresh.getByLabel(/password/i).fill(TEST_CREDENTIALS.password);
      await fresh.getByTestId('login-submit').click();
      await expect(fresh.getByTestId('page-header-title')).toBeVisible();
      await expect(fresh.getByTestId('mfa-code-input')).toHaveCount(0);
    } finally {
      await context.close();
    }
  });
});
