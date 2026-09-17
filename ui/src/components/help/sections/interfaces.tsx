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

import { Activity, Cable, Signal, Wifi } from '../../ui/Icons';
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
        text: 'content.linkStatus.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.linkStatus.terms.carrier.term',
            description: 'content.linkStatus.terms.carrier.description',
          },
          {
            term: 'content.linkStatus.terms.speed.term',
            description: 'content.linkStatus.terms.speed.description',
          },
          {
            term: 'content.linkStatus.terms.duplex.term',
            description: 'content.linkStatus.terms.duplex.description',
          },
          {
            term: 'content.linkStatus.terms.autoNeg.term',
            description: 'content.linkStatus.terms.autoNeg.description',
          },
          {
            term: 'content.linkStatus.terms.mtu.term',
            description: 'content.linkStatus.terms.mtu.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:link.title',
            description: 'content.cardHelp.LinkCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:cable.title',
            description: 'content.cardHelp.CableCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:driverStats.title',
            description: 'content.cardHelp.DriverStatsCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:wifi.title',
            description: 'content.cardHelp.WiFiCard.description',
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
        text: 'content.cableTest.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.cableTest.terms.tdrTest.term',
            description: 'content.cableTest.terms.tdrTest.description',
          },
          {
            term: 'content.cableTest.terms.cableStatus.term',
            description: 'content.cableTest.terms.cableStatus.description',
          },
          {
            term: 'content.cableTest.terms.faultDistance.term',
            description: 'content.cableTest.terms.faultDistance.description',
          },
          {
            term: 'content.cableTest.terms.pairs.term',
            description: 'content.cableTest.terms.pairs.description',
          },
        ],
      },
      {
        kind: 'note',
        text: 'content.cableTest.note',
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
        text: 'content.wifiStatus.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.wifiStatus.terms.ssid.term',
            description: 'content.wifiStatus.terms.ssid.description',
          },
          {
            term: 'content.wifiStatus.terms.bssid.term',
            description: 'content.wifiStatus.terms.bssid.description',
          },
          {
            term: 'content.wifiStatus.terms.signal.term',
            description: 'content.wifiStatus.terms.signal.description',
          },
          {
            term: 'content.wifiStatus.terms.channel.term',
            description: 'content.wifiStatus.terms.channel.description',
          },
          {
            term: 'content.wifiStatus.terms.security.term',
            description: 'content.wifiStatus.terms.security.description',
          },
          {
            term: 'content.wifiStatus.terms.frequency.term',
            description: 'content.wifiStatus.terms.frequency.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:wifi.title',
            description: 'content.cardHelp.WiFiCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:wifi.channelGraph.title',
            description: 'content.cardHelp.WiFiChannelGraph.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'pages:wifi.airspaceTitle',
            description: 'content.cardHelp.WiFiAirspaceCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'pages:wifi.anomaliesTitle',
            description: 'content.cardHelp.WiFiAnomaliesCard.description',
          },
        ],
      },
    ],
  },
  {
    id: 'wifiTroubleshooting',
    titleKey: 'sections.wifiTroubleshooting',
    icon: <Signal className={ICON} />,
    keywords: ['wifi troubleshooting', 'channel utilization', 'neighbor', 'snr', 'canopy'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.wifiTroubleshooting.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.wifiTroubleshooting.terms.neighbors.term',
            description: 'content.wifiTroubleshooting.terms.neighbors.description',
          },
          {
            term: 'content.wifiTroubleshooting.terms.utilization.term',
            description: 'content.wifiTroubleshooting.terms.utilization.description',
          },
          {
            term: 'content.wifiTroubleshooting.terms.interference.term',
            description: 'content.wifiTroubleshooting.terms.interference.description',
          },
          {
            term: 'content.wifiTroubleshooting.terms.snr.term',
            description: 'content.wifiTroubleshooting.terms.snr.description',
          },
        ],
      },
    ],
  },
];
