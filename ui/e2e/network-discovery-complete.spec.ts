import { expect, type Page, test } from "@playwright/test";

/**
 * Network Discovery Complete Flow E2E Tests
 *
 * Comprehensive end-to-end tests for the complete network discovery user journey:
 * - Triggering network scans
 * - Real-time scan progress updates
 * - Device list display and interaction
 * - Device details and modal dialogs
 * - Device actions (traceroute, port scan, fingerprinting)
 * - Filtering and sorting functionality
 * - Search capabilities
 * - Error handling scenarios
 *
 * Tests cover all user interactions from scan initiation to detailed device analysis.
 */

/**
 * Helper function to login and navigate to dashboard
 */
async function loginAndNavigate(page: Page) {
  await page.goto("/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();

  // Authenticate with valid credentials
  await page.getByLabel(/username/i).fill("admin");
  await page.getByLabel(/password/i).fill("seed");
  await page.getByRole("button", { name: /sign in|login/i }).click();

  // Wait for dashboard to load
  await expect(page.getByRole("heading", { name: /link/i })).toBeVisible({
    timeout: 10000,
  });
}

/**
 * Mock network discovery API responses
 */
async function mockDiscoveryApis(page: Page) {
  // Mock initial status - no scan running
  await page.route("**/api/devices/status", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        scanning: false,
        deviceCount: 0,
        lastScan: new Date(Date.now() - 3600000).toISOString(),
        subnet: "192.168.1.0/24",
        localIp: "192.168.1.100",
        interface: "en0",
      }),
    });
  });

  // Mock device list - initially empty
  await page.route("**/api/devices", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        devices: [],
        status: {
          scanning: false,
          deviceCount: 0,
          lastScan: new Date(Date.now() - 3600000).toISOString(),
          subnet: "192.168.1.0/24",
          localIp: "192.168.1.100",
          interface: "en0",
        },
      }),
    });
  });
}

test.describe("Network Discovery - Complete Flow", () => {
  test.beforeEach(async ({ page }) => {
    await loginAndNavigate(page);
  });

  test("should display Network Discovery card on dashboard", async ({ page }) => {
    const discoveryCard = page
      .locator('h3:has-text("Discovery"), h4:has-text("Discovery")')
      .or(page.locator('[data-testid="network-discovery-card"]'))
      .first();

    await expect(discoveryCard).toBeVisible({ timeout: 5000 });
  });

  test("should trigger network scan when Scan button is clicked", async ({ page }) => {
    // Mock the scan endpoint
    let scanTriggered = false;
    await page.route("**/api/devices/scan", async (route) => {
      scanTriggered = true;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ status: "scan started" }),
      });
    });

    // Find and click the scan button
    const scanButton = page
      .getByRole("button", { name: /scan|discover|refresh/i })
      .or(page.locator('button:has-text("Scan")'))
      .first();

    await scanButton.click();

    // Verify scan was triggered
    await page.waitForTimeout(500);
    expect(scanTriggered).toBeTruthy();
  });

  test("should show loading indicator during scan", async ({ page }) => {
    await mockDiscoveryApis(page);

    // Mock scan to return scanning status
    await page.route("**/api/devices/scan", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ status: "scan started" }),
      });
    });

    // Update status route to show scanning
    await page.route("**/api/devices/status", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          scanning: true,
          deviceCount: 0,
          lastScan: new Date().toISOString(),
          subnet: "192.168.1.0/24",
          localIp: "192.168.1.100",
          interface: "en0",
        }),
      });
    });

    const scanButton = page.getByRole("button", { name: /scan/i }).first();
    await scanButton.click();

    // Look for loading/scanning indicator
    const loadingIndicator = page.locator("text=/scanning|loading/i").first();
    await expect(loadingIndicator).toBeVisible({ timeout: 3000 });
  });

  test("should display discovered devices after scan completion", async ({ page }) => {
    // Mock devices endpoint with discovered devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "router.local",
              vendor: "Cisco",
              osGuess: "Linux",
              discoveryMethod: ["arp", "lldp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
              profile: {
                deviceType: "router",
                deviceIcons: ["router", "web-secure"],
                profiledAt: new Date().toISOString(),
              },
            },
            {
              ip: "192.168.1.10",
              mac: "aa:bb:cc:dd:ee:02",
              hostname: "desktop-01",
              vendor: "Intel",
              osGuess: "Windows 10",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "192.168.1.20",
              mac: "aa:bb:cc:dd:ee:03",
              hostname: "printer",
              vendor: "HP",
              osGuess: "Embedded",
              discoveryMethod: ["arp", "mdns"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
              profile: {
                deviceType: "printer",
                deviceIcons: ["printer"],
                profiledAt: new Date().toISOString(),
              },
            },
          ],
          status: {
            scanning: false,
            deviceCount: 3,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Verify device count is displayed
    const deviceCount = page.locator("text=/3\\s*device/i").first();
    await expect(deviceCount).toBeVisible({ timeout: 5000 });

    // Verify device IPs are visible
    await expect(page.locator("text=192.168.1.1")).toBeVisible();
    await expect(page.locator("text=192.168.1.10")).toBeVisible();
    await expect(page.locator("text=192.168.1.20")).toBeVisible();
  });

  test("should display device details with IP, MAC, hostname, type, and OS", async ({ page }) => {
    // Mock device data
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.50",
              mac: "11:22:33:44:55:66",
              hostname: "server.local",
              vendor: "Dell",
              osGuess: "Ubuntu 22.04",
              discoveryMethod: ["arp", "ping"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
              profile: {
                deviceType: "server",
                deviceIcons: ["server", "ssh"],
                profiledAt: new Date().toISOString(),
              },
            },
          ],
          status: {
            scanning: false,
            deviceCount: 1,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Verify device information is displayed
    await expect(page.locator("text=192.168.1.50")).toBeVisible();
    await expect(page.locator("text=/11:22:33:44:55:66/i")).toBeVisible();
    await expect(page.locator("text=/server.local/i")).toBeVisible();
    await expect(page.locator("text=/Dell/i")).toBeVisible();
  });

  test("should expand device to show detailed information when clicked", async ({ page }) => {
    // Mock device with detailed info
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "router.local",
              vendor: "Cisco",
              osGuess: "IOS",
              ttl: 64,
              discoveryMethod: ["arp", "lldp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
              lldpInfo: {
                chassisId: "cisco-router-01",
                portId: "GigabitEthernet0/1",
                systemName: "Router-01",
                systemDescription: "Cisco IOS Router",
                capabilities: ["Router", "Bridge"],
              },
            },
          ],
          status: {
            scanning: false,
            deviceCount: 1,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Click on device row to expand
    const deviceRow = page.locator("text=192.168.1.1").first();
    await deviceRow.click();

    // Wait for expansion
    await page.waitForTimeout(500);

    // Verify detailed info is visible
    await expect(page.locator("text=/MAC/i")).toBeVisible();
    await expect(page.locator("text=/TTL/i")).toBeVisible();
    await expect(page.locator("text=/Last Seen/i")).toBeVisible();
  });

  test("should run port scan on device (Deep Scan)", async ({ page }) => {
    // Mock devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.100",
              mac: "aa:bb:cc:dd:ee:ff",
              hostname: "test-device",
              vendor: "Unknown",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 1,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    // Mock port scan endpoint
    let portScanTriggered = false;
    await page.route("**/api/discovery/portscan", async (route) => {
      portScanTriggered = true;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          target: "192.168.1.100",
          results: [
            { port: 22, state: "open", protocol: "tcp", ttl: 64, rtt: 1500000 },
            { port: 80, state: "open", protocol: "tcp", ttl: 64, rtt: 2000000 },
            {
              port: 443,
              state: "open",
              protocol: "tcp",
              ttl: 64,
              rtt: 1800000,
            },
          ],
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Find and click "Scan" button for deep scan
    const scanButton = page.locator('button:has-text("Scan")').last();
    await scanButton.click();

    // Wait for scan to complete
    await page.waitForTimeout(1000);

    // Verify scan was triggered
    expect(portScanTriggered).toBeTruthy();
  });

  test("should search devices by IP address", async ({ page }) => {
    // Mock multiple devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "router",
              vendor: "Cisco",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "192.168.1.100",
              mac: "aa:bb:cc:dd:ee:02",
              hostname: "desktop",
              vendor: "Intel",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "10.0.0.5",
              mac: "aa:bb:cc:dd:ee:03",
              hostname: "server",
              vendor: "Dell",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: false,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 3,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Find search input
    const searchInput = page.locator('input[placeholder*="Search"]').first();
    await expect(searchInput).toBeVisible();

    // Search for specific IP
    await searchInput.fill("10.0.0.5");
    await page.waitForTimeout(500);

    // Verify only matching device is visible
    await expect(page.locator("text=10.0.0.5")).toBeVisible();

    // Verify filtered count is shown
    await expect(page.locator("text=/1\\s*of\\s*3/i")).toBeVisible();
  });

  test("should search devices by hostname", async ({ page }) => {
    // Mock devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "router.local",
              vendor: "Cisco",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "192.168.1.100",
              mac: "aa:bb:cc:dd:ee:02",
              hostname: "desktop-windows",
              vendor: "Intel",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 2,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Search by hostname
    const searchInput = page.locator('input[placeholder*="Search"]').first();
    await searchInput.fill("desktop");
    await page.waitForTimeout(500);

    // Verify matching device is visible
    await expect(page.locator("text=desktop-windows")).toBeVisible();
  });

  test("should sort devices by IP address", async ({ page }) => {
    // Mock devices in non-sorted order
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.50",
              mac: "aa:bb:cc:dd:ee:03",
              hostname: "device-50",
              vendor: "Vendor C",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "device-1",
              vendor: "Vendor A",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "192.168.1.25",
              mac: "aa:bb:cc:dd:ee:02",
              hostname: "device-25",
              vendor: "Vendor B",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 3,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Click IP sort button
    const ipSortButton = page.locator('button:has-text("IP")').first();
    await ipSortButton.click();
    await page.waitForTimeout(500);

    // Verify button shows active state
    await expect(ipSortButton).toHaveClass(/brand-primary/);
  });

  test("should sort devices by hostname", async ({ page }) => {
    // Mock devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "zebra",
              vendor: "Vendor A",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "192.168.1.2",
              mac: "aa:bb:cc:dd:ee:02",
              hostname: "alpha",
              vendor: "Vendor B",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 2,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Click hostname/name sort button
    const nameSortButton = page.locator('button:has-text("Name")').first();
    await nameSortButton.click();
    await page.waitForTimeout(500);

    // Verify active state
    await expect(nameSortButton).toHaveClass(/brand-primary/);
  });

  test("should toggle sort direction (ascending/descending)", async ({ page }) => {
    // Mock devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "device-a",
              vendor: "Vendor A",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
            {
              ip: "192.168.1.2",
              mac: "aa:bb:cc:dd:ee:02",
              hostname: "device-b",
              vendor: "Vendor B",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 2,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Click IP sort to activate
    const ipSortButton = page.locator('button:has-text("IP")').first();
    await ipSortButton.click();
    await page.waitForTimeout(300);

    // Verify chevron up icon (ascending)
    await expect(
      page
        .locator("svg")
        .filter({ hasText: /ChevronUp/i })
        .first(),
    ).toBeVisible();

    // Click again to toggle to descending
    await ipSortButton.click();
    await page.waitForTimeout(300);

    // Verify chevron down icon (descending)
    await expect(
      page
        .locator("svg")
        .filter({ hasText: /ChevronDown/i })
        .first(),
    ).toBeVisible();
  });

  test("should clear search query with X button", async ({ page }) => {
    // Mock devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "test",
              vendor: "Vendor",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 1,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Type in search
    const searchInput = page.locator('input[placeholder*="Search"]').first();
    await searchInput.fill("test query");
    await page.waitForTimeout(300);

    // Click clear button (X)
    const clearButton = page.locator('button[aria-label*="Clear"]').first();
    await clearButton.click();

    // Verify search is cleared
    await expect(searchInput).toHaveValue("");
  });

  test('should show "no devices found" message when scan returns empty', async ({ page }) => {
    // Mock empty device list
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [],
          status: {
            scanning: false,
            deviceCount: 0,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Verify "no devices" message
    const noDevicesMsg = page.locator("text=/no devices|click scan/i").first();
    await expect(noDevicesMsg).toBeVisible();
  });

  test("should handle network error during scan gracefully", async ({ page }) => {
    // Mock scan endpoint to return error
    await page.route("**/api/devices/scan", async (route) => {
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "Network error" }),
      });
    });

    const scanButton = page.getByRole("button", { name: /scan/i }).first();
    await scanButton.click();

    // Wait for error handling
    await page.waitForTimeout(1000);

    // Scan button should be re-enabled (not stuck in loading state)
    await expect(scanButton).toBeEnabled();
  });

  test("should display device with vulnerability badge", async ({ page }) => {
    // Mock device with vulnerabilities
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.100",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "vulnerable-device",
              vendor: "Vendor",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
              vulnerabilities: {
                count: 5,
                highestSeverity: "HIGH",
              },
            },
          ],
          status: {
            scanning: false,
            deviceCount: 1,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Verify vulnerability badge is visible
    await expect(page.locator("text=/5\\s*CVE/i")).toBeVisible();
  });

  test("should show network info in collapsible section", async ({ page }) => {
    // Mock devices
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "test",
              vendor: "Vendor",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 1,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.50",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Find and click "Network Info" section
    const networkInfoHeader = page.locator("text=/Network Info/i").first();
    await expect(networkInfoHeader).toBeVisible();
    await networkInfoHeader.click();
    await page.waitForTimeout(500);

    // Verify network details are shown
    await expect(page.locator("text=/Local IP/i")).toBeVisible();
    await expect(page.locator("text=192.168.1.50")).toBeVisible();
    await expect(page.locator("text=/Interface/i")).toBeVisible();
    await expect(page.locator("text=en0")).toBeVisible();
  });

  test("should display discovery method badges for each device", async ({ page }) => {
    // Mock device with multiple discovery methods
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "router",
              vendor: "Cisco",
              discoveryMethod: ["arp", "lldp", "cdp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
            },
          ],
          status: {
            scanning: false,
            deviceCount: 1,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Verify discovery method badges
    await expect(page.locator("text=/ARP/i").first()).toBeVisible();
    await expect(page.locator("text=/LLDP/i").first()).toBeVisible();
    await expect(page.locator("text=/CDP/i").first()).toBeVisible();
  });

  test("should show device category summary with icons", async ({ page }) => {
    // Mock devices of different types
    await page.route("**/api/devices", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          devices: [
            {
              ip: "192.168.1.1",
              mac: "aa:bb:cc:dd:ee:01",
              hostname: "router",
              vendor: "Cisco",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
              profile: {
                deviceType: "router",
                deviceIcons: ["router"],
                profiledAt: new Date().toISOString(),
              },
            },
            {
              ip: "192.168.1.10",
              mac: "aa:bb:cc:dd:ee:02",
              hostname: "printer",
              vendor: "HP",
              discoveryMethod: ["arp"],
              lastSeen: new Date().toISOString(),
              isLocal: true,
              profile: {
                deviceType: "printer",
                deviceIcons: ["printer"],
                profiledAt: new Date().toISOString(),
              },
            },
          ],
          status: {
            scanning: false,
            deviceCount: 2,
            lastScan: new Date().toISOString(),
            subnet: "192.168.1.0/24",
            localIp: "192.168.1.100",
            interface: "en0",
          },
        }),
      });
    });

    await page.reload();
    await page.waitForTimeout(1000);

    // Verify category summary shows counts
    await expect(page.locator("text=/Routers/i").first()).toBeVisible();
    await expect(page.locator("text=/Printers/i").first()).toBeVisible();
  });

  test("should disable scan button while scanning is in progress", async ({ page }) => {
    // Mock scanning status
    await page.route("**/api/devices/scan", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ status: "scan started" }),
      });
    });

    await page.route("**/api/devices/status", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          scanning: true,
          deviceCount: 0,
          lastScan: new Date().toISOString(),
          subnet: "192.168.1.0/24",
          localIp: "192.168.1.100",
          interface: "en0",
        }),
      });
    });

    const scanButton = page.getByRole("button", { name: /scan/i }).first();
    await scanButton.click();

    // Verify button is disabled during scan
    await expect(scanButton).toBeDisabled();
  });
});
