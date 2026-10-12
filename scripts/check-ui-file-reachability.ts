#!/usr/bin/env node
// Fails when a ui/src module is reachable from no entry point.
//
// The Go side has had this gate for a while (check-package-reachability.sh);
// the UI only had dead-code.yml's weekly advisory grep, so AppDashboard.tsx
// sat unimported for months and took NetworkDiscoveryCard's only non-Pro
// mount with it (#2707). A whole file nothing imports has few honest false
// positives, so this one blocks.
//
// The walk starts at the module scripts index.html loads and follows static
// imports, re-exports and literal dynamic imports, resolving `@/` the way
// vite.config.ts does. Tests, stories, declaration files and src/test/ are
// neither roots nor candidates: a component only a test mounts is unmounted.
// src/types/generated/ is followed but never reported: gen-types emits one
// file per wire schema whether or not the UI reads it yet.
//
// Modules already unreachable when the gate landed are listed in
// scripts/ui-file-reachability-baseline.txt with a reason. The list may shrink
// and must not grow, and an entry that is reachable again or deleted fails
// until it is removed, so the baseline cannot rot.
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from '../ui/node_modules/@babel/parser/lib/index.js';
import * as t from '../ui/node_modules/@babel/types/lib/index.js';

const SOURCE = /\.tsx?$/;
const EXCLUDED = /(\.(test|spec|stories)\.tsx?|\.d\.ts)$/;

export interface ModuleShape {
  imports: { specifier: string; typeOnly: boolean }[];
  runtimeExports: boolean;
}

const TYPE_DECLARATION = /^(TSInterfaceDeclaration|TSTypeAliasDeclaration|TSDeclareFunction)$/;

export function parseModule(source: string): ModuleShape {
  const ast = parse(source, { sourceType: 'module', plugins: ['typescript', 'jsx'] });
  const shape: ModuleShape = { imports: [], runtimeExports: false };
  const visit = (node: t.Node): void => {
    if (t.isImportDeclaration(node)) {
      const typeOnly =
        node.importKind === 'type' ||
        (node.specifiers.length > 0 &&
          node.specifiers.every((s) => t.isImportSpecifier(s) && s.importKind === 'type'));
      shape.imports.push({ specifier: node.source.value, typeOnly });
    } else if (t.isExportAllDeclaration(node) || t.isExportNamedDeclaration(node)) {
      const typeOnly =
        node.exportKind === 'type' ||
        (t.isExportNamedDeclaration(node) &&
          node.declaration != null &&
          TYPE_DECLARATION.test(node.declaration.type));
      if (!typeOnly) shape.runtimeExports = true;
      if (node.source) shape.imports.push({ specifier: node.source.value, typeOnly });
    } else if (t.isExportDefaultDeclaration(node)) {
      shape.runtimeExports = true;
    } else if (t.isCallExpression(node) && t.isImport(node.callee)) {
      const arg = node.arguments[0];
      // A computed import() would hide an edge from the walk and let a live
      // module look dead, so it is an error rather than a silent skip.
      if (!t.isStringLiteral(arg)) {
        throw new Error(
          `dynamic import() with a non-literal specifier (line ${node.loc?.start.line})`,
        );
      }
      shape.imports.push({ specifier: arg.value, typeOnly: false });
    }
    const fields = node as unknown as Record<string, unknown>;
    for (const key of t.VISITOR_KEYS[node.type] ?? []) {
      const value = fields[key];
      for (const child of Array.isArray(value) ? value : [value]) {
        if (child && typeof child === 'object' && 'type' in child) visit(child as t.Node);
      }
    }
  };
  visit(ast);
  return shape;
}

// Returns the module a specifier names, null for one outside ui/src (a package,
// @locales/, a stylesheet or asset), or throws when a local path resolves to
// nothing, which would otherwise drop an edge without anyone noticing.
function resolveSpecifier(uiRoot: string, from: string, specifier: string): string | null {
  let base: string;
  if (specifier.startsWith('@/')) base = join(uiRoot, 'src', specifier.slice(2));
  else if (specifier.startsWith('.')) base = resolve(dirname(from), specifier);
  else return null;
  const candidates = SOURCE.test(base)
    ? [base]
    : [`${base}.ts`, `${base}.tsx`, join(base, 'index.ts'), join(base, 'index.tsx')];
  const found = candidates.find((file) => existsSync(file) && statSync(file).isFile());
  if (found) return found;
  if (existsSync(base) && statSync(base).isFile()) return null;
  throw new Error(`${relative(uiRoot, from)}: cannot resolve '${specifier}'`);
}

function entryPoints(uiRoot: string): string[] {
  const html = readFileSync(join(uiRoot, 'index.html'), 'utf8');
  const entries = [...html.matchAll(/<script\b[^>]*\btype="module"[^>]*\bsrc="\/([^"]+)"/g)].map(
    (match) => join(uiRoot, match[1] ?? ''),
  );
  if (entries.length === 0) throw new Error('index.html loads no module script');
  return entries;
}

function candidates(dir: string, skip: string[]): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const file = join(dir, entry.name);
    if (entry.isDirectory()) return skip.includes(file) ? [] : candidates(file, skip);
    return SOURCE.test(file) && !EXCLUDED.test(file) ? [file] : [];
  });
}

// A module counts as reachable when the bundle loads it, through value
// imports from an entry point. A type-only import keeps a module alive only if
// it is all types: a component whose last mount went away is still dead when
// something imports its props type, and that is the #2707 shape exactly.
export function unreachableModules(uiRoot: string): string[] {
  const shapes = new Map<string, ModuleShape>();
  const shapeOf = (file: string): ModuleShape => {
    let shape = shapes.get(file);
    if (!shape) {
      try {
        shape = parseModule(readFileSync(file, 'utf8'));
      } catch (error) {
        throw new Error(`${relative(uiRoot, file)}: ${(error as Error).message}`);
      }
      shapes.set(file, shape);
    }
    return shape;
  };
  const walk = (roots: string[], follow: (typeOnly: boolean) => boolean): Set<string> => {
    const seen = new Set<string>();
    for (let file = roots.pop(); file !== undefined; file = roots.pop()) {
      if (seen.has(file)) continue;
      seen.add(file);
      for (const { specifier, typeOnly } of shapeOf(file).imports) {
        const target = follow(typeOnly) ? resolveSpecifier(uiRoot, file, specifier) : null;
        if (target && !seen.has(target)) roots.push(target);
      }
    }
    return seen;
  };
  const loaded = walk(entryPoints(uiRoot), (typeOnly) => !typeOnly);
  const typed = walk([...loaded], () => true);
  const src = join(uiRoot, 'src');
  return candidates(src, [join(src, 'test'), join(src, 'types/generated')])
    .filter((file) => !loaded.has(file) && (!typed.has(file) || shapeOf(file).runtimeExports))
    .map((file) => relative(uiRoot, file))
    .sort();
}

export function readBaseline(text: string): Set<string> {
  return new Set(
    text
      .split('\n')
      .map((line) => line.replace(/#.*/, '').trim())
      .filter(Boolean),
  );
}

export function compare(
  unreachable: string[],
  baseline: Set<string>,
): { added: string[]; stale: string[] } {
  const current = new Set(unreachable);
  return {
    added: unreachable.filter((file) => !baseline.has(file)),
    stale: [...baseline].filter((file) => !current.has(file)).sort(),
  };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const repo = dirname(dirname(fileURLToPath(import.meta.url)));
  const baselineFile = join(repo, 'scripts/ui-file-reachability-baseline.txt');
  const unreachable = unreachableModules(join(repo, 'ui'));
  const { added, stale } = compare(unreachable, readBaseline(readFileSync(baselineFile, 'utf8')));
  for (const file of added) {
    console.error(`::error file=ui/${file}::reachable from no entry point in ui/index.html`);
  }
  for (const file of stale) {
    console.error(`::error::${file} is baselined but no longer unreachable; delete its line`);
  }
  if (added.length > 0) {
    console.error(
      '\nNothing the app loads imports these modules, so they never reach a user. Mount them,\n' +
        'delete them, or record a reason in scripts/ui-file-reachability-baseline.txt.',
    );
  }
  if (added.length > 0 || stale.length > 0) process.exitCode = 1;
  else console.log(`✓ no new unreachable UI modules (${unreachable.length} baselined)`);
}
