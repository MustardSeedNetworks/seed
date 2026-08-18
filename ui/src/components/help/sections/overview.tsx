/**
 * overview.tsx — Orientation — what The Seed is and how to get moving.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { Info, LayoutDashboard, SlidersHorizontal } from '../../ui/icons';
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
        text: 'The Seed is a network diagnostics and monitoring tool by Mustard Seed Networks. It gives you visibility into physical-layer link state, IP configuration, gateway and DNS reachability, device discovery, throughput, Wi-Fi connection quality, and endpoint health from a single dashboard.',
      },
      {
        kind: 'note',
        text: 'The Seed is source-available software (BUSL-1.1). The version, backend commit, and build time are shown in the drawer header and at the /__version endpoint.',
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
        kind: 'paragraph',
        text: 'The dashboard shows a card for each diagnostic area. Each card displays live information about one aspect of your network and can be opened for detail.',
      },
      {
        kind: 'steps',
        heading: 'First steps',
        ordered: true,
        items: [
          {
            title: 'Select a network interface',
            description:
              'Use the interface selector in the header to choose which interface to monitor (for example eth0 or wlan0).',
          },
          {
            title: 'Review the dashboard',
            description:
              'Each card reflects the selected interface. Cards update as the underlying tests run.',
          },
          {
            title: 'Configure thresholds',
            description:
              'Open Settings to set warning and critical levels for metrics such as DNS latency, gateway ping, and Wi-Fi signal strength.',
          },
          {
            title: 'Run tests',
            description:
              'Use the Run All Tests action to execute speed tests, discovery, and health checks together, or run an individual test from its card.',
          },
        ],
      },
      {
        kind: 'tips',
        heading: 'Tips',
        items: [
          'Use Network Discovery to find every device on the local subnet.',
          'Save per-site configuration as a Profile and switch between profiles from the header.',
          'Export diagnostics for documentation or troubleshooting handoff.',
        ],
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
        text: 'Profiles are saved configuration sets — thresholds, health-check targets, discovery settings, and interface preferences — so you can switch between clients, sites, or test scenarios without reconfiguring.',
      },
      {
        kind: 'terms',
        heading: 'What profiles store',
        items: [
          {
            term: 'Site-specific settings',
            description:
              'Each profile can carry its own thresholds, health-check targets, and discovery settings tailored to that environment.',
          },
          {
            term: 'Quick switching',
            description:
              'Switch profiles from the header; settings apply immediately without restarting the app.',
          },
          {
            term: 'Export & import',
            description:
              'Export a profile as JSON to back it up or move it to another Seed installation, and import profiles from elsewhere.',
          },
          {
            term: 'Default profile',
            description:
              'One profile can be marked as the default and loaded automatically on startup.',
          },
        ],
      },
      {
        kind: 'tips',
        heading: 'Best practices',
        items: [
          'Use descriptive profile names that identify the site or client and location.',
          'Keep a baseline default profile with your standard settings.',
          'Export a profile before making major changes so you have a backup.',
        ],
      },
    ],
  },
];
