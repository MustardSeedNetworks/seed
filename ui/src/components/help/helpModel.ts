/**
 * helpModel.ts — the typed content model behind the HelpDrawer.
 *
 * Section bodies are authored as a sequence of typed `HelpBlock`s so a single
 * generic renderer (`HelpSectionBody`) can present every section without
 * bespoke per-section JSX. The content itself lives in `sections/`.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import type { ReactNode } from 'react';
import type { HelpTranslations } from '../../i18n/types';
/** A labelled term + its explanation (definition lists, metric glossaries). */
export interface HelpTerm {
  term: string;
  description: string;
}

/** An ordered step within a how-to / configuration walkthrough. */
export interface HelpStep {
  title?: string;
  description: string;
}

/**
 * A single renderable unit within a section body. The renderer
 * (`HelpSectionBody`) switches on `kind`.
 */
export type HelpBlock =
  | { kind: 'paragraph'; text: string }
  | { kind: 'heading'; text: string }
  | { kind: 'terms'; heading?: string; items: HelpTerm[] }
  | { kind: 'steps'; heading?: string; ordered?: boolean; items: HelpStep[] }
  | { kind: 'tips'; heading?: string; items: string[] }
  | { kind: 'note'; text: string };

/** Fully-qualified `help` namespace key for a section title (e.g. `sections.about`). */
export type HelpSectionTitleKey = `sections.${keyof HelpTranslations['sections']}`;

/** A top-level help section, shown as a TOC entry + a content pane. */
export interface HelpSection {
  /** Stable id — matches the legacy modal section ids + `sections.*` i18n keys. */
  id: string;
  /** i18n key under the `help` namespace for the human title. */
  titleKey: HelpSectionTitleKey;
  icon: ReactNode;
  /** Extra search keywords beyond the title + rendered body text. */
  keywords: string[];
  blocks: HelpBlock[];
}

/**
 * Flatten a section's blocks into a single lowercase string for search
 * matching (titles are matched separately by the drawer via i18n).
 */
export function sectionSearchText(section: HelpSection): string {
  const parts: string[] = [...section.keywords];
  for (const block of section.blocks) {
    switch (block.kind) {
      case 'paragraph':
      case 'heading':
      case 'note':
        parts.push(block.text);
        break;
      case 'terms':
        if (block.heading) {
          parts.push(block.heading);
        }
        for (const item of block.items) {
          parts.push(item.term, item.description);
        }
        break;
      case 'steps':
        if (block.heading) {
          parts.push(block.heading);
        }
        for (const item of block.items) {
          if (item.title) {
            parts.push(item.title);
          }
          parts.push(item.description);
        }
        break;
      case 'tips':
        if (block.heading) {
          parts.push(block.heading);
        }
        parts.push(...block.items);
        break;
    }
  }
  return parts.join(' ').toLowerCase();
}
