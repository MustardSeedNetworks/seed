import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { test } from 'node:test';
import {
  compare,
  parseModule,
  readBaseline,
  unreachableModules,
} from './check-ui-file-reachability.ts';

function tree(files: Record<string, string>): string {
  const root = mkdtempSync(join(tmpdir(), 'seed-ui-reach-'));
  for (const [path, body] of Object.entries(files)) {
    mkdirSync(dirname(join(root, path)), { recursive: true });
    writeFileSync(join(root, path), body);
  }
  return root;
}

const app = {
  'index.html': '<script type="module" src="/src/main.tsx"></script>',
  'src/main.tsx': "import { App } from './App';\nimport './index.css';\nApp();\n",
  'src/index.css': '',
  'src/App.tsx':
    "import { Card } from '@/components/Card';\nimport type { T } from './types';\n" +
    "export { util } from './lib';\nconst Page = () => import('./pages/Page');\n" +
    'export const App = (): T => [Card, Page];\n',
  'src/types.ts': 'export type T = unknown;\n',
  'src/lib/index.ts': 'export const util = 1;\n',
  'src/pages/Page.tsx': 'export default 1;\n',
  'src/components/Card.tsx': 'export const Card = 1;\n',
  'src/components/Card.test.tsx': "import { Orphan } from './Orphan';\n",
  'src/components/Card.stories.tsx': "import { Orphan } from './Orphan';\n",
  'src/components/Orphan.tsx': 'export const Orphan = 1;\n',
  'src/test/setup.ts': '',
  'src/types/generated/wire.ts': 'export type W = 1;\n',
  'src/vite-env.d.ts': '',
};

test('a module only tests and stories import is unreachable; every import form is an edge', () => {
  const root = tree(app);
  try {
    assert.deepEqual(unreachableModules(root), ['src/components/Orphan.tsx']);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('deleting the last import of a component reports it', () => {
  const root = tree({
    ...app,
    'src/App.tsx': "const Page = () => import('./pages/Page');\nexport const App = () => Page;\n",
  });
  try {
    assert.deepEqual(unreachableModules(root), [
      'src/components/Card.tsx',
      'src/components/Orphan.tsx',
      'src/lib/index.ts',
      'src/types.ts',
    ]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('a component imported only for its types is unreachable; a pure type module is not', () => {
  const root = tree({
    ...app,
    'src/App.tsx':
      "import type { CardProps } from '@/components/Card';\nimport type { T } from './types';\n" +
      'export const App = (p: CardProps): T => p;\n',
    'src/components/Card.tsx':
      'export interface CardProps { id: string }\nexport const Card = 1;\n',
  });
  try {
    assert.deepEqual(unreachableModules(root), [
      'src/components/Card.tsx',
      'src/components/Orphan.tsx',
      'src/lib/index.ts',
      'src/pages/Page.tsx',
    ]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('an unresolvable local import fails rather than dropping the edge', () => {
  const root = tree({ ...app, 'src/main.tsx': "import { App } from './Missing';\n" });
  try {
    assert.throws(() => unreachableModules(root), /cannot resolve '\.\/Missing'/);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('a computed dynamic import fails rather than hiding an edge', () => {
  assert.throws(() => parseModule('const m = (n) => import(`./pages/${n}`);'), /non-literal/);
});

test('the baseline ratchets: new orphans and stale entries both fail', () => {
  const baseline = readBaseline('# header\nsrc/a.tsx   # reason\nsrc/gone.tsx\n\n');
  assert.deepEqual(compare(['src/a.tsx', 'src/b.tsx'], baseline), {
    added: ['src/b.tsx'],
    stale: ['src/gone.tsx'],
  });
});
