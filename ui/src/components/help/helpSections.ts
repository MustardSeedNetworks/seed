/**
 * helpSections.ts — the drawer's table of contents, in reading order.
 *
 * Composed from the content modules in `sections/`. Order here is the order
 * the reader sees: orientation first, then diagnostics by layer, then the
 * continuous checks, guides, reporting surfaces, and finally reference.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import type { HelpSection } from './helpModel';
import { guidesSections } from './sections/guides';
import { interfaceSections } from './sections/interfaces';
import { monitoringSections } from './sections/monitoring';
import { overviewSections } from './sections/overview';
import { referenceSections } from './sections/reference';
import { reportingSections } from './sections/reporting';
import { upstreamSections } from './sections/upstream';

export const helpSections: HelpSection[] = [
  ...overviewSections,
  ...interfaceSections,
  ...upstreamSections,
  ...monitoringSections,
  ...guidesSections,
  ...reportingSections,
  ...referenceSections,
];
