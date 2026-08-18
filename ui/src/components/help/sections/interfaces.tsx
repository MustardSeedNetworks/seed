/**
 * interfaces.tsx — The local interface: cabling, link state, and wireless attachment.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { Activity, Cable, Wifi } from '../../ui/icons';
import type { HelpSection } from '../helpModel';

const ICON = 'w-4 h-4';

export const interfaceSections: HelpSection[] = [
  {
    id: 'link',
    titleKey: 'sections.link',
    icon: <Activity className={ICON} />,
    keywords: ['link', 'carrier', 'speed', 'duplex', 'mtu', 'physical layer'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Link Status monitors the physical-layer connection of the selected network interface.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Carrier',
            description:
              "Physical-layer signal detection. Shows 'Connected' when the NIC detects a link partner (a cable into an active port).",
          },
          {
            term: 'Speed',
            description:
              'Negotiated link speed between your interface and the connected device (for example 1000 Mbps).',
          },
          {
            term: 'Duplex',
            description:
              'Communication mode — full duplex allows simultaneous bidirectional data; half duplex is one direction at a time.',
          },
          {
            term: 'Auto-Negotiation',
            description:
              'Whether speed and duplex were negotiated automatically with the link partner or set manually.',
          },
          {
            term: 'MTU',
            description:
              'Maximum Transmission Unit — the largest packet size (in bytes) that can be sent without fragmentation. Standard is 1500 bytes.',
          },
        ],
      },
    ],
  },
  {
    id: 'cable',
    titleKey: 'sections.cable',
    icon: <Cable className={ICON} />,
    keywords: ['cable', 'tdr', 'fault', 'pairs', 'open', 'short'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'The Cable Test uses Time Domain Reflectometry (TDR) to check cable quality and locate faults.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'TDR test',
            description:
              'Sends electrical pulses down the cable and measures reflections to detect faults and estimate length.',
          },
          {
            term: 'Cable status',
            description:
              'Reports whether each pair is OK, open (disconnected), short (wires touching), or has an impedance mismatch.',
          },
          {
            term: 'Fault distance',
            description:
              'Distance to a detected fault in meters, to help locate the physical problem.',
          },
          {
            term: 'Pairs',
            description:
              'Ethernet cables have four twisted pairs. Gigabit uses all four; Fast Ethernet uses pairs 1-2 and 3-6.',
          },
        ],
      },
      {
        kind: 'note',
        text: 'Cable testing requires compatible network hardware. Not all NICs support TDR.',
      },
    ],
  },
  {
    id: 'wifi',
    titleKey: 'sections.wifi',
    icon: <Wifi className={ICON} />,
    keywords: ['wifi', 'wireless', 'ssid', 'bssid', 'signal', 'channel', 'canopy'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Wi-Fi Status monitors the quality and settings of the current wireless connection — Wi-Fi visibility and troubleshooting.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'SSID',
            description:
              "Service Set Identifier — the name of the wireless network you're connected to.",
          },
          {
            term: 'BSSID',
            description: 'Basic Service Set Identifier — the MAC address of the access point.',
          },
          {
            term: 'Signal strength',
            description:
              'Signal level in dBm. -30 is excellent, -67 is good, -70 is fair, -80 is weak. Higher (less negative) is better.',
          },
          {
            term: 'Channel',
            description:
              'Wi-Fi channel number (1-14 for 2.4 GHz, 36-165 for 5 GHz). Overlapping channels cause interference.',
          },
          {
            term: 'Security',
            description:
              'Encryption protocol protecting the connection (WPA2, WPA3, WEP, or Open).',
          },
          {
            term: 'Frequency',
            description:
              'Radio band — 2.4 GHz has better range; 5 GHz offers higher speeds and less interference.',
          },
        ],
      },
    ],
  },
];
