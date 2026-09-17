/**
 * guides.tsx — Task-shaped walkthroughs rather than screen reference.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { AlertTriangle, Lightbulb } from '../../ui/Icons';
import type { HelpSection } from '../helpModel';

const ICON = 'w-4 h-4';

export const guidesSections: HelpSection[] = [
  {
    id: 'troubleshooting',
    titleKey: 'sections.troubleshooting',
    icon: <AlertTriangle className={ICON} />,
    keywords: ['troubleshooting', 'no carrier', 'slow', 'open', 'short', 'unreachable'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.troubleshooting.description',
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'content.troubleshooting.categories.linkIssues.noCarrier.symptom',
            description: 'content.troubleshooting.categories.linkIssues.noCarrier.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'content.troubleshooting.categories.linkIssues.slowSpeed.symptom',
            description: 'content.troubleshooting.categories.linkIssues.slowSpeed.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'content.troubleshooting.categories.cableIssues.open.symptom',
            description: 'content.troubleshooting.categories.cableIssues.open.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'content.troubleshooting.categories.cableIssues.short.symptom',
            description: 'content.troubleshooting.categories.cableIssues.short.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'content.troubleshooting.categories.gatewayIssues.unreachable.symptom',
            description: 'content.troubleshooting.categories.gatewayIssues.unreachable.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'content.troubleshooting.categories.performanceIssues.slowInternet.symptom',
            description:
              'content.troubleshooting.categories.performanceIssues.slowInternet.description',
          },
        ],
      },
    ],
  },
  {
    id: 'howTo',
    titleKey: 'sections.howTo',
    icon: <Lightbulb className={ICON} />,
    keywords: ['how to', 'guide', 'diagnose', 'health checks', 'walkthrough'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.howTo.description',
      },
      {
        kind: 'steps',
        heading: 'content.howTo.guides.diagnoseSlowNetwork.title',
        ordered: true,
        items: [
          {
            title: 'content.howTo.guides.diagnoseSlowNetwork.steps.0.step',
            description: 'content.howTo.guides.diagnoseSlowNetwork.steps.0.description',
          },
          {
            title: 'content.howTo.guides.diagnoseSlowNetwork.steps.1.step',
            description: 'content.howTo.guides.diagnoseSlowNetwork.steps.1.description',
          },
          {
            title: 'content.howTo.guides.diagnoseSlowNetwork.steps.2.step',
            description: 'content.howTo.guides.diagnoseSlowNetwork.steps.2.description',
          },
          {
            title: 'content.howTo.guides.diagnoseSlowNetwork.steps.3.step',
            description: 'content.howTo.guides.diagnoseSlowNetwork.steps.3.description',
          },
          {
            title: 'content.howTo.guides.diagnoseSlowNetwork.steps.4.step',
            description: 'content.howTo.guides.diagnoseSlowNetwork.steps.4.description',
          },
        ],
      },
      {
        kind: 'steps',
        heading: 'content.howTo.guides.setupHealthChecks.title',
        ordered: true,
        items: [
          {
            title: 'content.howTo.guides.setupHealthChecks.steps.0.step',
            description: 'content.howTo.guides.setupHealthChecks.steps.0.description',
          },
          {
            title: 'content.howTo.guides.setupHealthChecks.steps.1.step',
            description: 'content.howTo.guides.setupHealthChecks.steps.1.description',
          },
          {
            title: 'content.howTo.guides.setupHealthChecks.steps.2.step',
            description: 'content.howTo.guides.setupHealthChecks.steps.2.description',
          },
          {
            title: 'content.howTo.guides.setupHealthChecks.steps.3.step',
            description: 'content.howTo.guides.setupHealthChecks.steps.3.description',
          },
          {
            title: 'content.howTo.guides.setupHealthChecks.steps.4.step',
            description: 'content.howTo.guides.setupHealthChecks.steps.4.description',
          },
          {
            title: 'content.howTo.guides.setupHealthChecks.steps.5.step',
            description: 'content.howTo.guides.setupHealthChecks.steps.5.description',
          },
        ],
      },
    ],
  },
];
