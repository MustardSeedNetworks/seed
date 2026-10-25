import { expect, test } from '@playwright/test';

import { loginAndAwaitDashboard } from './helpers/auth';

/**
 * Passkey enrolment and username-free sign-in against the real daemon
 * (D-SEED-114, #3644), with Chromium's virtual authenticator standing in for a
 * platform authenticator.
 *
 * WebAuthn refuses an IP address as a relying party, and the harness serves
 * 127.0.0.1, so this spec signs in at https://localhost:<port>: the daemon's
 * relying party when no public origin is configured. It enrols a passkey on the
 * shared `admin` account, which a password sign-in ignores, so it runs in the
 * `serial-admin` project with the other account-changing specs. WebKit has no
 * virtual authenticator; this flow runs on Chromium only.
 */
test.use({ storageState: { cookies: [], origins: [] } });

test('enrols a passkey on the security page and signs in with it, naming no account', async ({
  page,
  baseURL,
}) => {
  const origin = new URL(baseURL ?? '');
  origin.hostname = 'localhost';

  const cdp = await page.context().newCDPSession(page);
  await cdp.send('WebAuthn.enable');
  await cdp.send('WebAuthn.addVirtualAuthenticator', {
    options: {
      protocol: 'ctap2',
      transport: 'internal',
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });

  await page.goto(`${origin.origin}/`);
  await loginAndAwaitDashboard(page);

  const finish = page.waitForResponse(
    (candidate) =>
      candidate.url().includes('/api/v1/auth/webauthn/register/finish') &&
      candidate.request().method() === 'POST',
  );
  await page.goto(`${origin.origin}/security`);
  await page.getByTestId('mfa-add-passkey').click();
  expect((await finish).status(), 'register/finish refused the new passkey').toBe(200);
  await expect(page.getByTestId('mfa-error')).toHaveCount(0);

  // Sign out by dropping the session, then sign in with the passkey alone.
  await page.context().clearCookies();
  const outcome = await page.evaluate(async () => {
    const post = (path: string, body: unknown): Promise<Response> =>
      fetch(path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
    const begin = await post('/api/v1/auth/webauthn/login/begin', {});
    const { publicKey } = (await begin.json()) as {
      publicKey: PublicKeyCredentialRequestOptionsJSON;
    };
    const credential = (await navigator.credentials.get({
      publicKey: PublicKeyCredential.parseRequestOptionsFromJSON(publicKey),
    })) as PublicKeyCredential;
    const assertion = credential.toJSON();
    const signedIn = await post('/api/v1/auth/webauthn/login/finish', assertion);
    const { token } = (await signedIn.json()) as { token?: string };
    // The ceremony cookie was spent: the same assertion cannot sign in twice.
    const replay = await post('/api/v1/auth/webauthn/login/finish', assertion);
    return { begin: begin.status, finish: signedIn.status, token, replay: replay.status };
  });

  expect(outcome.begin).toBe(200);
  expect(outcome.finish, 'login/finish refused the enrolled passkey').toBe(200);
  expect(outcome.token).toBeTruthy();
  expect(outcome.replay).toBe(400);
});
