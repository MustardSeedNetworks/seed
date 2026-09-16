import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { checkHelp, consumedHelpKeys } from './check-help-i18n.ts';

test('an unused help key fails even when mentioned in a comment or test', () => {
  const root = mkdtempSync(join(tmpdir(), 'seed-help-i18n-'));
  try {
    mkdirSync(join(root, 'ui/src'), { recursive: true });
    mkdirSync(join(root, 'internal/i18n/locales/en'), { recursive: true });
    writeFileSync(
      join(root, 'internal/i18n/locales/en/help.json'),
      JSON.stringify({ modal: { title: 'Help', orphan: 'Unused' } }),
    );
    writeFileSync(
      join(root, 'ui/src/Help.tsx'),
      "const {t}=useTranslation('help'); t('modal.title'); // t('modal.orphan')\n",
    );
    writeFileSync(join(root, 'ui/src/Help.test.tsx'), "t('help:modal.orphan');");
    assert.deepEqual(checkHelp(root), ['modal.orphan']);
    writeFileSync(
      join(root, 'ui/src/Help.tsx'),
      "const {t}=useTranslation('help'); t('modal.title'); t('modal.orphan');\n",
    );
    assert.deepEqual(checkHelp(root), []);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('tracks aliases, qualified calls and explicit model keys without counting other namespaces', () => {
  const source =
    "const {t:help}=useTranslation(['help','cards']); const {t}=useTranslation('settings'); help('modal.title'); t('help:thresholds.httpTotal'); t('settings.title'); log('help:unused'); const blocks=[{text:'content.about.description'}];";
  assert.deepEqual([...consumedHelpKeys(source, true)].sort(), [
    'content.about.description',
    'modal.title',
    'thresholds.httpTotal',
  ]);
});
