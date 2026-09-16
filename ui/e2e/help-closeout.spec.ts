import { expect, type Locator, type Page, test } from '@playwright/test';
import axe from 'axe-core';
import enHelp from '../../internal/i18n/locales/en/help.json' with { type: 'json' };
import esHelp from '../../internal/i18n/locales/es/help.json' with { type: 'json' };
import {
  disableAnimations,
  revealSidebar,
  sidebarHelpButton,
  sidebarSettingsButton,
  skipSetupWizard,
} from './helpers/auth';

async function tabTo(page: Page, target: Locator): Promise<void> {
  for (let count = 0; count < 100; count++) {
    await page.keyboard.press('Tab');
    if (await target.evaluate((element) => element === document.activeElement)) return;
  }
  throw new Error(`Target is not reachable through Tab: ${await target.textContent()}`);
}

async function expectAccessible(page: Page): Promise<void> {
  const result = await page.evaluate(async () => {
    const engine = (window as Window & { axe: typeof import('axe-core') }).axe;
    const report = await engine.run(
      {
        include: [
          [
            document.querySelector('[data-testid=help-drawer]')
              ? '[data-testid=help-drawer]'
              : 'button[aria-describedby], input[aria-describedby]',
          ],
        ],
      },
      { runOnly: ['wcag2a', 'wcag2aa', 'wcag21aa'] },
    );
    const contrast: { text: string; ratio: number }[] = [];
    for (const rule of report.incomplete) {
      if (rule.id !== 'color-contrast') throw new Error(`Unreviewed axe result: ${rule.id}`);
      for (const node of rule.nodes) {
        if (!node.any.every((check) => check.data?.messageKey === 'bgGradient')) {
          throw new Error(`Unreviewed contrast result: ${node.failureSummary}`);
        }
        const [selector] = node.target;
        if (typeof selector !== 'string') throw new Error('Unexpected contrast target');
        const element = document.querySelector(selector);
        const rail = element?.closest('aside');
        if (!element || !rail) throw new Error('Gradient contrast target is outside the rail');
        const canvas = document.createElement('canvas');
        const context = canvas.getContext('2d');
        if (!context) throw new Error('Canvas color conversion is unavailable');
        const rgb = (color: string): number[] => {
          context.clearRect(0, 0, 1, 1);
          context.fillStyle = color;
          context.fillRect(0, 0, 1, 1);
          const values = [...context.getImageData(0, 0, 1, 1).data];
          if (values[3] !== 255) throw new Error(`Non-opaque contrast color: ${color}`);
          return values.slice(0, 3);
        };
        const railStyle = getComputedStyle(rail);
        const from = rgb(railStyle.getPropertyValue('--color-rail-from'));
        const to = rgb(railStyle.getPropertyValue('--color-rail-to'));
        // Component-wise maxima bound every gradient stop's luminance from above.
        const background = from.map((channel, index) => Math.max(channel, to[index] ?? 0));
        const luminance = (channels: number[]): number =>
          channels.reduce((sum, channel, index) => {
            const value = channel / 255;
            return (
              sum +
              (value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4) *
                ([0.2126, 0.7152, 0.0722][index] ?? 0)
            );
          }, 0);
        const foreground = luminance(rgb(getComputedStyle(element).color));
        const backdrop = luminance(background);
        if (foreground <= backdrop) throw new Error('Expected light text on the dark rail');
        contrast.push({
          text: element.textContent ?? '',
          ratio: (foreground + 0.05) / (backdrop + 0.05),
        });
      }
    }
    return {
      violations: report.violations.map(({ id, nodes }) => ({
        id,
        nodes: nodes.map(({ html }) => html),
      })),
      contrast,
      incomplete: report.incomplete.map(({ id, nodes }) => ({
        id,
        nodes: nodes.map(({ html, failureSummary, any }) => ({ html, failureSummary, any })),
      })),
    };
  });
  await test
    .info()
    .attach('axe-results.json', { body: JSON.stringify(result), contentType: 'application/json' });
  expect(result.violations).toEqual([]);
  for (const check of result.contrast) expect(check.ratio, check.text).toBeGreaterThanOrEqual(4.5);
}

test.beforeEach(async ({ page }) => {
  await skipSetupWizard(page);
  await disableAnimations(page);
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.addInitScript({ content: axe.source });
});

for (const width of [1440, 390]) {
  for (const [route, section, body] of [
    ['/network', 'network', esHelp.content.cardHelp.NeighbourCacheCard.description],
    ['/security', 'security', esHelp.content.cardHelp.MfaCard.description],
    ['/performance', 'performance', esHelp.content.performanceTests.description],
  ] as const) {
    test(`${width}px Spanish header and footer help agree on ${route}`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.addInitScript(() => localStorage.setItem('language', 'es'));
      await page.goto(route);
      await expect(page.getByTestId('page-header-title')).toBeVisible();
      await page.getByTestId('page-header-help-button').click();
      const content = page.getByTestId('help-drawer-content');
      await expect(content.getByRole('heading').first()).toHaveText(esHelp.sections[section]);
      await expect(content).toContainText(body);
      if (width === 390 && route === '/network')
        await test.info().attach('help-390.png', {
          body: await page.screenshot({ animations: 'disabled' }),
          contentType: 'image/png',
        });
      if (width === 390) await page.getByTestId('help-section-select').selectOption('glossary');
      else
        await page
          .getByTestId('help-drawer')
          .getByRole('button', { name: esHelp.sections.glossary, exact: true })
          .click();
      await page.getByTestId('help-drawer-close').click();
      await revealSidebar(page);
      await sidebarHelpButton(page).click();
      await expect(content.getByRole('heading').first()).toHaveText(esHelp.sections[section]);
      await expect(content).toContainText(body);
      await expect(page.getByTestId('help-drawer')).not.toContainText(/vv\d/);
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
        true,
      );
      await page.keyboard.press('Escape');
      await expect(page.getByTestId('help-drawer')).not.toBeVisible();
    });
  }
}

test('all 25 Spanish sections render and remain selectable at 390px', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.addInitScript(() => localStorage.setItem('language', 'es'));
  await page.goto('/network');
  await page.getByTestId('page-header-help-button').click();
  for (const [section, title] of Object.entries(esHelp.sections)) {
    await page.getByTestId('help-section-select').selectOption(section);
    const content = page.getByTestId('help-drawer-content');
    await expect(content.getByRole('heading').first()).toHaveText(title);
    expect((await content.textContent())?.length).toBeGreaterThan(100);
    await expect(content).not.toContainText(/content\.[a-z]+|cards:|pages:|common:/);
  }
});

test('HTTP timing segments are reached with Tab, described and dismissed with Escape', async ({
  page,
}) => {
  await page.route('**/api/v1/telemetry/probes/run', (route) =>
    route.fulfill({
      json: {
        hasTests: true,
        pingResults: [],
        tcpResults: [],
        udpResults: [],
        httpResults: [
          {
            name: 'Timing fixture',
            url: 'https://example.test/',
            success: true,
            latency: 150,
            dnsLatency: 10,
            tcpConnect: 20,
            tlsLatency: 30,
            ttfbLatency: 40,
            status: 200,
          },
        ],
      },
    }),
  );
  await page.goto('/performance');
  const segments = page.getByTestId('http-timing-segment');
  await expect(segments).toHaveCount(5);
  for (let index = 0; index < 5; index++) {
    const trigger = segments.nth(index);
    await tabTo(page, trigger);
    await expect(trigger).toBeFocused();
    await expect(page.getByRole('tooltip')).toBeVisible();
    const description = await trigger.getAttribute('aria-describedby');
    expect(description).toBeTruthy();
    await expect(trigger).toHaveAccessibleDescription(await page.getByRole('tooltip').innerText());
    await page.keyboard.press('Escape');
    await expect(page.getByRole('tooltip')).toHaveCount(0);
    await expect(trigger).toBeFocused();
  }
});

test('a viewer can read disabled-action reasons without starting a scan', async ({ page }) => {
  await page.route('**/api/v1/users/me', (route) =>
    route.fulfill({ json: { username: 'reader', role: 'viewer', isActive: true } }),
  );
  const writes: string[] = [];
  page.on('request', (request) => {
    if (request.method() === 'POST' && /insecure|guest/.test(request.url()))
      writes.push(request.url());
  });
  await page.goto('/security');
  // The concrete scan control is disabled semantically so its explanation stays in the tab order.
  const scan = page.locator('button[aria-disabled="true"]').filter({ hasText: /scan/i }).first();
  await expect(scan).toBeVisible();
  await tabTo(page, scan);
  await expect(scan).toHaveAccessibleDescription(/operator|read.only|permission|viewer/i);
  await page.keyboard.press('Enter');
  await page.keyboard.press('Space');
  const bounds = await scan.boundingBox();
  if (!bounds) throw new Error('Disabled scan control has no pointer target');
  await page.mouse.click(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
  expect(writes).toEqual([]);
  await expect(scan).toHaveAttribute('aria-disabled', 'true');
});

for (const route of [
  '/link',
  '/network',
  '/path',
  '/wifi',
  '/security',
  '/performance',
  '/reports',
  '/logs',
  '/polling-targets',
  '/topology',
  '/alerts',
]) {
  test(`accessible help and tooltip surfaces on ${route}`, async ({ page }) => {
    await page.goto(route);
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    await expect(page.getByTestId('page-header-help-button')).toHaveAccessibleName(/help/i);
    await expect(page.locator('[title]')).toHaveCount(0);
    await expectAccessible(page);
    await page.getByTestId('page-header-help-button').click();
    await expect(page.getByTestId('help-drawer')).toBeVisible();
    await expectAccessible(page);
  });
}

test('collapsed sidebar buttons keep their names and hover-only Escape closes the bubble', async ({
  page,
}) => {
  await page.addInitScript(() => localStorage.setItem('stem-sidebar-collapsed', 'true'));
  await page.goto('/network');
  const help = sidebarHelpButton(page);
  await expect(help).toHaveAccessibleName('Open help');
  await help.hover();
  await expect(page.getByRole('tooltip')).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('tooltip')).toHaveCount(0);
  await help.click();
  await expect(page.getByTestId('help-drawer-content').getByRole('heading').first()).toHaveText(
    enHelp.sections.network,
  );
});

test('the fresh embedded build reports a UI hash', async ({ request }) => {
  const response = await request.get('/__version');
  expect(response.ok()).toBe(true);
  const version: unknown = await response.json();
  expect(version).toMatchObject({ uiBuildHash: expect.stringMatching(/^[a-f0-9]{32}$/) });
  await test
    .info()
    .attach('version.json', { body: JSON.stringify(version), contentType: 'application/json' });
});

test('footer help follows trailing slashes and browser history', async ({ page }) => {
  await page.goto('/network/');
  await sidebarHelpButton(page).click();
  const content = page.getByTestId('help-drawer-content');
  await expect(content.getByRole('heading').first()).toHaveText(enHelp.sections.network);
  await page.getByTestId('help-drawer-close').click();
  await page.getByTestId('sidebar-nav-security').filter({ visible: true }).click();
  await sidebarHelpButton(page).click();
  await expect(content.getByRole('heading').first()).toHaveText(enHelp.sections.security);
  await page.goBack();
  await expect(page.getByTestId('help-drawer')).toBeHidden();
  await sidebarHelpButton(page).click();
  await expect(content.getByRole('heading').first()).toHaveText(enHelp.sections.network);
  await page.goForward();
  await expect(page.getByTestId('help-drawer')).toBeHidden();
  await sidebarHelpButton(page).click();
  await expect(content.getByRole('heading').first()).toHaveText(enHelp.sections.security);
});

test('opening help on another route resets a previous search', async ({ page }) => {
  await page.goto('/network');
  await sidebarHelpButton(page).click();
  await page.getByPlaceholder(enHelp.modal.searchPlaceholder).fill('glossary');
  await page.getByTestId('help-drawer-close').click();
  await page.getByTestId('sidebar-nav-security').filter({ visible: true }).click();
  await sidebarHelpButton(page).click();
  await expect(page.getByPlaceholder(enHelp.modal.searchPlaceholder)).toHaveValue('');
  await expect(page.getByTestId('help-drawer-content').getByRole('heading').first()).toHaveText(
    enHelp.sections.security,
  );
});

test('viewer settings retain disabled controls and a reachable reason', async ({ page }) => {
  await page.route('**/api/v1/users/me', (route) =>
    route.fulfill({ json: { username: 'reader', role: 'viewer', isActive: true } }),
  );
  await page.goto('/network');
  await sidebarSettingsButton(page).click();
  await page.getByTestId('discovery-settings-section').click();
  const fieldset = page
    .getByTestId('discovery-settings-section')
    .locator('fieldset[disabled]')
    .first();
  const reason = fieldset.locator('legend button');
  await tabTo(page, reason);
  await expect(reason).toHaveAccessibleDescription(/read.only|viewer/i);
  await expect(fieldset.locator('input').first()).toBeDisabled();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('tooltip')).toHaveCount(0);
  await expect(page.getByTestId('settings-drawer')).toBeVisible();
});

for (const width of [1440, 390]) {
  test(`${width}px help content scrolls by keyboard and returns focus to the section navigation`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 700 });
    await page.goto('/network');
    await page.getByTestId('page-header-help-button').click();
    const back = page.getByTestId(width === 390 ? 'help-return-to-select' : 'help-return-to-toc');
    await tabTo(page, back);
    await expect(back).toBeFocused();
    const content = page.getByTestId('help-drawer-content');
    await page.keyboard.press('PageDown');
    await expect.poll(() => content.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
    await page.keyboard.press('Enter');
    await expect(
      page.getByTestId(width === 390 ? 'help-section-select' : 'help-section-network'),
    ).toBeFocused();
  });
}
