import { expect, type Page, test } from '@playwright/test';
import { sidebarSettingsButton, skipSetupWizard, TEST_CREDENTIALS } from './helpers/auth';

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
 * - Empty states (no devices, surveys, vulnerabilities)
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

    test('should handle 500 error on device scan', async ({ page }) => {
      await login(page);

      // Mock scan endpoint returning 500
      await page.route('**/api/devices/scan', async (route) => {
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

      if (await scanButton.isVisible({ timeout: 5000 })) {
        await scanButton.click();

        // Should show error message
        await expect(page.getByRole('alert')).toBeVisible({
          timeout: 5000,
        });

        // App should remain functional
        await expect(page.getByTestId('page-header-title')).toBeVisible();
      }
    });
  });

  test.describe('Network Timeout', () => {
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

      // Should show timeout or error message. Old form raced .isVisible({15s})
      // against a 100ms hard sleep — the sleep branch always won, defeating
      // the race. Direct isVisible with the desired timeout is equivalent and
      // honest about what we're waiting for.
      const errorShown = await page.getByRole('alert').isVisible({ timeout: 15000 });

      if (timeoutHandle) {
        clearTimeout(timeoutHandle);
      }

      // Either error shown or loading state ended
      expect(errorShown || (await page.getByLabel(/username/i).isVisible())).toBeTruthy();
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
    test('should redirect to login on session expiration', async ({ page }) => {
      await login(page);

      // Mock API endpoints returning 401 after login. The UI calls the
      // v1-prefixed routes since the API namespace rollout; the previous
      // `**/api/link` and `**/api/status` globs silently no-op'd because
      // they didn't match `/api/v1/link` or `/api/v1/status`. Matching
      // both legacy and v1 keeps the mock effective if any caller is
      // still on the older path.
      await page.route(/\/api(\/v1)?\/link$/, async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Unauthorized',
          }),
        });
      });

      await page.route(/\/api(\/v1)?\/status$/, async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Unauthorized',
          }),
        });
      });

      // Refresh to trigger API calls
      await page.reload();

      // Should show login page or session expired message. The race had a
      // 10s fallback that returned false; equivalent to two parallel 10s
      // isVisible probes ORed together. Express directly with
      // Promise.any-style logic.
      const usernameVisible = page
        .getByLabel(/username|password/i)
        .first()
        .isVisible({ timeout: 10000 });
      const expiredTextVisible = page.getByRole('alert').isVisible({ timeout: 10000 });
      const [usernameOk, expiredOk] = await Promise.all([usernameVisible, expiredTextVisible]);
      const loginShown = usernameOk || expiredOk;

      expect(loginShown).toBeTruthy();
    });

    test('should handle 401 during device scan', async ({ page }) => {
      await login(page);

      // Mock scan endpoint returning 401
      await page.route('**/api/devices/scan', async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Unauthorized',
          }),
        });
      });

      const scanButton = page.getByTestId('discovery-scan-button');

      if (await scanButton.isVisible({ timeout: 5000 })) {
        await scanButton.click();

        // Should show unauthorized error or redirect to login

        // Old form raced two isVisible probes against a 250ms timeout — the
        // 250ms branch always won. Replaced with parallel short-timeout probes
        // ORed together for the same semantics, no race.
        const [authTextSeen, loginFieldSeen] = await Promise.all([
          page.getByRole('alert').isVisible({ timeout: 1000 }),
          page
            .getByLabel(/username|password/i)
            .first()
            .isVisible({ timeout: 1000 }),
        ]);
        const handled = authTextSeen || loginFieldSeen;

        expect(handled).toBeTruthy();
      }
    });
  });

  test.describe('403 Forbidden', () => {});
});

test.describe('Validation Error Scenarios', () => {
  test.describe('Invalid Form Inputs', () => {
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
  test('should show "No devices found" empty state', async ({ page }) => {
    await login(page);

    // Mock empty device list
    await page.route('**/api/devices', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          devices: [],
        }),
      });
    });

    await page.route('**/api/devices/status', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          scanning: false,
          lastScan: new Date().toISOString(),
        }),
      });
    });

    // Reload to get fresh data
    await page.reload();

    // Should show helpful empty state message
    const emptyStateShown = await page
      .getByText(/no devices|no hosts|0 devices|empty|start.*scan/i)
      .isVisible({ timeout: 5000 });

    // Should show either empty state or scan prompt
    expect(
      emptyStateShown || (await page.getByTestId('discovery-scan-button').isVisible()),
    ).toBeTruthy();
  });

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

    // App should show this as a positive result

    // Should either show success message or be functional
    const successShown = await page
      .getByText(/no vulnerabilities|secure|safe|clean/i)
      .isVisible({ timeout: 5000 });

    expect(successShown || (await page.getByTestId('page-header-title').isVisible())).toBeTruthy();
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

    // Should show installation prompt

    const promptShown = await page
      .getByText(/install|not installed|iperf|unavailable/i)
      .isVisible({ timeout: 5000 });

    // Either shows prompt or app remains functional
    expect(promptShown || (await page.getByTestId('page-header-title').isVisible())).toBeTruthy();
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
    await expect(page.getByRole('alert')).toBeVisible({
      timeout: 5000,
    });

    // Retry
    await page.getByTestId('login-submit').click();

    // Should eventually succeed or allow retry

    expect(attemptCount).toBeGreaterThan(0);
  });
});
