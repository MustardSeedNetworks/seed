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
      // color-contrast belongs to the theme row (UI-SEED-7, seed#2647), which
      // has its own contrast acceptance and its own canonical palette. Judging
      // it here measured the rail gradient by hand and disagreed between
      // engines: webkit reports the gradient as a violation where chromium
      // reports it as incomplete.
      {
        runOnly: ['wcag2a', 'wcag2aa', 'wcag21aa'],
        rules: { 'color-contrast': { enabled: false } },
      },
    );
    return {
      violations: report.violations.map(({ id, nodes }) => ({
        id,
        nodes: nodes.map(({ html }) => html),
      })),
    };
  });
  await test
    .info()
    .attach('axe-results.json', { body: JSON.stringify(result), contentType: 'application/json' });
  expect(result.violations).toEqual([]);
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
