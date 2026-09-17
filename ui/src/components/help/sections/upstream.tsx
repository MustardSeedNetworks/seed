/**
 * upstream.tsx — Everything past the interface: addressing, reachability, throughput, neighbours.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { Network, Search, Server, Zap } from '../../ui/icons';
import type { HelpSection } from '../helpModel';

const ICON = 'w-4 h-4';

export const upstreamSections: HelpSection[] = [
  {
    id: 'network',
    titleKey: 'sections.network',
    icon: <Network className={ICON} />,
    keywords: ['network', 'dhcp', 'lease', 'ip', 'subnet', 'gateway', 'vlan', 'upstream'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.networkDhcp.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.networkDhcp.terms.leaseTime.term',
            description: 'content.networkDhcp.terms.leaseTime.description',
          },
          {
            term: 'content.networkDhcp.terms.dhcpServer.term',
            description: 'content.networkDhcp.terms.dhcpServer.description',
          },
          {
            term: 'content.networkDhcp.terms.gateway.term',
            description: 'content.networkDhcp.terms.gateway.description',
          },
          {
            term: 'content.networkDhcp.terms.dnsServers.term',
            description: 'content.networkDhcp.terms.dnsServers.description',
          },
          {
            term: 'content.networkDhcp.terms.subnetMask.term',
            description: 'content.networkDhcp.terms.subnetMask.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:network.title',
            description: 'content.cardHelp.NetworkCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:gateway.title',
            description: 'content.cardHelp.GatewayCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:dns.title',
            description: 'content.cardHelp.DnsCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:neighbours.title',
            description: 'content.cardHelp.NeighbourCacheCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:bonjour.title',
            description: 'content.cardHelp.BonjourCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:publicIp.title',
            description: 'content.cardHelp.PublicIpCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:switch.title',
            description: 'content.cardHelp.SwitchCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:discovery.title',
            description: 'content.cardHelp.NetworkDiscoveryCard.description',
          },
        ],
      },
    ],
  },
  {
    id: 'gateway',
    titleKey: 'sections.gateway',
    icon: <Server className={ICON} />,
    keywords: ['gateway', 'router', 'reachability', 'latency', 'packet loss'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.gatewayHelp.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.gatewayHelp.terms.ipv4Gateway.term',
            description: 'content.gatewayHelp.terms.ipv4Gateway.description',
          },
          {
            term: 'content.gatewayHelp.terms.ipv6Gateway.term',
            description: 'content.gatewayHelp.terms.ipv6Gateway.description',
          },
          {
            term: 'content.gatewayHelp.terms.reachability.term',
            description: 'content.gatewayHelp.terms.reachability.description',
          },
          {
            term: 'content.gatewayHelp.terms.latency.term',
            description: 'content.gatewayHelp.terms.latency.description',
          },
          {
            term: 'content.gatewayHelp.terms.packetLoss.term',
            description: 'content.gatewayHelp.terms.packetLoss.description',
          },
        ],
      },
    ],
  },
  {
    id: 'dns',
    titleKey: 'sections.dns',
    icon: <Search className={ICON} />,
    keywords: ['dns', 'lookup', 'resolution', 'a record', 'ptr', 'aaaa'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.dnsTests.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.dnsTests.terms.forwardLookup.term',
            description: 'content.dnsTests.terms.forwardLookup.description',
          },
          {
            term: 'content.dnsTests.terms.reverseLookup.term',
            description: 'content.dnsTests.terms.reverseLookup.description',
          },
          {
            term: 'content.dnsTests.terms.ipv6Lookup.term',
            description: 'content.dnsTests.terms.ipv6Lookup.description',
          },
          {
            term: 'content.dnsTests.terms.latency.term',
            description: 'content.dnsTests.terms.latency.description',
          },
        ],
      },
    ],
  },
  {
    id: 'performance',
    titleKey: 'sections.performance',
    icon: <Zap className={ICON} />,
    keywords: ['performance', 'speed test', 'iperf3', 'throughput', 'download', 'upload', 'jitter'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.performanceTests.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.performanceTests.terms.internetSpeed.term',
            description: 'content.performanceTests.terms.internetSpeed.description',
          },
          {
            term: 'content.performanceTests.terms.lanSpeed.term',
            description: 'content.performanceTests.terms.lanSpeed.description',
          },
          {
            term: 'content.performanceTests.terms.download.term',
            description: 'content.performanceTests.terms.download.description',
          },
          {
            term: 'content.performanceTests.terms.upload.term',
            description: 'content.performanceTests.terms.upload.description',
          },
          {
            term: 'content.performanceTests.terms.latency.term',
            description: 'content.performanceTests.terms.latency.description',
          },
          {
            term: 'content.performanceTests.terms.jitter.term',
            description: 'content.performanceTests.terms.jitter.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:health.title',
            description: 'content.cardHelp.HealthCheckCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:performance.title',
            description: 'content.cardHelp.PerformanceCard.description',
          },
        ],
      },
    ],
  },
  {
    id: 'discovery',
    titleKey: 'sections.discovery',
    icon: <Search className={ICON} />,
    keywords: ['discovery', 'scan', 'arp', 'lldp', 'cdp', 'devices', 'neighbor'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'content.networkDiscovery.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.networkDiscovery.terms.networkScan.term',
            description: 'content.networkDiscovery.terms.networkScan.description',
          },
          {
            term: 'content.networkDiscovery.terms.macAddress.term',
            description: 'content.networkDiscovery.terms.macAddress.description',
          },
          {
            term: 'content.networkDiscovery.terms.vendor.term',
            description: 'content.networkDiscovery.terms.vendor.description',
          },
          {
            term: 'content.networkDiscovery.terms.hostname.term',
            description: 'content.networkDiscovery.terms.hostname.description',
          },
          {
            term: 'content.networkDiscovery.terms.lldpCdp.term',
            description: 'content.networkDiscovery.terms.lldpCdp.description',
          },
        ],
      },
    ],
  },
];
