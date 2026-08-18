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

import { BookOpen } from '../../ui/icons';
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
        kind: 'terms',
        heading: 'Common terms',
        items: [
          {
            term: 'ARP',
            description:
              'Address Resolution Protocol — maps IP addresses to MAC addresses on a local network.',
          },
          { term: 'BSSID', description: 'MAC address of a Wi-Fi access point.' },
          {
            term: 'C-ECHO',
            description:
              'DICOM verification service used to confirm a remote imaging node is reachable.',
          },
          {
            term: 'DHCP',
            description:
              'Dynamic Host Configuration Protocol — assigns IP configuration to devices automatically.',
          },
          {
            term: 'DNS',
            description: 'Domain Name System — resolves hostnames to IP addresses and back.',
          },
          {
            term: 'Duplex',
            description:
              'Whether a link can send and receive at the same time (full) or one direction at a time (half).',
          },
          {
            term: 'iperf3',
            description:
              'A tool for measuring achievable throughput between two hosts on a network.',
          },
          {
            term: 'LLDP / CDP',
            description:
              'Discovery protocols that reveal details about directly connected switches.',
          },
          {
            term: 'MTU',
            description:
              'Maximum Transmission Unit — the largest packet that can be sent without fragmentation.',
          },
          {
            term: 'RTSP',
            description: 'Real Time Streaming Protocol — used by IP cameras and video systems.',
          },
          {
            term: 'SNR',
            description:
              'Signal-to-noise ratio — how far a Wi-Fi signal rises above background noise.',
          },
          { term: 'SSID', description: 'The name of a wireless network.' },
          {
            term: 'TDR',
            description:
              'Time Domain Reflectometry — measures cable length and locates faults using signal reflections.',
          },
        ],
      },
    ],
  },
];
