import { expect, test } from '@playwright/test';

/**
 * MFA enrolment reaches the server from the browser (#2725).
 *
 * The defect was client-side: `mutate()` read the whole `/api/v1/auth/` prefix
 * as "pre-session" and sent no `X-CSRF-Token`, but the server exempts only the
 * login half — so `totp/setup` answered `403 CSRF token required` and MFA could
 * not be enabled from the shipped UI at all. Nothing in the Go suite saw it:
 * those tests drove `GetAuthenticatedHandler()`, which omits CSRF entirely.
 * Only a real browser against the real chain closes that gap, which is what
 * this spec is for.
 *
 * Scope, deliberately: this drives `totp/setup` and stops. `setup` persists the
 * candidate secret with `totp_enabled = 0` (`handlers_mfa.go`) — it is `verify`
 * that turns the second factor on. Going further would enrol the shared `admin`
 * account: `playwright.config.ts` is `fullyParallel` with four CI workers, and
 * five other specs drive a real admin login, so a second factor — even one
 * disabled again at the end of this test — would make them fail for the length
 * of the window. Enrolling a throwaway user instead is not available either:
 * `POST /api/v1/users` is licence-gated and answers 402 on the Free daemon the
 * harness runs. The rest of the flow — setup → verify → sign in with the code —
 * is covered server-side by `TestTOTPEnrollmentFlow`, which now runs through
 * `Handler()`, the chain production serves, CSRF included.
 */
test.describe('MFA enrolment', () => {
  test('starts TOTP setup from the security page', async ({ page }) => {
    const setup = page.waitForResponse(
      (candidate) =>
        candidate.url().includes('/api/v1/auth/totp/setup') &&
        candidate.request().method() === 'POST',
    );

    await page.goto('/security');
    await page.getByTestId('mfa-setup-totp').click();

    const response = await setup;
    expect(
      response.status(),
      'POST /auth/totp/setup was refused — the enrolment mutation left the browser with no CSRF token (#2725)',
    ).toBe(200);

    // The card can only show the QR and the code field once the call succeeds,
    // which is the operator-visible half of the same assertion.
    const { secret } = (await response.json()) as { secret: string };
    expect(secret).not.toHaveLength(0);
    await expect(page.getByTestId('mfa-totp-code')).toBeVisible({ timeout: 10_000 });
  });
});
