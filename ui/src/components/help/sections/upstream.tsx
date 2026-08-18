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
        text: 'The Network page shows the diagnostic state of the upstream link: DHCP lease, default gateway, DNS resolvers, the public IP the gateway uses, and (when wired) the directly-attached switch and its VLAN configuration.',
      },
      {
        kind: 'paragraph',
        text: 'Each card refreshes on its own schedule and reflects the active interface selected in the header. If a card shows "no data", either the interface has not yet been probed, or that piece of the upstream config is not present (e.g., no IPv6 gateway).',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Lease time',
            description:
              'How long the current IP address assignment is valid before renewal is required.',
          },
          {
            term: 'DHCP server',
            description:
              'IP address of the DHCP server that issued the lease (usually your router).',
          },
          {
            term: 'Gateway',
            description: 'Default gateway assigned by DHCP for routing traffic off-subnet.',
          },
          {
            term: 'DNS servers',
            description: 'DNS servers assigned by DHCP for name resolution.',
          },
          {
            term: 'Subnet mask',
            description: 'Network mask defining the size of the local subnet.',
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
        text: 'Gateway tests reachability and latency to your default gateway.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'IPv4 gateway',
            description: 'Default router for IPv4 traffic leaving your local network.',
          },
          {
            term: 'IPv6 gateway',
            description: 'Default router for IPv6 traffic (may be a link-local address).',
          },
          {
            term: 'Reachability',
            description: 'Whether the gateway responds to ICMP ping requests.',
          },
          {
            term: 'Latency',
            description:
              'Round-trip time to the gateway. It should be under 1 ms on a local network.',
          },
          {
            term: 'Packet loss',
            description: "Percentage of ping packets that didn't receive a response.",
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
        text: 'DNS Tests check name-resolution performance and functionality.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Forward lookup',
            description: 'Resolves a hostname to an IPv4 address (A record).',
          },
          {
            term: 'Reverse lookup',
            description: 'Resolves an IP address back to a hostname (PTR record).',
          },
          {
            term: 'IPv6 lookup',
            description: 'Resolves a hostname to an IPv6 address (AAAA record).',
          },
          {
            term: 'Latency',
            description:
              'Time for the DNS query to complete. Under 50 ms is good for a local resolver.',
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
        text: 'Performance Tests measure network throughput and latency.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Internet speed test',
            description:
              'Measures download and upload speed to public speed-test servers — your connection to the internet.',
          },
          {
            term: 'LAN speed (iperf3)',
            description:
              'Measures throughput on the local network using iperf3 against a configured server.',
          },
          {
            term: 'Download / Upload',
            description: 'Maximum download and upload speeds achieved during the test.',
          },
          {
            term: 'Latency',
            description: 'Round-trip time (ping) to the test server.',
          },
          {
            term: 'Jitter',
            description:
              'Variation in latency over time. Lower is better for real-time traffic such as voice and video.',
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
        text: 'Network Discovery finds devices on your network and identifies directly connected switches.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Network scan',
            description:
              'Discovers active devices on the local subnet using ARP and ICMP ping sweeps.',
          },
          {
            term: 'MAC address',
            description: 'The hardware address of a device interface — a unique identifier.',
          },
          {
            term: 'Vendor',
            description: 'Manufacturer identified from the MAC address OUI (first three bytes).',
          },
          {
            term: 'Hostname',
            description: 'The DNS hostname, when a reverse lookup succeeds.',
          },
          {
            term: 'LLDP / CDP',
            description:
              'Link Layer Discovery Protocol (standard) or Cisco Discovery Protocol — reveal details about directly connected switches.',
          },
        ],
      },
    ],
  },
];
