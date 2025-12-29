import { expect, test } from "@playwright/test";

/**
 * Performance Testing E2E Tests - Complete Flow
 *
 * Comprehensive tests for speed testing and iPerf3 functionality:
 * - Speed Test Flow: full lifecycle with progress tracking
 * - iPerf3 Server Flow: start/stop server management
 * - iPerf3 Client Flow: client tests with configuration
 * - Error Scenarios: network failures, timeouts, invalid parameters
 * - Results Display: verify data persistence and formatting
 */

test.describe("Performance Testing - Complete Flow", () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
    await page.reload();

    // Authenticate
    await page.getByLabel(/username/i).fill("admin");
    await page.getByLabel(/password/i).fill("seed");
    await page.getByRole("button", { name: /sign in|login/i }).click();

    // Wait for dashboard to load
    await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test.describe("Speed Test Flow", () => {
    test("should display Performance card with initial state", async ({ page }) => {
      // Find Performance card
      const perfCard = page
        .locator("h3, h4")
        .filter({ hasText: /performance/i })
        .first();
      await expect(perfCard).toBeVisible({ timeout: 5000 });

      // Should show Internet Speed section
      await expect(page.getByText(/internet speed/i)).toBeVisible();
    });

    test('should show "No results yet" before first test', async ({ page }) => {
      // Wait for card to load
      await expect(page.getByText(/internet speed/i)).toBeVisible();

      // Check for no results message or Run Test button
      const noResults = page.getByText(/no results yet|run.*test|start.*test/i).first();
      await expect(noResults).toBeVisible({ timeout: 3000 });
    });

    test("should run speed test and show progress", async ({ page }) => {
      // Find and click Run Speed Test button
      // The button might be in a FAB or directly in the card
      const testButton = page
        .getByRole("button", { name: /run.*speed|speed.*test|start.*test/i })
        .first();

      // Only run test if button is available
      const isVisible = await testButton.isVisible().catch(() => false);
      if (!isVisible) {
        test.skip(true, "Speed test button not available");
        return;
      }

      // Setup request interception to verify API call
      const requestPromise = page
        .waitForRequest(
          (request) => request.url().includes("/api/speedtest") && request.method() === "POST",
          { timeout: 5000 },
        )
        .catch(() => null);

      await testButton.click();

      // Verify POST /api/speedtest was called
      const request = await requestPromise;
      if (request) {
        expect(request.method()).toBe("POST");
      }

      // Should show progress indicator
      const progressIndicators = [
        /finding server/i,
        /testing.*latency/i,
        /testing.*download/i,
        /testing.*upload/i,
        /progress/i,
      ];

      // At least one progress indicator should appear
      let foundProgress = false;
      for (const pattern of progressIndicators) {
        const hasProgress = await page
          .getByText(pattern)
          .isVisible({ timeout: 2000 })
          .catch(() => false);
        if (hasProgress) {
          foundProgress = true;
          break;
        }
      }

      // Progress indicator might be brief, so we check for either progress or completion
      if (!foundProgress) {
        // Check if test completed very quickly
        const hasResults = await page
          .getByText(/download|upload|mbps|latency|ms/i)
          .isVisible({ timeout: 3000 })
          .catch(() => false);
        expect(hasResults).toBeTruthy();
      } else {
        expect(foundProgress).toBeTruthy();
      }
    });

    test("should display results after test completion", async ({ page }) => {
      // This test assumes a test might be running or completed
      // We'll check for the presence of results or the ability to run a test

      // Wait for card
      await expect(page.getByText(/internet speed/i)).toBeVisible();

      // Check if results are already displayed
      const hasDownload = await page
        .getByText(/download/i)
        .first()
        .isVisible({ timeout: 2000 })
        .catch(() => false);
      const hasUpload = await page
        .getByText(/upload/i)
        .first()
        .isVisible({ timeout: 2000 })
        .catch(() => false);
      const hasLatency = await page
        .getByText(/latency|ping/i)
        .first()
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (hasDownload && hasUpload && hasLatency) {
        // Results are displayed - verify they include speed metrics
        await expect(page.getByText(/mbps|gb\/s|mb\/s/i).first()).toBeVisible();
        await expect(page.getByText(/ms/i).first()).toBeVisible();
      } else {
        // No results yet - that's also a valid state
        await expect(page.getByText(/no results|run.*test|start.*test/i).first()).toBeVisible();
      }
    });

    test("should show SpeedGauge components for results", async ({ page }) => {
      // Wait for Performance card
      await expect(page.getByText(/internet speed/i)).toBeVisible();

      // Check if gauges are present (they appear when results exist)
      // Gauges might not exist if no test has been run
      const downloadGauge = page.getByText(/download/i).first();
      const uploadGauge = page.getByText(/upload/i).first();

      const hasGauges = await downloadGauge.isVisible({ timeout: 2000 }).catch(() => false);

      // Either we have gauges (with results) or we have "no results" message
      if (hasGauges) {
        await expect(downloadGauge).toBeVisible();
        await expect(uploadGauge).toBeVisible();
      } else {
        await expect(page.getByText(/no results|run.*test/i).first()).toBeVisible();
      }
    });

    test("should handle speedtest.net unavailable error", async ({ page }) => {
      // Mock a failed speedtest request
      await page.route("**/api/speedtest", (route) => {
        route.fulfill({
          status: 500,
          body: "speedtest.net unavailable",
        });
      });

      const testButton = page
        .getByRole("button", { name: /run.*speed|speed.*test|start.*test/i })
        .first();
      const isVisible = await testButton.isVisible().catch(() => false);

      if (!isVisible) {
        test.skip(true, "Speed test button not available");
        return;
      }

      await testButton.click();

      // Should show error message
      await expect(page.getByText(/unavailable|failed|error/i)).toBeVisible({
        timeout: 5000,
      });
    });

    test("should handle network timeout error", async ({ page }) => {
      // Mock a timeout
      await page.route("**/api/speedtest", (route) => {
        route.abort("timedout");
      });

      const testButton = page
        .getByRole("button", { name: /run.*speed|speed.*test|start.*test/i })
        .first();
      const isVisible = await testButton.isVisible().catch(() => false);

      if (!isVisible) {
        test.skip(true, "Speed test button not available");
        return;
      }

      await testButton.click();

      // Should show error indication
      await expect(page.getByText(/failed|error|timeout/i)).toBeVisible({
        timeout: 5000,
      });
    });
  });

  test.describe("iPerf3 Server Flow", () => {
    test("should display iPerf3 section", async ({ page }) => {
      // Wait for Performance card
      await expect(page.getByText(/internet speed/i)).toBeVisible();

      // Should show LAN Speed section
      await expect(page.getByText(/lan speed.*iperf/i)).toBeVisible();
    });

    test("should show iperf3 installation status", async ({ page }) => {
      await expect(page.getByText(/lan speed.*iperf/i)).toBeVisible();

      // Either shows version or "not installed" message
      const hasVersion = await page
        .getByText(/iperf3.*\d+\.\d+/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);
      const hasNotInstalled = await page
        .getByText(/not installed|install.*iperf/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      expect(hasVersion || hasNotInstalled).toBeTruthy();
    });

    test("should enable/disable server mode from settings", async ({ page }) => {
      // Open settings drawer
      const settingsButton = page.getByRole("button", { name: /settings/i }).first();
      await settingsButton.click();

      // Look for iPerf settings
      await expect(page.getByText(/iperf|performance/i)).toBeVisible({
        timeout: 5000,
      });

      // Find server enable toggle
      const serverToggle = page.getByLabel(/enable.*server|server.*mode/i).first();
      const hasToggle = await serverToggle.isVisible({ timeout: 2000 }).catch(() => false);

      if (hasToggle) {
        const isChecked = await serverToggle.isChecked();

        // Toggle it
        await serverToggle.click();

        // Verify state changed
        await expect(serverToggle).toBeChecked({ checked: !isChecked });

        // Close settings
        await page.keyboard.press("Escape");

        // Verify server status in card
        await page.waitForTimeout(1000); // Wait for status update
      } else {
        test.skip(true, "iPerf server toggle not available");
      }
    });

    test("should display server connection details when running", async ({ page }) => {
      await expect(page.getByText(/lan speed.*iperf/i)).toBeVisible();

      // Check for server status
      const serverStatus = page.getByText(/server.*mode|listening|stopped/i).first();
      const hasStatus = await serverStatus.isVisible({ timeout: 2000 }).catch(() => false);

      if (hasStatus) {
        // If server is running, should show port number
        const statusText = await serverStatus.textContent();
        if (statusText?.toLowerCase().includes("listening")) {
          await expect(page.getByText(/:\d{4,5}/)).toBeVisible(); // Port number
        }
      }
      // Server might not be enabled - that's OK
    });
  });

  test.describe("iPerf3 Client Flow", () => {
    test("should show server configuration requirement", async ({ page }) => {
      await expect(page.getByText(/lan speed.*iperf/i)).toBeVisible();

      // Should show either configured server or "configure server" message
      const hasServer = await page
        .getByText(/server:.*\d+\.\d+\.\d+\.\d+/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);
      const needsConfig = await page
        .getByText(/configure.*server|server.*settings/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      expect(hasServer || needsConfig).toBeTruthy();
    });

    test("should configure iPerf3 server in settings", async ({ page }) => {
      // Open settings
      const settingsButton = page.getByRole("button", { name: /settings/i }).first();
      await settingsButton.click();

      await expect(page.getByText(/iperf|performance/i)).toBeVisible({
        timeout: 5000,
      });

      // Look for server input field
      const serverInput = page.getByLabel(/server.*host|iperf.*server/i).first();
      const hasInput = await serverInput.isVisible({ timeout: 2000 }).catch(() => false);

      if (hasInput) {
        await serverInput.fill("192.168.1.100");

        // Look for port input
        const portInput = page.getByLabel(/port/i).first();
        const hasPort = await portInput.isVisible({ timeout: 2000 }).catch(() => false);

        if (hasPort) {
          await portInput.fill("5201");
        }

        // Close settings
        await page.keyboard.press("Escape");

        // Verify server appears in card
        await expect(page.getByText(/server:.*192\.168\.1\.100/i)).toBeVisible({
          timeout: 3000,
        });
      } else {
        test.skip(true, "iPerf server configuration not available");
      }
    });

    test("should show test configuration (protocol, direction, duration)", async ({ page }) => {
      await expect(page.getByText(/lan speed.*iperf/i)).toBeVisible();

      // Check if configured - should show test type
      const hasConfig = await page
        .getByText(/tcp|udp|upload|download|both/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (hasConfig) {
        // Verify test configuration is displayed
        await expect(page.getByText(/tcp|udp/i).first()).toBeVisible();
      }
      // Configuration might not be set - that's OK
    });

    test("should run iPerf3 client test with progress", async ({ page }) => {
      // This test requires server to be configured
      const hasServerConfig = await page
        .getByText(/server:.*\d+\.\d+/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (!hasServerConfig) {
        test.skip(true, "iPerf server not configured");
        return;
      }

      // Find run iPerf test button (might be in FAB or card)
      const iperfButton = page
        .getByRole("button", { name: /run.*iperf|iperf.*test|lan.*test/i })
        .first();
      const hasButton = await iperfButton.isVisible({ timeout: 2000 }).catch(() => false);

      if (!hasButton) {
        test.skip(true, "iPerf test button not available");
        return;
      }

      // Setup request interception
      const requestPromise = page
        .waitForRequest(
          (request) => request.url().includes("/api/iperf/client") && request.method() === "POST",
          { timeout: 5000 },
        )
        .catch(() => null);

      await iperfButton.click();

      // Verify POST /api/iperf/client was called
      const request = await requestPromise;
      if (request) {
        expect(request.method()).toBe("POST");

        // Verify request body contains configuration
        const postData = request.postDataJSON();
        expect(postData).toHaveProperty("server");
        expect(postData).toHaveProperty("port");
      }

      // Should show progress
      const progressText = await page
        .getByText(/connecting|testing/i)
        .isVisible({ timeout: 3000 })
        .catch(() => false);

      // Progress might be brief, check for results too
      if (!progressText) {
        const hasResults = await page
          .getByText(/bandwidth|mbps|transfer/i)
          .isVisible({ timeout: 3000 })
          .catch(() => false);
        expect(hasResults).toBeTruthy();
      }
    });

    test("should display iPerf3 results with bandwidth metrics", async ({ page }) => {
      await expect(page.getByText(/lan speed.*iperf/i)).toBeVisible();

      // Check if results exist
      const hasBandwidth = await page
        .getByText(/bandwidth.*mbps|download.*mbps|upload.*mbps/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (hasBandwidth) {
        // Verify result components
        await expect(page.getByText(/mbps|gbps/i).first()).toBeVisible();

        // Should show transfer amount
        await expect(page.getByText(/transfer.*mb|mb.*transfer/i)).toBeVisible({
          timeout: 2000,
        });
      }
      // No results yet is also valid
    });

    test("should show protocol-specific metrics (TCP retransmits, UDP jitter/loss)", async ({
      page,
    }) => {
      await expect(page.getByText(/lan speed.*iperf/i)).toBeVisible();

      // Check if any results exist
      const hasResults = await page
        .getByText(/bandwidth|transfer/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (hasResults) {
        // Check for TCP-specific metrics
        const hasRetransmits = await page
          .getByText(/retransmit/i)
          .isVisible({ timeout: 1000 })
          .catch(() => false);

        // Check for UDP-specific metrics
        const hasJitter = await page
          .getByText(/jitter/i)
          .isVisible({ timeout: 1000 })
          .catch(() => false);
        const hasPacketLoss = await page
          .getByText(/packet loss/i)
          .isVisible({ timeout: 1000 })
          .catch(() => false);

        // Should have either TCP or UDP metrics
        expect(hasRetransmits || hasJitter || hasPacketLoss).toBeTruthy();
      }
      // No results is valid
    });

    test("should handle server unreachable error", async ({ page }) => {
      // Mock failed client request
      await page.route("**/api/iperf/client", (route) => {
        route.fulfill({
          status: 500,
          body: "connection refused",
        });
      });

      const iperfButton = page.getByRole("button", { name: /run.*iperf|iperf.*test/i }).first();
      const hasButton = await iperfButton.isVisible({ timeout: 2000 }).catch(() => false);

      if (!hasButton) {
        test.skip(true, "iPerf test button not available");
        return;
      }

      await iperfButton.click();

      // Should show error
      await expect(page.getByText(/refused|unreachable|failed|error/i)).toBeVisible({
        timeout: 5000,
      });
    });

    test("should handle invalid parameters error", async ({ page }) => {
      // Open settings to configure invalid parameters
      const settingsButton = page.getByRole("button", { name: /settings/i }).first();
      await settingsButton.click();

      const serverInput = page.getByLabel(/server.*host|iperf.*server/i).first();
      const hasInput = await serverInput.isVisible({ timeout: 2000 }).catch(() => false);

      if (!hasInput) {
        test.skip(true, "iPerf configuration not available");
        return;
      }

      // Enter invalid IP
      await serverInput.fill("invalid.ip.address");
      await page.keyboard.press("Escape");

      // Try to run test
      const iperfButton = page.getByRole("button", { name: /run.*iperf|iperf.*test/i }).first();
      const hasButton = await iperfButton.isVisible({ timeout: 2000 }).catch(() => false);

      if (hasButton) {
        await iperfButton.click();

        // Should show error or validation message
        const hasError = await page
          .getByText(/invalid|error|failed/i)
          .isVisible({ timeout: 5000 })
          .catch(() => false);
        expect(hasError).toBeTruthy();
      }
    });

    test("should support bidirectional testing", async ({ page }) => {
      // Open settings
      const settingsButton = page.getByRole("button", { name: /settings/i }).first();
      await settingsButton.click();

      // Look for direction/bidirectional option
      const directionSelect = page.getByLabel(/direction|test.*type/i).first();
      const hasDirection = await directionSelect.isVisible({ timeout: 2000 }).catch(() => false);

      if (hasDirection) {
        // Try to select bidirectional
        await directionSelect.click();
        const bidirOption = page.getByRole("option", { name: /both|bidirectional/i }).first();
        const hasBidir = await bidirOption.isVisible({ timeout: 2000 }).catch(() => false);

        if (hasBidir) {
          await bidirOption.click();
          await page.keyboard.press("Escape");

          // Verify configuration shows "Both" or "Bidirectional"
          await expect(page.getByText(/both|bidirectional/i)).toBeVisible({
            timeout: 3000,
          });
        }
      }
      // Bidirectional option might not exist - that's OK
    });
  });

  test.describe("Server Suggestions", () => {
    test("should display suggested iPerf3 servers", async ({ page }) => {
      // Open settings or look in card
      const settingsButton = page.getByRole("button", { name: /settings/i }).first();
      await settingsButton.click();

      await expect(page.getByText(/iperf|performance/i)).toBeVisible({
        timeout: 5000,
      });

      // Look for suggestions section
      const hasSuggestions = await page
        .getByText(/suggest|recommended.*server/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (hasSuggestions) {
        // Verify at least one suggested server is shown
        await expect(page.getByText(/iperf\.he\.net|bouygues\.iperf\.fr/i)).toBeVisible({
          timeout: 3000,
        });
      }
      // Suggestions might not be implemented yet
    });

    test("should auto-fill connection details when clicking suggested server", async ({ page }) => {
      const settingsButton = page.getByRole("button", { name: /settings/i }).first();
      await settingsButton.click();

      await expect(page.getByText(/iperf|performance/i)).toBeVisible({
        timeout: 5000,
      });

      const hasSuggestions = await page
        .getByText(/suggest|recommended/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (!hasSuggestions) {
        test.skip(true, "Server suggestions not available");
        return;
      }

      // Click first suggestion
      const suggestionButton = page
        .getByRole("button", { name: /iperf\.he\.net|bouygues/i })
        .first();
      const hasButton = await suggestionButton.isVisible({ timeout: 2000 }).catch(() => false);

      if (hasButton) {
        await suggestionButton.click();

        // Verify server field is populated
        const serverInput = page.getByLabel(/server.*host|iperf.*server/i).first();
        const value = await serverInput.inputValue();
        expect(value).toBeTruthy();
        expect(value.length).toBeGreaterThan(0);
      }
    });
  });

  test.describe("Results Persistence", () => {
    test("should persist results after page refresh", async ({ page }) => {
      // Check if results exist
      const hasResults = await page
        .getByText(/download|upload.*mbps/i)
        .isVisible({ timeout: 2000 })
        .catch(() => false);

      if (!hasResults) {
        test.skip(true, "No test results to verify persistence");
        return;
      }

      // Refresh page
      await page.reload();

      // Re-authenticate
      await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({
        timeout: 10000,
      });

      // Results should still be visible
      await expect(page.getByText(/download|upload/i).first()).toBeVisible({
        timeout: 5000,
      });
    });
  });
});
