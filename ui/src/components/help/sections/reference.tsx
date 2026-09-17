/**
 * reference.tsx — Terminology used across the product.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { BookOpen } from '../../ui/Icons';
import type { HelpSection } from '../helpModel';

const ICON = 'w-4 h-4';

export const referenceSections: HelpSection[] = [
  {
    id: 'glossary',
    titleKey: 'sections.glossary',
    icon: <BookOpen className={ICON} />,
    keywords: ['glossary', 'terms', 'definitions', 'acronyms'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.glossary.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.glossary.terms.arp.term',
            description: 'content.glossary.terms.arp.description',
          },
          {
            term: 'content.glossary.terms.dhcp.term',
            description: 'content.glossary.terms.dhcp.description',
          },
          {
            term: 'content.glossary.terms.dns.term',
            description: 'content.glossary.terms.dns.description',
          },
          {
            term: 'content.glossary.terms.iperf.term',
            description: 'content.glossary.terms.iperf.description',
          },
          {
            term: 'content.glossary.terms.lldp.term',
            description: 'content.glossary.terms.lldp.description',
          },
          {
            term: 'content.glossary.terms.rtsp.term',
            description: 'content.glossary.terms.rtsp.description',
          },
          {
            term: 'content.glossary.terms.snr.term',
            description: 'content.glossary.terms.snr.description',
          },
          {
            term: 'content.glossary.terms.tdr.term',
            description: 'content.glossary.terms.tdr.description',
          },
        ],
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.wifiStatus.terms.bssid.term',
            description: 'content.wifiStatus.terms.bssid.description',
          },
          {
            term: 'content.wifiStatus.terms.ssid.term',
            description: 'content.wifiStatus.terms.ssid.description',
          },
        ],
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.linkStatus.terms.duplex.term',
            description: 'content.linkStatus.terms.duplex.description',
          },
          {
            term: 'content.linkStatus.terms.mtu.term',
            description: 'content.linkStatus.terms.mtu.description',
          },
        ],
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.dicomChecks.terms.cEcho.term',
            description: 'content.dicomChecks.terms.cEcho.description',
          },
        ],
      },
    ],
  },
];
