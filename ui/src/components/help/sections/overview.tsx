/**
 * overview.tsx — Orientation — what Seed is and how to get moving.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { Info, LayoutDashboard, SlidersHorizontal } from '../../ui/Icons';
import type { HelpSection } from '../helpModel';

const ICON = 'w-4 h-4';

export const overviewSections: HelpSection[] = [
  {
    id: 'about',
    titleKey: 'sections.about',
    icon: <Info className={ICON} />,
    keywords: ['about', 'overview', 'live telemetry', 'diagnostics', 'monitoring', 'reporting'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.about.description',
      },
      {
        kind: 'paragraph',
        text: 'content.about.openSource.description',
      },
    ],
  },
  {
    id: 'gettingStarted',
    titleKey: 'sections.gettingStarted',
    icon: <LayoutDashboard className={ICON} />,
    keywords: ['getting started', 'dashboard', 'interface', 'run tests', 'cards'],
    blocks: [
      {
        kind: 'steps',
        ordered: true,
        items: [
          {
            title: 'content.gettingStarted.steps.interface.title',
            description: 'content.gettingStarted.steps.interface.description',
          },
          {
            title: 'content.gettingStarted.steps.dashboard.title',
            description: 'content.gettingStarted.steps.dashboard.description',
          },
          {
            title: 'content.gettingStarted.steps.thresholds.title',
            description: 'content.gettingStarted.steps.thresholds.description',
          },
          {
            title: 'content.gettingStarted.steps.runTests.title',
            description: 'content.gettingStarted.steps.runTests.description',
          },
        ],
      },
      {
        kind: 'tips',
        heading: 'content.gettingStarted.proTips.title',
        items: ['content.gettingStarted.proTips.tips.0', 'content.gettingStarted.proTips.tips.2'],
      },
    ],
  },
  {
    id: 'profiles',
    titleKey: 'sections.profiles',
    icon: <SlidersHorizontal className={ICON} />,
    keywords: ['profiles', 'configuration', 'export', 'import', 'msp', 'sites'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.profiles.description',
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'content.profiles.features.clientSpecific.title',
            description: 'content.profiles.features.clientSpecific.description',
          },
          {
            term: 'content.profiles.features.quickSwitch.title',
            description: 'content.profiles.features.quickSwitch.description',
          },
          {
            term: 'content.profiles.features.exportImport.title',
            description: 'content.profiles.features.exportImport.description',
          },
          {
            term: 'content.profiles.features.defaultProfile.title',
            description: 'content.profiles.features.defaultProfile.description',
          },
        ],
      },
      {
        kind: 'tips',
        heading: 'content.profiles.bestPractices.title',
        items: [
          'content.profiles.bestPractices.tips.0',
          'content.profiles.bestPractices.tips.1',
          'content.profiles.bestPractices.tips.2',
        ],
      },
    ],
  },
];
