import { expect, type Page, type Request, test } from '@playwright/test';
import type { SubnetDecisionRequest } from '../src/types/generated/subnet-decision-request';
import type { SubnetResponse } from '../src/types/generated/subnet-response';
import { skipSetupWizard } from './helpers/auth';

/**
 * The discovery card asks the operator about each learned network (seed#3108).
 *
 * The E2E daemon has no router to learn from, so the offered networks are
 * served by a route that answers like the daemon's pending list: an answer
 * removes the network from it. The decision POST still goes through the app's
 * own client, so the body and CSRF header asserted here are what the daemon
 * receives. The daemon's half (nothing pending is swept, a dismissal is never
 * re-offered) is pinned by settings_test.go and the routed NIAC spec.
 */

const PENDING = '**/api/v1/security/devices/subnets/pending';

const offered: SubnetResponse[] = [
  {
    cidr: '10.51.2.0/24',
    name: 'Learned from 10.51.0.1 (routing table)',
    enabled: false,
    learned: true,
  },
  {
    cidr: '10.51.3.0/24',
    name: 'Learned from 10.51.0.1 (interface addresses)',
    enabled: false,
    learned: true,
  },
];

interface Decision {
  body: SubnetDecisionRequest;
  csrf: string | undefined;
}

/** Serves `offered` as the pending list and records each decision posted. */
async function offer(page: Page): Promise<Decision[]> {
  let pending = [...offered];
  const decisions: Decision[] = [];
  await page.route(PENDING, async (route) => {
    const request = route.request();
    if (request.method() === 'POST') {
      const body = request.postDataJSON() as SubnetDecisionRequest;
      decisions.push({ body, csrf: request.headers()['x-csrf-token'] });
      pending = pending.filter((network) => network.cidr !== body.cidr);
      await route.fulfill({ json: { message: 'Decision recorded' } });
      return;
    }
    await route.fulfill({ json: pending });
  });
  return decisions;
}

function row(page: Page, cidr: string) {
  return page.locator(`[data-testid="learned-network"][data-cidr="${cidr}"]`);
}

test.describe('Learned network prompt', () => {
  test.beforeEach(async ({ page }) => {
    await skipSetupWizard(page);
  });

  test('the daemon offers nothing, so no prompt shows', async ({ page }) => {
    const asked = page.waitForResponse(
      (response) =>
        response.request().method() === 'GET' &&
        response.url().endsWith('/api/v1/security/devices/subnets/pending'),
    );
    await page.goto('/network');

    const answered = await asked;
    expect(answered.status()).toBe(200);
    expect(await answered.json()).toEqual([]);
    await expect(page.getByTestId('learned-networks')).toHaveCount(0);
  });

  test('an operator adds one network and dismisses the other', async ({ page }) => {
    const decisions = await offer(page);
    await page.goto('/network');

    const notice = page.getByTestId('learned-networks');
    await expect(notice).toBeVisible({ timeout: 10000 });
    await expect(notice).toContainText('Seed learned 2 new networks');
    await expect(row(page, '10.51.2.0/24')).toContainText('Learned from 10.51.0.1 (routing table)');

    await row(page, '10.51.2.0/24').getByTestId('learned-network-add').click();
    await expect(row(page, '10.51.2.0/24')).toHaveCount(0);
    await expect(notice).toContainText('Seed learned a new network');

    await row(page, '10.51.3.0/24').getByTestId('learned-network-dismiss').click();
    await expect(notice).toHaveCount(0);

    expect(decisions.map((d) => d.body)).toEqual([
      { cidr: '10.51.2.0/24', decision: 'added' },
      { cidr: '10.51.3.0/24', decision: 'dismissed' },
    ]);
    for (const decision of decisions) {
      expect(decision.csrf, 'decision POST carries the CSRF token').toBeTruthy();
    }
  });

  test('the prompt also leads the security page', async ({ page }) => {
    await offer(page);
    await page.goto('/security');

    await expect(page.getByTestId('learned-networks')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('learned-network')).toHaveCount(2);
  });

  test('a viewer is never asked', async ({ page }) => {
    await page.route('**/api/v1/users/me', (route) =>
      route.fulfill({ json: { username: 'reader', role: 'viewer', isActive: true } }),
    );
    await offer(page);
    const asked: Request[] = [];
    page.on('request', (request) => {
      if (request.url().includes('/api/v1/security/devices/subnets/pending')) {
        asked.push(request);
      }
    });

    await page.goto('/network');
    await expect(page.getByTestId('discovery-scan-button').first()).toBeVisible({ timeout: 10000 });

    await expect(page.getByTestId('learned-networks')).toHaveCount(0);
    expect(asked).toEqual([]);
  });

  test('fits a 390 px phone with both actions in reach', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await offer(page);
    await page.goto('/network');

    const notice = page.getByTestId('learned-networks');
    await expect(notice).toBeVisible({ timeout: 10000 });
    const overflow = await notice.evaluate((el) => el.scrollWidth - el.clientWidth);
    expect(overflow, 'notice scrolls sideways').toBeLessThanOrEqual(0);
    const first = row(page, '10.51.2.0/24');
    await first.scrollIntoViewIfNeeded();
    for (const action of ['learned-network-add', 'learned-network-dismiss']) {
      await expect(first.getByTestId(action)).toBeInViewport({ ratio: 1 });
    }
  });
});
