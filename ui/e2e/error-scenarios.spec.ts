import { expect, type Page, test } from '@playwright/test';
import { loginAndAwaitDashboard, skipSetupWizard, TEST_CREDENTIALS } from './helpers/auth';

/**
 * Comprehensive Error Scenario E2E Tests
 *
 * Tests error handling and graceful degradation across all features:
 *
 * API Error Scenarios:
 * - 500 Internal Server Error
 * - Network timeouts
 * - 404 Not Found
 * - 401 Unauthorized (session expired)
 * - 403 Forbidden
 *
 * Validation Error Scenarios:
 * - Invalid form inputs
 * - File upload errors
 *
 * WebSocket Error Scenarios:
 * - Connection failures
 * - Invalid messages
 *
 * Resource Error Scenarios:
 * - Empty states (no devices, vulnerabilities)
 * - Backend service unavailable
 *
 * Edge Cases:
 * - Large data sets
 * - Rapid successive actions
 * - Concurrent operations
 *
 * Ensures robust error handling that doesn't crash the app and provides
 * clear user feedback with recovery options.
 */

/**
 * Helper: Login to the application
 */
async function login(page: Page): Promise<void> {
  await skipSetupWizard(page);
  await page.goto('/');
  await expect(page.getByTestId('page-header-title')).toBeVisible({
    timeout: 10000,
  });
}

test.describe('API Error Scenarios', () => {
  test.describe('500 Internal Server Error', () => {
    test.describe('on login form', () => {
      test.use({ storageState: { cookies: [], origins: [] } });
      test('should handle 500 error on login', async ({ page }) => {
        await page.goto('/');

        // Mock login endpoint returning 500. Match both /api/auth/login (legacy)
        // and /api/v1/auth/login (current — UI calls this since the v1 prefix
        // rollout). The previous glob `**/api/auth/login` would not intercept
        // the v1 form, so the mock was silently inert.
        await page.route(/\/api(\/v1)?\/auth\/login$/, async (route) => {
          await route.fulfill({
            status: 500,
            contentType: 'application/json',
            body: JSON.stringify({
              error: 'Internal server error',
            }),
          });
        });

        await page.getByLabel(/username/i).fill(TEST_CREDENTIALS.username);
        await page.getByLabel(/password/i).fill(TEST_CREDENTIALS.password);
        await page.getByTestId('login-submit').click();

        // Should show user-friendly error message
        await expect(page.getByRole('alert')).toBeVisible({
          timeout: 5000,
        });

        // Should not crash the app
        await expect(page.getByLabel(/username/i)).toBeVisible();
      });
    });

    // #2394's fix landed (useDeviceScan / NetworkDiscoveryCard now surface a
    // failed scan via discovery-scan-error) but this test still cannot reach
    // it live: NetworkDiscoveryCard's only mount point is /path, gated by
    // <RequireFeature feature="path_analysis"> (Pro tier -- see
    // internal/api/server_routes.go's setupPathRoutes), and the E2E suite
    // runs unlicensed (Free). Confirmed against a live daemon: navigating to
    // /path renders "Path Analysis is a Pro-tier feature... seed license
    // trial", not the card, so discovery-scan-button never exists. Same
    // blocker reports-page.spec.ts documents for export_csv_json (Starter+)
    // and defers as "needs a licensed fixture, out of scope". The card's own
    // backend routes (/api/v1/security/devices*) carry no `feature:` gate --
    // network discovery is a Free-tier-capable feature trapped behind a
    // Pro-only page, a placement accident from the router refactor
    // (68130b0b) rather than a deliberate gate; filed separately.
    //
    // The route glob and testid scoping below are the real fix, kept for
    // when a licensed E2E fixture exists. Real coverage of the fix lives in
    // ui/src/hooks/useDeviceScan.test.ts (the error state) and
    // ui/src/components/cards/NetworkDiscoveryCard.test.tsx (the banner).
    test.fixme('should handle 500 error on device scan', async ({ page }) => {
      await login(page);
      await page.goto('/path');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });

      // Mock scan endpoint returning 500. The real endpoint is
      // /api/v1/security/devices/scan -- the previous glob
      // (**/api/devices/scan) does not match that path (no `/v1/security`
      // segment), so the route was never intercepted and the request always
      // reached the real backend.
      await page.route('**/api/v1/security/devices/scan', async (route) => {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Failed to start scan',
          }),
        });
      });

      // Try to trigger a scan
      const scanButton = page.getByTestId('discovery-scan-button');
      await expect(scanButton).toBeVisible({ timeout: 5000 });
      await scanButton.click();

      // Should show the scan's own error banner, scoped by testid -- an
      // unscoped getByRole('alert') also matches AppShell's always-present
      // CapabilityWarnings banner in a CI container, which is why this
      // assertion could not fail before (#2394).
      const scanError = page.getByTestId('discovery-scan-error');
      await expect(scanError).toBeVisible({ timeout: 5000 });
      await expect(scanError).toHaveAttribute('role', 'alert');

      // App should remain functional
      await expect(page.getByTestId('page-header-title')).toBeVisible();

      // And the operator can retry without reloading.
      await expect(scanButton).toBeEnabled();
    });
  });

  test.describe('Network Timeout', () => {
    test.describe('on login form', () => {
      test.use({ storageState: { cookies: [], origins: [] } });
      test('should handle API timeout gracefully', async ({ page }) => {
        await page.goto('/');

        // Mock login endpoint that never responds (simulates timeout).
        // RegExp matches both /api/auth/login and /api/v1/auth/login.
        let timeoutHandle: NodeJS.Timeout;
        await page.route(/\/api(\/v1)?\/auth\/login$/, async (route) => {
          // Delay indefinitely to trigger timeout
          await new Promise((resolve) => {
            timeoutHandle = setTimeout(resolve, 60000); // 1 minute
          });
          await route.abort('timedout');
        });

        await page.getByLabel(/username/i).fill(TEST_CREDENTIALS.username);
        await page.getByLabel(/password/i).fill(TEST_CREDENTIALS.password);
        await page.getByTestId('login-submit').click();

        // The login request carries AbortSignal.timeout(15s), so a server that
        // accepts the connection and never answers must surface an error rather
        // than leaving the form disabled. Waiting slightly longer than the
        // client's own deadline, since that deadline is the thing under test.
        //
        // The previous assertion was `errorShown || usernameField.isVisible()`,
        // which passed whichever happened: the username field is always visible
        // on a page that never navigates, so it could not fail.
        // Matched on the timeout copy specifically, not on any role="alert":
        // a stale session-expired banner is also an alert and is already on
        // the page, so a bare getByRole('alert') passes at once and the
        // assertion proves nothing about the deadline.
        await expect(page.getByText(/did not respond/i)).toBeVisible({ timeout: 20000 });

        if (timeoutHandle) {
          clearTimeout(timeoutHandle);
        }

        // And the operator can retry: a submit button still disabled is the
        // stuck state this exists to catch. By now the deadline has fired, so
        // this needs no long wait of its own.
        await expect(page.getByTestId('login-submit')).toBeEnabled({ timeout: 5000 });
      });
    });

    test('should handle device scan timeout', async ({ page }) => {
      await login(page);

      // Mock scan endpoint with timeout
      await page.route('**/api/devices/scan', async (route) => {
        await new Promise((resolve) => setTimeout(resolve, 10000));
        await route.abort('timedout');
      });

      const scanButton = page.getByTestId('discovery-scan-button');

      if (await scanButton.isVisible({ timeout: 5000 })) {
        await scanButton.click();

        // Should handle timeout gracefully (loading ends or error shown)

        // App should remain functional
        await expect(page.getByTestId('page-header-title')).toBeVisible();
      }
    });
  });

  test.describe('404 Not Found', () => {
    test('should handle missing device', async ({ page }) => {
      await login(page);

      // Mock device list
      await page.route('**/api/devices', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            devices: [
              {
                ip: '192.168.1.100',
                mac: '00:11:22:33:44:55',
                hostname: 'test-device',
              },
            ],
          }),
        });
      });

      // Mock device detail returning 404
      await page.route('**/api/devices/192.168.1.100', async (route) => {
        await route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Device not found',
          }),
        });
      });

      // App should handle missing device gracefully
      await expect(page.getByTestId('page-header-title')).toBeVisible();
    });
  });

  test.describe('401 Unauthorized (Session Expired)', () => {
    // Isolated from the suite-wide shared storageState: this test forces a
    // real session-expiry logout, which blacklists the shared token
    // server-side and poisons `login()` for every later shared-state test in
    // the worker (observed: iPerf3/speedtest/retry-login tests failed with
    // ERR_CONNECTION_REFUSED and a re-run's `page-header-title` wait timed
    // out, downstream of this test's own POST /api/v1/auth/logout going
    // through against the shared token). Same isolation
    // auth-complete.spec.ts's own 401 test uses.
    test.use({ storageState: { cookies: [], origins: [] } });

    // See the 500 case above (#2394) for why this is fixme: the discovery
    // card only mounts on /path, gated Pro-tier, and the suite runs
    // unlicensed. Kept for when a licensed E2E fixture exists.
    test.fixme('should handle 401 during device scan', async ({ page }) => {
      await page.goto('/');
      await loginAndAwaitDashboard(page);
      await page.goto('/path');
      await expect(page.getByTestId('page-header-title')).toBeVisible({
        timeout: 10000,
      });

      // Mock scan endpoint returning 401. The real endpoint is
      // /api/v1/security/devices/scan (see the 500 case above for why the
      // old glob never matched it). The api client reacts to a 401 by
      // attempting POST /api/v1/auth/refresh before giving up -- that must
      // also fail, or the refresh would hit the real backend with a still-
      // valid session cookie, succeed, and silently retry the scan instead
      // of expiring the session. Same idiom as auth-complete.spec.ts's
      // "should handle 401 unauthorized response gracefully".
      await page.route('**/api/v1/security/devices/scan', async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Unauthorized' }),
        });
      });
      await page.route('**/api/v1/auth/refresh', async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Unauthorized' }),
        });
      });

      const scanButton = page.getByTestId('discovery-scan-button');
      await expect(scanButton).toBeVisible({ timeout: 5000 });
      await scanButton.click();

      // A session-expiry 401 is routed into the app's existing re-auth
      // flow (api client -> onSessionExpired -> logout -> LoginForm),
      // rather than the scan's own generic error banner: the operator
      // needs to log back in, not retry the scan. Scoped to the login
      // form's own alert -- an unscoped getByRole('alert') also matches
      // AppShell's always-present CapabilityWarnings banner in a CI
      // container, which is why this could not fail before (#2394).
      await expect(page.getByTestId('login-title')).toBeVisible({
        timeout: 10000,
      });
      await expect(page.getByTestId('login-error')).toBeVisible();
    });
  });

  test.describe('403 Forbidden', () => {});
});

test.describe('Validation Error Scenarios', () => {
  test.describe('Invalid Form Inputs', () => {
    test.use({ storageState: { cookies: [], origins: [] } });
    test('should validate empty login credentials', async ({ page }) => {
      await page.goto('/');

      // Try to submit empty form
      const loginButton = page.getByTestId('login-submit');
      await loginButton.click();

      // Should show validation error or button be disabled
      const hasError = await page.getByRole('alert').isVisible({ timeout: 3000 });
      const buttonDisabled = await loginButton.isDisabled();

      expect(hasError || buttonDisabled).toBeTruthy();
    });
  });
});

test.describe('Resource Error Scenarios - Empty States', () => {
  test('should show "No vulnerabilities found" success state', async ({ page }) => {
    await login(page);

    // Mock vulnerability scan with no findings
    await page.route('**/api/vulnerabilities/results', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          vulnerabilities: [],
          scannedAt: new Date().toISOString(),
        }),
      });
    });

    await page.route('**/api/vulnerabilities/status', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          scanning: false,
          lastScan: new Date().toISOString(),
        }),
      });
    });

    // The /no vulnerabilities|secure|safe|clean/i success-text regex
    // was i18n-fragile and its OR-with-page-header-title meant the
    // assertion never failed independently. Reduced to the survivor:
    // the page renders. Real empty-state coverage needs a stable
    // testid on the vulnerabilities EmptyState component.
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 5000,
    });
  });
});

test.describe('Backend Service Unavailable', () => {
  test('should handle iPerf3 not installed', async ({ page }) => {
    await login(page);

    // Mock iPerf info showing not installed
    await page.route('**/api/iperf/info', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          available: false,
          version: '',
          error: 'iperf3 not found in PATH',
        }),
      });
    });

    // The /install|not installed|iperf|unavailable/i regex was
    // partially DNT-safe ("iperf" is a product name) but the rest
    // of the alternation was i18n-fragile, and the OR-with-page-
    // header-title meant the assertion never failed independently.
    // Reduced to the survivor: the page renders. Real iperf3-missing
    // coverage needs a stable testid on the iperf-availability
    // banner / error component.
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 5000,
    });
  });

  test('should handle speedtest.net unavailable', async ({ page }) => {
    await login(page);

    // Mock speedtest endpoint returning service unavailable
    await page.route('**/api/speedtest', async (route) => {
      await route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Unable to connect to speedtest.net servers',
        }),
      });
    });

    await page.route('**/api/speedtest/status', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          running: false,
        }),
      });
    });

    // App should handle this gracefully
    await expect(page.getByTestId('page-header-title')).toBeVisible();
  });
});

test.describe('Error Recovery Mechanisms', () => {
  test.use({ storageState: { cookies: [], origins: [] } });
  test('should allow retry after failed login', async ({ page }) => {
    await page.goto('/');

    let attemptCount = 0;

    // First attempt fails, second succeeds.
    // RegExp matches both /api/auth/login and /api/v1/auth/login.
    await page.route(/\/api(\/v1)?\/auth\/login$/, async (route) => {
      attemptCount++;
      if (attemptCount === 1) {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Server error' }),
        });
      } else {
        await route.continue();
      }
    });

    // First attempt
    await page.getByLabel(/username/i).fill(TEST_CREDENTIALS.username);
    await page.getByLabel(/password/i).fill(TEST_CREDENTIALS.password);
    await page.getByTestId('login-submit').click();

    // Should show error
    await expect(page.getByTestId('login-error')).toBeVisible({
      timeout: 5000,
    });

    // Retry. The point of the test is that the second attempt actually reaches
    // the server and succeeds, so wait for the app shell rather than for the
    // click to return.
    await page.getByTestId('login-submit').click();

    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 15000 });
    // The login error specifically, not any alert: the dashboard legitimately
    // renders a role=alert capability-degradation banner (#2315) on a host
    // whose platform cannot do everything, which a CI container never can.
    await expect(page.getByTestId('login-error')).toBeHidden();

    // Exactly two: the failure and the retry. `toBeGreaterThan(0)` was true
    // after the first attempt alone, so it passed whether or not the retry
    // happened at all -- which is the thing being tested.
    expect(attemptCount).toBe(2);
  });
});
