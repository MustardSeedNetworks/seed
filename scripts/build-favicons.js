#!/usr/bin/env node
/**
 * build-favicons.js (ESM)
 *
 * Generates the favicon raster set (PNG + ICO) from `ui/public/favicon.svg`.
 *
 * Outputs into `ui/public/`:
 *   favicon-16x16.png
 *   favicon-32x32.png
 *   apple-touch-icon.png       (180x180)
 *   android-chrome-192x192.png
 *   android-chrome-512x512.png
 *   favicon.ico                (16+32+48 multi-resolution)
 *
 * Dependencies: `sharp` and `png-to-ico` (installed transiently if absent).
 * Run from repo root:  node scripts/build-favicons.js
 */

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const repoRoot = path.resolve(__dirname, '..');
const publicDir = path.join(repoRoot, 'ui', 'public');
const svgPath = path.join(publicDir, 'favicon.svg');

if (!fs.existsSync(svgPath)) {
  console.error(`favicon.svg not found at ${svgPath}`);
  process.exit(1);
}

async function tryImport(mod) {
  try {
    return (await import(mod)).default;
  } catch {
    return null;
  }
}

let sharp = await tryImport('sharp');
let pngToIco = await tryImport('png-to-ico');

if (!sharp || !pngToIco) {
  console.log('Installing transient build deps (sharp, png-to-ico)...');
  execFileSync(
    'npm',
    ['install', '--no-save', '--no-package-lock', 'sharp@0.34.5', 'png-to-ico@3.0.1'],
    { cwd: repoRoot, stdio: 'inherit' },
  );
  sharp = (await import('sharp')).default;
  pngToIco = (await import('png-to-ico')).default;
}

const svgBuffer = fs.readFileSync(svgPath);

const pngTargets = [
  { name: 'favicon-16x16.png', size: 16 },
  { name: 'favicon-32x32.png', size: 32 },
  { name: 'apple-touch-icon.png', size: 180 },
  { name: 'android-chrome-192x192.png', size: 192 },
  { name: 'android-chrome-512x512.png', size: 512 },
];

for (const target of pngTargets) {
  const outPath = path.join(publicDir, target.name);
  await sharp(svgBuffer, { density: 384 })
    .resize(target.size, target.size, {
      fit: 'contain',
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    })
    .png({ compressionLevel: 9 })
    .toFile(outPath);
  console.log(`wrote ${target.name} (${target.size}x${target.size})`);
}

// ICO from 16, 32, 48 PNGs (48 generated in-memory).
const ico16 = await sharp(svgBuffer, { density: 384 }).resize(16, 16).png().toBuffer();
const ico32 = await sharp(svgBuffer, { density: 384 }).resize(32, 32).png().toBuffer();
const ico48 = await sharp(svgBuffer, { density: 384 }).resize(48, 48).png().toBuffer();
const icoBuf = await pngToIco([ico16, ico32, ico48]);
fs.writeFileSync(path.join(publicDir, 'favicon.ico'), icoBuf);
console.log('wrote favicon.ico (16+32+48)');
