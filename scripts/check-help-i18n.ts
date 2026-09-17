#!/usr/bin/env node
import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from '../ui/node_modules/@babel/parser/lib/index.js';
import * as t from '../ui/node_modules/@babel/types/lib/index.js';

export function leafKeys(value: unknown, prefix = ''): string[] {
  if (typeof value === 'string') return [prefix];
  if (!value || typeof value !== 'object') return [];
  return Object.entries(value).flatMap(([key, child]) =>
    leafKeys(child, prefix ? `${prefix}.${key}` : key),
  );
}

export function consumedHelpKeys(source: string, model = false): Set<string> {
  const ast = parse(source, { sourceType: 'module', plugins: ['typescript', 'jsx'] });
  const keys = new Set<string>();
  const translators = new Set<string>();
  const helpTranslators = new Set<string>();
  const visit = (node: t.Node, inspect: (node: t.Node) => void): void => {
    inspect(node);
    const fields = node as unknown as Record<string, unknown>;
    for (const key of t.VISITOR_KEYS[node.type] ?? []) {
      const value = fields[key];
      for (const child of Array.isArray(value) ? value : [value]) {
        if (child && typeof child === 'object' && 'type' in child) visit(child as t.Node, inspect);
      }
    }
  };
  visit(ast, (node) => {
    if (
      !t.isVariableDeclarator(node) ||
      !t.isObjectPattern(node.id) ||
      !t.isCallExpression(node.init) ||
      !t.isIdentifier(node.init.callee, { name: 'useTranslation' })
    )
      return;
    const ns = node.init.arguments[0];
    const first = t.isArrayExpression(ns) ? ns.elements[0] : ns;
    for (const property of node.id.properties) {
      if (
        t.isObjectProperty(property) &&
        t.isIdentifier(property.key, { name: 't' }) &&
        t.isIdentifier(property.value)
      ) {
        translators.add(property.value.name);
        if (t.isStringLiteral(first, { value: 'help' })) helpTranslators.add(property.value.name);
      }
    }
  });
  visit(ast, (node) => {
    if (
      t.isCallExpression(node) &&
      t.isIdentifier(node.callee) &&
      translators.has(node.callee.name)
    ) {
      const arg = node.arguments[0];
      if (t.isStringLiteral(arg)) {
        if (arg.value.startsWith('help:')) keys.add(arg.value.slice(5));
        else if (helpTranslators.has(node.callee.name)) keys.add(arg.value);
      }
    }
    // The renderer consumes these explicit keys from the typed HelpSection model.
    if (model && t.isStringLiteral(node) && /^(content|sections)\./.test(node.value))
      keys.add(node.value);
  });
  return keys;
}

export function checkHelp(root: string): string[] {
  const used = new Set<string>();
  const scan = (dir: string): void => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const file = join(dir, entry.name);
      if (entry.isDirectory()) scan(file);
      else if (
        /\.tsx?$/.test(file) &&
        !/\.(test|stories)\.tsx?$/.test(file) &&
        !file.includes('/test/')
      ) {
        const model = relative(root, file).startsWith('ui/src/components/help/sections/');
        for (const key of consumedHelpKeys(readFileSync(file, 'utf8'), model)) used.add(key);
      }
    }
  };
  scan(join(root, 'ui/src'));
  const locale: unknown = JSON.parse(
    readFileSync(join(root, 'internal/i18n/locales/en/help.json'), 'utf8'),
  );
  return leafKeys(locale)
    .filter((key) => !used.has(key))
    .sort();
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const root = process.argv[2] ?? dirname(dirname(fileURLToPath(import.meta.url)));
  const orphaned = checkHelp(root);
  if (orphaned.length) {
    process.stderr.write(`Help keys without production consumers:\n${orphaned.join('\n')}\n`);
    process.exitCode = 1;
  } else process.stdout.write('Help locale: every key has a production consumer.\n');
}
