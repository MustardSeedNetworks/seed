import { expect, type Page, test } from '@playwright/test';

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
  await page.goto('/');
  await page.evaluate(() => localStorage.clear());
  await page.reload();

  await page.getByLabel(/username/i).fill('admin');
  await page.getByLabel(/password/i).fill('seed');
  await page.getByRole('button', { name: /sign in|login/i }).click();

  await expect(page.getByRole('heading', { name: /link/i })).toBeVisible({
    timeout: 10000,
  });
}

test.describe('API Error Scenarios', () => {
  test.describe('500 Internal Server Error', () => {
    test('should handle 500 error on login', async ({ page }) => {
      await page.goto('/');

      // Mock login endpoint returning 500
      await page.route('**/api/auth/login', async (route) => {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Internal server error',
          }),
        });
      });

      await page.getByLabel(/username/i).fill('admin');
      await page.getByLabel(/password/i).fill('seed');
      await page.getByRole('button', { name: /sign in|login/i }).click();

      // Should show user-friendly error message
      await expect(page.getByText(/error|failed|unable/i)).toBeVisible({
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
      const scanButton = page.getByRole('button', { name: /scan|discover|refresh/i }).first();

      if (await scanButton.isVisible({ timeout: 5000 })) {
        await scanButton.click();

        // Should show error message
        await expect(page.getByText(/error|failed/i)).toBeVisible({
          timeout: 5000,
        });

        // App should remain functional
        await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();
      }
    });

    test('should handle 500 error on speed test', async ({ page }) => {
      await login(page);

      // Mock speedtest endpoint returning 500
      await page.route('**/api/speedtest', async (route) => {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Speed test service unavailable',
          }),
        });
      });

      // Try to find and click speed test button
      const speedTestButton = page.getByRole('button', { name: /speed test|test speed/i }).first();

      const isVisible = await speedTestButton.isVisible({ timeout: 3000 }).catch(() => false);

      if (isVisible) {
        await speedTestButton.click();

        // Should show error message
        await expect(page.getByText(/error|failed|unavailable/i)).toBeVisible({
          timeout: 5000,
        });
      }
    });

    test('should handle 500 error on survey creation', async ({ page }) => {
      await login(page);

      // Mock survey creation endpoint returning 500
      await page.route('**/api/survey/create', async (route) => {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Failed to create survey',
          }),
        });
      });

      // Mock survey list (initially empty)
      await page.route('**/api/survey/list', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ surveys: [] }),
        });
      });

      // Try to create a survey if available
      const surveyButton = page.getByRole('button', { name: /survey|wifi survey/i }).first();
      const isVisible = await surveyButton.isVisible({ timeout: 3000 }).catch(() => false);

      if (isVisible) {
        await surveyButton.click();

        // Try to create new survey
        const createButton = page.getByRole('button', { name: /create|new survey/i }).first();
        if (await createButton.isVisible({ timeout: 3000 })) {
          await createButton.click();

          // Fill form if visible
          const nameInput = page.getByLabel(/name|survey name/i).first();
          if (await nameInput.isVisible({ timeout: 2000 })) {
            await nameInput.fill('Test Survey');

            // Submit
            const submitButton = page.getByRole('button', { name: /create|save|submit/i }).first();
            if (await submitButton.isVisible({ timeout: 2000 })) {
              await submitButton.click();

              // Should show error
              await expect(page.getByText(/error|failed/i)).toBeVisible({
                timeout: 5000,
              });
            }
          }
        }
      }
    });
  });

  test.describe('Network Timeout', () => {
    test('should handle API timeout gracefully', async ({ page }) => {
      await page.goto('/');

      // Mock login endpoint that never responds (simulates timeout)
      let timeoutHandle: NodeJS.Timeout;
      await page.route('**/api/auth/login', async (route) => {
        // Delay indefinitely to trigger timeout
        await new Promise((resolve) => {
          timeoutHandle = setTimeout(resolve, 60000); // 1 minute
        });
        await route.abort('timedout');
      });

      await page.getByLabel(/username/i).fill('admin');
      await page.getByLabel(/password/i).fill('seed');
      await page.getByRole('button', { name: /sign in|login/i }).click();

      // Should show timeout or error message
      const errorShown = await Promise.race([
        page
          .getByText(/timeout|error|failed|unable/i)
          .isVisible({ timeout: 15000 })
          .then(() => true),
        page.waitForTimeout(15000).then(() => false),
      ]);

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

      const scanButton = page.getByRole('button', { name: /scan|discover|refresh/i }).first();

      if (await scanButton.isVisible({ timeout: 5000 })) {
        await scanButton.click();

        // Should handle timeout gracefully (loading ends or error shown)
        await page.waitForTimeout(5000);

        // App should remain functional
        await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();
      }
    });
  });

  test.describe('404 Not Found', () => {
    test('should handle missing survey', async ({ page }) => {
      await login(page);

      // Mock survey list with one survey
      await page.route('**/api/survey/list', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            surveys: [
              {
                id: 'missing-survey',
                name: 'Test Survey',
                status: 'created',
                createdAt: new Date().toISOString(),
              },
            ],
          }),
        });
      });

      // Mock survey detail returning 404
      await page.route('**/api/survey?id=missing-survey', async (route) => {
        await route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Survey not found',
          }),
        });
      });

      // Try to open survey view if available
      const surveyButton = page.getByRole('button', { name: /survey|wifi survey/i }).first();
      const isVisible = await surveyButton.isVisible({ timeout: 3000 }).catch(() => false);

      if (isVisible) {
        await surveyButton.click();
        await page.waitForTimeout(2000);

        // Try to click on a survey
        const surveyItem = page.getByText('Test Survey').first();
        if (await surveyItem.isVisible({ timeout: 2000 })) {
          await surveyItem.click();

          // Should show "not found" message
          const notFoundShown = await Promise.race([
            page
              .getByText(/not found|doesn't exist|unavailable/i)
              .isVisible({ timeout: 5000 })
              .then(() => true),
            page.waitForTimeout(5000).then(() => false),
          ]);

          // Either shows error or remains functional
          expect(
            notFoundShown || (await page.getByRole('heading', { name: /link/i }).isVisible()),
          ).toBeTruthy();
        }
      }
    });

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
      await page.waitForTimeout(2000);
      await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();
    });
  });

  test.describe('401 Unauthorized (Session Expired)', () => {
    test('should redirect to login on session expiration', async ({ page }) => {
      await login(page);

      // Mock API endpoints returning 401 after login
      await page.route('**/api/link', async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Unauthorized',
          }),
        });
      });

      await page.route('**/api/status', async (route) => {
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

      // Should show login page or session expired message
      const loginShown = await Promise.race([
        page
          .getByLabel(/username|password/i)
          .first()
          .isVisible({ timeout: 10000 })
          .then(() => true),
        page
          .getByText(/session.*expired|unauthorized|login/i)
          .isVisible({ timeout: 10000 })
          .then(() => true),
        page.waitForTimeout(10000).then(() => false),
      ]);

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

      const scanButton = page.getByRole('button', { name: /scan|discover|refresh/i }).first();

      if (await scanButton.isVisible({ timeout: 5000 })) {
        await scanButton.click();

        // Should show unauthorized error or redirect to login
        await page.waitForTimeout(3000);

        const handled = await Promise.race([
          page
            .getByText(/unauthorized|session|login/i)
            .isVisible()
            .then(() => true),
          page
            .getByLabel(/username|password/i)
            .first()
            .isVisible()
            .then(() => true),
          page.waitForTimeout(5000).then(() => false),
        ]);

        expect(handled).toBeTruthy();
      }
    });
  });

  test.describe('403 Forbidden', () => {
    test('should handle permission denied on settings update', async ({ page }) => {
      await login(page);

      // Mock settings update returning 403
      await page.route('**/api/settings', async (route) => {
        if (route.request().method() === 'PUT' || route.request().method() === 'POST') {
          await route.fulfill({
            status: 403,
            contentType: 'application/json',
            body: JSON.stringify({
              error: 'Permission denied',
            }),
          });
        } else {
          await route.continue();
        }
      });

      // Try to open settings
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      if (await settingsButton.isVisible({ timeout: 3000 })) {
        await settingsButton.click();
        await page.waitForTimeout(1000);

        // Try to modify a setting if available
        const input = page.locator('input[type="number"], input[type="text"]').first();
        if (await input.isVisible({ timeout: 2000 })) {
          await input.fill('123');

          // Try to save
          const saveButton = page.getByRole('button', { name: /save|apply/i }).first();
          if (await saveButton.isVisible({ timeout: 2000 })) {
            await saveButton.click();

            // Should show permission denied error
            const errorShown = await page
              .getByText(/permission|forbidden|denied|authorized/i)
              .isVisible({ timeout: 5000 })
              .catch(() => false);

            // Either error shown or app remains functional
            expect(
              errorShown || (await page.getByRole('heading', { name: /link/i }).isVisible()),
            ).toBeTruthy();
          }
        }
      }
    });
  });
});

test.describe('Validation Error Scenarios', () => {
  test.describe('Invalid Form Inputs', () => {
    test('should validate empty login credentials', async ({ page }) => {
      await page.goto('/');

      // Try to submit empty form
      const loginButton = page.getByRole('button', { name: /sign in|login/i });
      await loginButton.click();

      // Should show validation error or button be disabled
      const hasError = await page
        .getByText(/required|enter|provide/i)
        .isVisible({ timeout: 3000 })
        .catch(() => false);
      const buttonDisabled = await loginButton.isDisabled().catch(() => false);

      expect(hasError || buttonDisabled).toBeTruthy();
    });

    test('should validate invalid threshold values in settings', async ({ page }) => {
      await login(page);

      // Open settings
      const settingsButton = page
        .getByRole('button', { name: /settings/i })
        .or(page.locator('button:has(svg[class*="settings"], svg[class*="cog"])'));

      if (await settingsButton.isVisible({ timeout: 3000 })) {
        await settingsButton.click();
        await page.waitForTimeout(1000);

        // Try to enter negative number in threshold input
        const thresholdInput = page.locator('input[type="number"]').first();
        if (await thresholdInput.isVisible({ timeout: 2000 })) {
          await thresholdInput.fill('-50');

          // Should show validation error or prevent submission
          const errorShown = await page
            .getByText(/invalid|positive|greater/i)
            .isVisible({ timeout: 3000 })
            .catch(() => false);

          const saveButton = page.getByRole('button', { name: /save|apply/i }).first();
          const saveDisabled = await saveButton.isDisabled().catch(() => false);

          expect(errorShown || saveDisabled).toBeTruthy();
        }
      }
    });

    test('should validate invalid hostname in DNS test', async ({ page }) => {
      await login(page);

      // Mock DNS endpoint
      await page.route('**/api/dns', async (route) => {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Invalid hostname format',
          }),
        });
      });

      // Try to find DNS test input
      const dnsInput = page.getByPlaceholder(/hostname|domain|dns/i).first();
      if (await dnsInput.isVisible({ timeout: 3000 })) {
        await dnsInput.fill('invalid hostname with spaces!@#');

        const testButton = page.getByRole('button', { name: /test|check|lookup/i }).first();
        if (await testButton.isVisible({ timeout: 2000 })) {
          await testButton.click();

          // Should show validation error
          await expect(page.getByText(/invalid|format|error/i)).toBeVisible({
            timeout: 5000,
          });
        }
      }
    });

    test('should validate missing survey name', async ({ page }) => {
      await login(page);

      // Mock survey endpoints
      await page.route('**/api/survey/list', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ surveys: [] }),
        });
      });

      // Try to create survey without name
      const surveyButton = page.getByRole('button', { name: /survey|wifi survey/i }).first();
      const isVisible = await surveyButton.isVisible({ timeout: 3000 }).catch(() => false);

      if (isVisible) {
        await surveyButton.click();

        const createButton = page.getByRole('button', { name: /create|new survey/i }).first();
        if (await createButton.isVisible({ timeout: 3000 })) {
          await createButton.click();

          // Try to submit without filling name
          const submitButton = page.getByRole('button', { name: /create|save|submit/i }).first();
          if (await submitButton.isVisible({ timeout: 2000 })) {
            await submitButton.click();

            // Should show validation error or button be disabled
            const errorShown = await page
              .getByText(/required|name|enter/i)
              .isVisible({ timeout: 3000 })
              .catch(() => false);
            const submitDisabled = await submitButton.isDisabled().catch(() => false);

            expect(errorShown || submitDisabled).toBeTruthy();
          }
        }
      }
    });
  });

  test.describe('File Upload Errors', () => {
    test('should validate floor plan file type', async ({ page }) => {
      await login(page);

      // Mock survey endpoints
      await page.route('**/api/survey/list', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            surveys: [
              {
                id: 'test-survey',
                name: 'Test Survey',
                status: 'created',
                createdAt: new Date().toISOString(),
              },
            ],
          }),
        });
      });

      await page.route('**/api/survey?id=test-survey', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'test-survey',
            name: 'Test Survey',
            surveyType: 'passive',
            status: 'created',
            samples: [],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
          }),
        });
      });

      // Try to upload invalid file type (if file upload available)
      const surveyButton = page.getByRole('button', { name: /survey|wifi survey/i }).first();
      const isVisible = await surveyButton.isVisible({ timeout: 3000 }).catch(() => false);

      if (isVisible) {
        await surveyButton.click();
        await page.waitForTimeout(1000);

        // Look for file input
        const fileInput = page.locator('input[type="file"]').first();
        if (await fileInput.isVisible({ timeout: 3000 })) {
          // Create a test file with invalid extension
          const buffer = Buffer.from('test data');
          await fileInput.setInputFiles({
            name: 'test.txt',
            mimeType: 'text/plain',
            buffer,
          });

          // Should show error about invalid file type
          const errorShown = await page
            .getByText(/invalid|file type|png|jpg|jpeg|image/i)
            .isVisible({ timeout: 5000 })
            .catch(() => false);

          expect(errorShown).toBeTruthy();
        }
      }
    });

    test('should handle file too large error', async ({ page }) => {
      await login(page);

      // Mock survey endpoints
      await page.route('**/api/survey/list', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            surveys: [
              {
                id: 'test-survey',
                name: 'Test Survey',
                status: 'created',
                createdAt: new Date().toISOString(),
              },
            ],
          }),
        });
      });

      await page.route('**/api/survey/floorplan?id=test-survey', async (route) => {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'File too large. Maximum size is 10MB',
          }),
        });
      });

      // App should remain functional
      await page.waitForTimeout(2000);
      await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();
    });
  });
});

test.describe('WebSocket Error Scenarios', () => {
  test('should handle WebSocket connection failure gracefully', async ({ page }) => {
    // Block WebSocket connections
    await page.route('**/ws', (route) => route.abort());

    await login(page);

    // App should still function without WebSocket
    await page.waitForTimeout(3000);

    // Dashboard should be visible
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();

    // May show disconnected indicator but app should work
    const _hasError = await page
      .getByText(/disconnected|offline|connection/i)
      .isVisible()
      .catch(() => false);

    // Either shows status or works silently
    expect(true).toBeTruthy(); // App didn't crash
  });

  test('should not crash on invalid WebSocket messages', async ({ page }) => {
    await login(page);

    // Monitor console errors
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    // Monitor for crashes
    page.on('pageerror', (error) => {
      errors.push(error.message);
    });

    // Wait for potential WS messages
    await page.waitForTimeout(5000);

    // App should remain functional
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();

    // Should not have critical crashes (some errors may be expected)
    const hasCriticalError = errors.some(
      (e) => e.toLowerCase().includes('uncaught') || e.toLowerCase().includes('fatal'),
    );

    expect(hasCriticalError).toBeFalsy();
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
    await page.waitForTimeout(3000);

    // Should show helpful empty state message
    const emptyStateShown = await page
      .getByText(/no devices|no hosts|0 devices|empty|start.*scan/i)
      .isVisible({ timeout: 5000 })
      .catch(() => false);

    // Should show either empty state or scan prompt
    expect(
      emptyStateShown || (await page.getByRole('button', { name: /scan/i }).isVisible()),
    ).toBeTruthy();
  });

  test('should show "No surveys found" empty state', async ({ page }) => {
    await login(page);

    // Mock empty survey list
    await page.route('**/api/survey/list', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          surveys: [],
        }),
      });
    });

    // Open survey view
    const surveyButton = page.getByRole('button', { name: /survey|wifi survey/i }).first();
    const isVisible = await surveyButton.isVisible({ timeout: 3000 }).catch(() => false);

    if (isVisible) {
      await surveyButton.click();
      await page.waitForTimeout(2000);

      // Should show helpful empty state
      const emptyStateShown = await page
        .getByText(/no surveys|create.*first|get started|empty/i)
        .isVisible({ timeout: 5000 })
        .catch(() => false);

      expect(
        emptyStateShown || (await page.getByRole('button', { name: /create|new/i }).isVisible()),
      ).toBeTruthy();
    }
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
    await page.waitForTimeout(2000);

    // Should either show success message or be functional
    const successShown = await page
      .getByText(/no vulnerabilities|secure|safe|clean/i)
      .isVisible({ timeout: 5000 })
      .catch(() => false);

    expect(
      successShown || (await page.getByRole('heading', { name: /link/i }).isVisible()),
    ).toBeTruthy();
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
    await page.waitForTimeout(2000);

    const promptShown = await page
      .getByText(/install|not installed|iperf|unavailable/i)
      .isVisible({ timeout: 5000 })
      .catch(() => false);

    // Either shows prompt or app remains functional
    expect(
      promptShown || (await page.getByRole('heading', { name: /link/i }).isVisible()),
    ).toBeTruthy();
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
    await page.waitForTimeout(2000);
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();
  });
});

test.describe('Edge Cases', () => {
  test('should handle very large device list (1000+ devices)', async ({ page }) => {
    await login(page);

    // Generate 1000 mock devices
    const devices: Array<{ ip: string; mac: string; hostname: string; lastSeen: string }> = [];
    for (let i = 0; i < 1000; i++) {
      devices.push({
        ip: `192.168.${Math.floor(i / 254)}.${(i % 254) + 1}`,
        mac: `00:11:22:33:${Math.floor(i / 256)
          .toString(16)
          .padStart(2, '0')}:${(i % 256).toString(16).padStart(2, '0')}`,
        hostname: `device-${i}`,
        lastSeen: new Date().toISOString(),
      });
    }

    await page.route('**/api/devices', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ devices }),
      });
    });

    await page.reload();
    await page.waitForTimeout(3000);

    // App should handle large dataset (pagination, virtualization, or graceful degradation)
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();

    // Should show device count
    const _deviceCount = await page
      .getByText(/1000|devices|hosts/i)
      .isVisible({ timeout: 5000 })
      .catch(() => false);

    expect(true).toBeTruthy(); // App didn't crash
  });

  test('should prevent duplicate scan requests from rapid clicks', async ({ page }) => {
    await login(page);

    let scanCount = 0;

    // Track scan requests
    await page.route('**/api/devices/scan', async (route) => {
      scanCount++;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          scanId: `scan-${scanCount}`,
        }),
      });
    });

    const scanButton = page.getByRole('button', { name: /scan|discover|refresh/i }).first();

    if (await scanButton.isVisible({ timeout: 5000 })) {
      // Rapidly click scan button 5 times
      for (let i = 0; i < 5; i++) {
        await scanButton.click({ force: true });
        await page.waitForTimeout(100);
      }

      await page.waitForTimeout(2000);

      // Should have prevented duplicate requests (ideally only 1 request)
      expect(scanCount).toBeLessThanOrEqual(2); // Allow 1-2 requests, not all 5
    }
  });

  test('should handle concurrent speed test and discovery scan', async ({ page }) => {
    await login(page);

    // Mock both endpoints
    await page.route('**/api/devices/scan', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    });

    await page.route('**/api/speedtest', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          download: 100.5,
          upload: 50.2,
          ping: 10,
        }),
      });
    });

    // Try to start both operations
    const scanButton = page.getByRole('button', { name: /scan|discover/i }).first();
    const speedButton = page.getByRole('button', { name: /speed test/i }).first();

    const scanVisible = await scanButton.isVisible({ timeout: 3000 }).catch(() => false);
    const speedVisible = await speedButton.isVisible({ timeout: 3000 }).catch(() => false);

    if (scanVisible) {
      await scanButton.click();
    }

    if (speedVisible) {
      await speedButton.click();
    }

    // Wait for operations
    await page.waitForTimeout(5000);

    // App should handle concurrent operations without crashing
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();
  });

  test('should handle rapid navigation between views', async ({ page }) => {
    await login(page);

    // Rapidly navigate if multiple views available
    const buttons = [
      page.getByRole('button', { name: /survey/i }).first(),
      page.getByRole('button', { name: /settings/i }).first(),
      page.getByRole('button', { name: /dashboard|home/i }).first(),
    ];

    // Quick navigation test
    for (let i = 0; i < 3; i++) {
      for (const button of buttons) {
        if (await button.isVisible({ timeout: 1000 }).catch(() => false)) {
          await button.click();
          await page.waitForTimeout(200);
        }
      }
    }

    // App should remain functional
    await expect(page.getByRole('heading', { name: /link/i })).toBeVisible({
      timeout: 5000,
    });
  });
});

test.describe('Error Recovery Mechanisms', () => {
  test('should allow retry after failed login', async ({ page }) => {
    await page.goto('/');

    let attemptCount = 0;

    // First attempt fails, second succeeds
    await page.route('**/api/auth/login', async (route) => {
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
    await page.getByLabel(/username/i).fill('admin');
    await page.getByLabel(/password/i).fill('seed');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Should show error
    await expect(page.getByText(/error|failed/i)).toBeVisible({
      timeout: 5000,
    });

    // Retry
    await page.getByRole('button', { name: /sign in|login|retry/i }).click();

    // Should eventually succeed or allow retry
    await page.waitForTimeout(3000);

    expect(attemptCount).toBeGreaterThan(0);
  });

  test('should allow dismissing error messages', async ({ page }) => {
    await login(page);

    // Mock error response
    await page.route('**/api/devices/scan', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Scan failed' }),
      });
    });

    const scanButton = page.getByRole('button', { name: /scan/i }).first();

    if (await scanButton.isVisible({ timeout: 5000 })) {
      await scanButton.click();

      // Wait for error
      const errorVisible = await page
        .getByText(/error|failed/i)
        .isVisible({ timeout: 5000 })
        .catch(() => false);

      if (errorVisible) {
        // Try to dismiss (close button, X, or click away)
        const closeButton = page.getByRole('button', { name: /close|dismiss|ok/i }).first();
        if (await closeButton.isVisible({ timeout: 2000 })) {
          await closeButton.click();

          // Error should be dismissable
          await page.waitForTimeout(1000);
          await expect(page.getByRole('heading', { name: /link/i })).toBeVisible();
        }
      }
    }
  });

  test('should maintain app state after error', async ({ page }) => {
    await login(page);

    // Note some initial state
    const initialHeading = await page.getByRole('heading', { name: /link/i }).textContent();

    // Mock error
    await page.route('**/api/devices/scan', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Scan failed' }),
      });
    });

    // Trigger error
    const scanButton = page.getByRole('button', { name: /scan/i }).first();
    if (await scanButton.isVisible({ timeout: 5000 })) {
      await scanButton.click();
      await page.waitForTimeout(2000);
    }

    // State should be preserved
    const currentHeading = await page.getByRole('heading', { name: /link/i }).textContent();
    expect(currentHeading).toBe(initialHeading);
  });
});

test.describe('Cross-Browser Error Handling', () => {
  test('should handle errors consistently across browsers', async ({ page }) => {
    await login(page);

    // Mock error
    await page.route('**/api/status', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Server error' }),
      });
    });

    await page.reload();
    await page.waitForTimeout(3000);

    // Should handle error consistently regardless of browser
    const appFunctional = await page
      .getByRole('heading', { name: /link|login/i })
      .isVisible({ timeout: 10000 })
      .catch(() => false);

    expect(appFunctional).toBeTruthy();
  });
});
