import { expect, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Gateway E2E Tests
 *
 * Brittleness fix #7 of the i18n-fragile-selector sweep pared this
 * file from 11 tests + 14 fragile selectors down to 6 tests + 0
 * i18n-fragile selectors. Removed:
 *
 *   - "should show packet loss percentage" — gated entirely behind
 *     `if (await lossText.isVisible())`. When the gateway has 0
 *     loss the test passed silently.
 *   - "should update gateway status in real-time" — used a CSS
 *     `:text-matches()` regex AND `if (visible)` gating; never
 *     verified a real update, just that some latency string is
 *     still present after a wait.
 *   - "should show success indicator when gateway reachable" /
 *     "should show error indicator when gateway unreachable" —
 *     both ORed `[class*="success"]`/`[class*="error"]` substring
 *     matches with English status-text regex. Collapsed into the
 *     single new "gateway status badge is present" test below
 *     (testid-based, deterministic).
 *
 * Surviving tests use:
 *   - `#card-title-gateway` (stable id from BaseCard)
 *   - `data-testid="gateway-ip"`, `gateway-status-badge`,
 *     `gateway-latency-{min,avg,max}` (added in this PR)
 *   - real-data regexes (IPv4 dotted-quad, "Nms" latency) — these
 *     match the *value format*, not localised UI text, so they
 *     stay valid under es / future locales.
 *   - IPv6 / IPv4 protocol nouns — DNT per the language memo.
 */

/**
 * Synthetic gateway response. The /api/v1/telemetry/gateway endpoint in CI
 * returns empty data (no active interface), so the GatewayCard falls
 * into its "no gateway detected" branch and never emits the gateway-ip,
 * gateway-status-badge, or gateway-latency-* testids the assertions
 * below depend on. Mocking the endpoint with a deterministic dual-stack
 * record makes the populated render branch fire every time.
 */
const MOCK_GATEWAY = {
  gateway: '192.168.1.1',
  reachable: true,
  sent: 10,
  received: 10,
  lossPercent: 0,
  minTime: 1.2,
  maxTime: 3.4,
  avgTime: 2.1,
  lastTime: 2.0,
  status: 'success',
  ipv6: {
    gateway: 'fe80::1',
    reachable: true,
    sent: 10,
    received: 10,
    lossPercent: 0,
    minTime: 1.5,
    maxTime: 3.0,
    avgTime: 2.2,
    lastTime: 2.1,
    status: 'success',
  },
};

async function mockGatewayEndpoint(page: import('@playwright/test').Page): Promise<void> {
  await page.route(/\/api\/v1\/sap\/gateway(\?.*)?$/, (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_GATEWAY),
    });
  });
}

test.describe('Gateway', () => {
  test.beforeEach(async ({ page }) => {
    await mockGatewayEndpoint(page);
    await skipSetupWizard(page);
    await page.goto('/network');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should display Gateway card', async ({ page }) => {
    await expect(page.locator('#card-title-gateway')).toBeVisible({ timeout: 5000 });
  });

  test('should show gateway IP address', async ({ page }) => {
    // Asserts an IPv4 dotted-quad is rendered inside the
    // gateway-ip testid span. The value regex matches the data
    // format, not localised UI text.
    const ip = page.getByTestId('gateway-ip');
    await expect(ip).toBeVisible({ timeout: 5000 });
    await expect(ip).toContainText(/\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/);
  });

  test('should show min/avg/max latency stats', async ({ page }) => {
    // Three stable testids on the latency divs. Previous version
    // used /avg|average/i which would miss under es.
    await expect(page.getByTestId('gateway-latency-min')).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('gateway-latency-avg')).toBeVisible();
    await expect(page.getByTestId('gateway-latency-max')).toBeVisible();
  });

  test('should show a status badge (reachable or unreachable)', async ({ page }) => {
    // Single deterministic assertion replacing the previous
    // "success indicator OR error indicator" weak OR. The badge
    // is always rendered; success vs error is conveyed via
    // StatusBadge's ARIA label which is i18n-localised, so the
    // testid wrapper is the stable anchor.
    await expect(page.getByTestId('gateway-status-badge')).toBeVisible({ timeout: 5000 });
  });

  test('should show IPv6 or IPv4 gateway entry', async ({ page }) => {
    // IPv4 / IPv6 are protocol nouns and DNT per the language
    // memo. Asserting either one is present catches both single-
    // stack and dual-stack hosts without committing to one.
    const ipv6Text = page.getByText(/ipv6/i);
    const ipv4Pattern = page.getByText(/\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/);
    const [hasIpv6, hasIpv4] = await Promise.all([
      ipv6Text.first().isVisible(),
      ipv4Pattern.first().isVisible(),
    ]);
    expect(hasIpv6 || hasIpv4).toBeTruthy();
  });
});

test.describe('Gateway Help', () => {
  test.beforeEach(async ({ page }) => {
    await mockGatewayEndpoint(page);
    await skipSetupWizard(page);
    await page.goto('/network');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should open the page-header help panel', async ({ page }) => {
    // Previously matched the help open button by /help/i regex, and
    // the gateway help section by /gateway/i. Both miss under es.
    // Replaced with stable testids on PageHeader: page-header-help-
    // button opens the panel; page-header-help-close dismisses it.
    await page.getByTestId('page-header-help-button').first().click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    await page.getByTestId('page-header-help-close').click();
    await expect(page.getByRole('dialog')).toBeHidden();
  });
});
