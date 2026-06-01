/**
 * helpDrawerContent.tsx
 *
 * Typed, data-driven help content for The Seed's HelpDrawer. Mirrors niac's
 * data-layer shape (a flat section list rendered by small generic components)
 * but is adapted to Seed's diagnostic feature set.
 *
 * Section bodies are authored as a sequence of typed `HelpBlock`s so a single
 * generic renderer can present every section without bespoke per-section JSX.
 * Content is FACTUAL — drawn from Seed's real feature set
 * (Roots/Canopy/Shell/Sap/Harvest plus the dashboard diagnostics). Section
 * titles come from the `help` i18n namespace (`sections.*`); body copy is
 * authored here in English. No invented features, no banned vocabulary.
 *
 * Named `helpDrawerContent` (not `helpContent`) to avoid a case-insensitive
 * filesystem collision with the existing `HelpContent.tsx` tooltip constants.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import type { ReactNode } from 'react';
import type { HelpTranslations } from '../../i18n/types';
import {
  Activity,
  AlertTriangle,
  BarChart3,
  BookOpen,
  Cable,
  Heart,
  HeartPulse,
  Info,
  LayoutDashboard,
  Lightbulb,
  Monitor,
  Network,
  Route,
  ScrollText,
  Search,
  Server,
  Shield,
  Signal,
  SlidersHorizontal,
  Wifi,
  Zap,
} from '../ui/icons';

// ---------------------------------------------------------------------------
// Typed content model
// ---------------------------------------------------------------------------

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

const ICON = 'w-4 h-4';

// ---------------------------------------------------------------------------
// Sections — order matches the legacy modal / help.json `sections.*` set.
// ---------------------------------------------------------------------------

export const helpSections: HelpSection[] = [
  {
    id: 'about',
    titleKey: 'sections.about',
    icon: <Info className={ICON} />,
    keywords: ['about', 'overview', 'modules', 'roots', 'canopy', 'shell', 'sap', 'harvest'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'The Seed is a network diagnostics and monitoring tool by Mustard Seed Networks. It gives you visibility into physical-layer link state, IP configuration, gateway and DNS reachability, device discovery, throughput, Wi-Fi connection quality, and endpoint health from a single dashboard.',
      },
      {
        kind: 'terms',
        heading: 'Modules',
        items: [
          {
            term: 'Roots',
            description:
              'Path and route analysis — examines how traffic leaves the local network and reaches a destination.',
          },
          {
            term: 'Canopy',
            description:
              'Wi-Fi visibility and troubleshooting — connected-SSID signal/SNR, neighbor access-point scanning, and channel utilization. Focused on diagnosing wireless problems, not coverage planning.',
          },
          {
            term: 'Shell',
            description:
              'Security posture — surfaces open ports, device posture, and configuration risks discovered on the network.',
          },
          {
            term: 'Sap',
            description:
              'Live telemetry — streams real-time metrics (link, signal, latency, throughput) as tests run.',
          },
          {
            term: 'Harvest',
            description:
              'Reporting — collects diagnostic results into exportable reports for documentation and handoff.',
          },
        ],
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
          'Export diagnostics from Harvest for documentation or troubleshooting handoff.',
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
        text: 'Wi-Fi Status monitors the quality and settings of the current wireless connection — part of the Canopy module for Wi-Fi visibility and troubleshooting.',
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
  {
    id: 'wifiSurvey',
    titleKey: 'sections.wifiSurvey',
    icon: <Signal className={ICON} />,
    keywords: ['wifi survey', 'site survey', 'channel utilization', 'neighbor', 'snr', 'canopy'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'The Wi-Fi Site Survey scans visible wireless networks to help troubleshoot the local wireless environment: neighbor access points, the channels in use, and how busy each channel is. It is a visibility and troubleshooting tool focused on diagnosing wireless problems.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Neighbor AP scan',
            description:
              'Lists the access points the adapter can hear, with their SSID, BSSID, band, channel, and signal level.',
          },
          {
            term: 'Channel utilization',
            description:
              'Indicates how busy each Wi-Fi channel is. High utilization (over ~50%) points to congestion and likely interference.',
          },
          {
            term: 'Co-channel interference',
            description:
              'When multiple access points share the same channel they must share airtime, which reduces throughput.',
          },
          {
            term: 'Signal / SNR',
            description:
              'Signal level in dBm and the signal-to-noise ratio of the connection. A higher SNR means a cleaner, more reliable link.',
          },
        ],
      },
      {
        kind: 'terms',
        heading: 'Bands',
        items: [
          {
            term: '2.4 GHz',
            description:
              'Better range and wall penetration, but only three non-overlapping channels (1, 6, 11) and more congestion.',
          },
          {
            term: '5 GHz',
            description:
              'Many non-overlapping channels, less interference, and faster speeds, but shorter range.',
          },
        ],
      },
      {
        kind: 'note',
        text: 'Advanced Wi-Fi scanning depends on adapter capabilities; built-in laptop Wi-Fi may report a limited view of the environment.',
      },
    ],
  },
  {
    id: 'network',
    titleKey: 'sections.network',
    icon: <Network className={ICON} />,
    keywords: ['network', 'dhcp', 'lease', 'ip', 'subnet'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Network & DHCP shows the interface IP configuration and details of the current DHCP lease.',
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
  {
    id: 'healthChecks',
    titleKey: 'sections.healthChecks',
    icon: <Heart className={ICON} />,
    keywords: ['health checks', 'ping', 'tcp', 'http', 'monitoring', 'endpoints'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Health Checks monitor endpoint availability with automated ping, TCP, and HTTP tests.',
      },
      {
        kind: 'terms',
        heading: 'Check types',
        items: [
          {
            term: 'Ping (ICMP)',
            description:
              'Sends ICMP echo requests to verify reachability and report latency and packet loss.',
          },
          {
            term: 'TCP connection',
            description:
              'Attempts a TCP handshake to a port to verify a service is accepting connections.',
          },
          {
            term: 'HTTP',
            description:
              'Performs a full HTTP request including DNS, TCP, TLS, and response-time measurement.',
          },
          {
            term: 'Custom targets',
            description:
              'Add your own endpoints to monitor in Settings — internal servers, cloud services, or critical infrastructure.',
          },
          {
            term: 'Thresholds',
            description:
              'Set warning and critical latency thresholds in Settings to flag degraded endpoints.',
          },
        ],
      },
      {
        kind: 'tips',
        heading: 'Common issues',
        items: [
          'A timeout indicates the host is unreachable, a firewall is blocking, or there is a network-path problem.',
          'High latency may indicate congestion, routing issues, or an overloaded server.',
          'Connection refused means the service is not running or not listening on that port.',
        ],
      },
    ],
  },
  {
    id: 'rtspChecks',
    titleKey: 'sections.rtspChecks',
    icon: <Monitor className={ICON} />,
    keywords: ['rtsp', 'camera', 'stream', 'surveillance', 'nvr', 'options', 'describe'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'RTSP Monitoring verifies connectivity to RTSP endpoints such as IP cameras, NVRs, and video-management systems.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'RTSP URL',
            description:
              'The address of the stream, typically rtsp://host:port/path. Port 554 is the standard RTSP port.',
          },
          {
            term: 'OPTIONS request',
            description:
              'An RTSP command that asks the server which methods it supports — a lightweight connectivity check.',
          },
          {
            term: 'DESCRIBE request',
            description:
              'Requests the media description (SDP) to confirm the stream exists and is accessible.',
          },
          {
            term: 'Authentication',
            description:
              'RTSP servers usually require a username and password; Basic and Digest authentication are supported.',
          },
        ],
      },
      {
        kind: 'steps',
        heading: 'Configuring an endpoint',
        ordered: true,
        items: [
          { description: 'Open Settings and add an RTSP endpoint.' },
          { description: 'Give it a descriptive name (for example "Lobby Camera 1").' },
          { description: 'Enter the RTSP URL, for example rtsp://192.168.1.100:554/stream1.' },
          { description: 'Add credentials if the camera requires them.' },
          { description: 'Set the check interval and enable the endpoint to start monitoring.' },
        ],
      },
    ],
  },
  {
    id: 'dicomChecks',
    titleKey: 'sections.dicomChecks',
    icon: <HeartPulse className={ICON} />,
    keywords: ['dicom', 'c-echo', 'pacs', 'ae title', 'medical', 'imaging'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'DICOM Health Checks verify connectivity to medical-imaging systems (CT, MRI, ultrasound, PACS) using C-ECHO — a DICOM equivalent of ping that confirms an association can be established.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'AE Title',
            description:
              'Application Entity Title — a unique identifier (up to 16 characters) for a DICOM node. Both the calling (local) and called (remote) AE Titles must be configured correctly.',
          },
          {
            term: 'C-ECHO',
            description:
              'The DICOM verification service — a "ping" that confirms the remote node is reachable and responding.',
          },
          {
            term: 'Association',
            description:
              'A DICOM connection between two nodes. Both AE Titles must be registered in each other for an association to succeed.',
          },
          {
            term: 'DICOM port',
            description:
              'The standard port is 104, but many systems use 11112 or a custom port. Confirm with your PACS administrator.',
          },
        ],
      },
      {
        kind: 'note',
        text: 'C-ECHO is a non-destructive verification — it does not access patient data. Coordinate AE Title registration with your biomedical or IT team.',
      },
    ],
  },
  {
    id: 'security',
    titleKey: 'sections.security',
    icon: <Shield className={ICON} />,
    keywords: [
      'security',
      'port scan',
      'vulnerability',
      'posture',
      'rogue dhcp',
      'shell',
      'password',
    ],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Security & Administration covers device scanning, posture assessment, and account administration — the basis of the Shell module for security posture.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Port scanning',
            description:
              'Identifies open ports on discovered devices to help spot unauthorized services or risks.',
          },
          {
            term: 'Vulnerability scan',
            description: 'Checks devices for known issues based on detected services and versions.',
          },
          {
            term: 'Device posture',
            description:
              'Assesses the security posture of network devices — open ports, outdated services, and misconfigurations.',
          },
          {
            term: 'Rogue DHCP detection',
            description:
              'Detects unauthorized DHCP servers that could intercept traffic or hand out malicious configuration.',
          },
        ],
      },
      {
        kind: 'steps',
        heading: 'Password recovery',
        ordered: true,
        items: [
          { description: 'SSH into the server running The Seed.' },
          {
            description:
              'Create an empty .recovery file in the data directory (user mode: ~/.local/share/seed/.recovery; system mode: /var/lib/seed/.recovery).',
          },
          {
            description:
              'The server detects the file and generates a single-use recovery token in .recovery-token in the same directory.',
          },
          {
            description:
              'Enter that token on the login page with your new password. The token expires after 15 minutes.',
          },
        ],
      },
      {
        kind: 'note',
        text: 'Password recovery requires filesystem access to the server, which proves you have admin-level access to the machine.',
      },
    ],
  },
  {
    id: 'troubleshooting',
    titleKey: 'sections.troubleshooting',
    icon: <AlertTriangle className={ICON} />,
    keywords: ['troubleshooting', 'no carrier', 'slow', 'open', 'short', 'unreachable'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Common symptoms and where to look first.',
      },
      {
        kind: 'terms',
        heading: 'Link problems',
        items: [
          {
            term: 'No carrier detected',
            description:
              'Check the cable is seated at both ends, the switch/router is powered with an active port LED, and the interface shows UP. Try another cable or port.',
          },
          {
            term: 'Link speed lower than expected',
            description:
              'Use Cat5e or Cat6 cable for gigabit, run a cable test to confirm all four pairs are OK, and check for an auto-negotiation mismatch.',
          },
        ],
      },
      {
        kind: 'terms',
        heading: 'Cable faults',
        items: [
          {
            term: "Pair shows 'Open'",
            description:
              'A wire is broken or disconnected. Check terminations at the patch panel or wall jack, or re-terminate the connector.',
          },
          {
            term: "Pair shows 'Short'",
            description:
              'Two wires are touching. Inspect the connector for bent pins and check the cable for crush damage.',
          },
        ],
      },
      {
        kind: 'terms',
        heading: 'Connectivity',
        items: [
          {
            term: 'Gateway unreachable',
            description:
              'Verify the gateway IP matches the router LAN IP, check the physical connection, and restart the router if it is unresponsive.',
          },
          {
            term: 'Slow internet speed test',
            description:
              'Test over a wired connection to rule out Wi-Fi, retry at a different time, and restart the modem and router.',
          },
        ],
      },
    ],
  },
  {
    id: 'howTo',
    titleKey: 'sections.howTo',
    icon: <Lightbulb className={ICON} />,
    keywords: ['how to', 'guide', 'diagnose', 'survey', 'health checks', 'walkthrough'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Step-by-step guides for common tasks.',
      },
      {
        kind: 'steps',
        heading: 'Diagnose slow network speeds',
        ordered: true,
        items: [
          {
            description:
              'Check Link Status shows the expected speed (1 Gbps for gigabit). If it shows 100 Mbps, check cable quality or the switch port.',
          },
          {
            description:
              'Test gateway latency — it should be under 1 ms on a wired connection. High latency here points to a local problem.',
          },
          { description: 'Run an internet speed test and compare against your ISP plan.' },
          {
            description:
              'Run a LAN iperf3 test to isolate whether the bottleneck is local or internet-bound.',
          },
          {
            description:
              'For Wi-Fi, run a site survey to check channel congestion and interference.',
          },
        ],
      },
      {
        kind: 'steps',
        heading: 'Configure health checks for critical services',
        ordered: true,
        items: [
          {
            description:
              'List the services that need monitoring — servers, databases, cloud services, cameras, medical equipment.',
          },
          {
            description:
              'Choose a check type: ping for reachability, TCP for service ports, HTTP for web services, RTSP for cameras, DICOM for imaging.',
          },
          {
            description:
              'Add each endpoint in Settings with its parameters and set warning/critical thresholds.',
          },
          {
            description:
              'Run the tests manually to confirm connectivity, then enable continuous monitoring and save to the appropriate profile.',
          },
        ],
      },
    ],
  },
  {
    id: 'path',
    titleKey: 'sections.path',
    icon: <Route className={ICON} />,
    keywords: ['path', 'traceroute', 'route', 'hops', 'arp', 'l2', 'l3', 'gateway'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Path Analysis traces how traffic leaves the local network and reaches a destination. It surfaces every L2 hop on the local segment, every L3 hop on the route off-link, and the on-link devices ARP/ND can see along the way. The Roots module owns this surface.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'L2 path',
            description:
              'Hops within the same broadcast domain (switches, bridges), discovered via ARP and on-link MAC tables.',
          },
          {
            term: 'L3 path',
            description:
              'Per-hop IPv4/IPv6 traceroute with round-trip latency to each hop and any AS / reverse-DNS metadata.',
          },
          {
            term: 'Gateway hop',
            description:
              'The first L3 hop off the local subnet — usually the router that issued the DHCP lease.',
          },
          {
            term: 'On-link discovery',
            description:
              'ARP / ND sweep that surfaces neighbors visible without crossing a router.',
          },
        ],
      },
    ],
  },
  {
    id: 'reports',
    titleKey: 'sections.reports',
    icon: <BarChart3 className={ICON} />,
    keywords: ['reports', 'sla', 'compliance', 'history', 'export', 'csv', 'json', 'pdf'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Reports collects the results of Seed’s diagnostic tests over time and exports them as SLA dashboards, compliance summaries, and historical CSV/JSON. The Harvest module owns this surface.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'SLA dashboard',
            description:
              'Rolling availability and latency view derived from health-check probe results.',
          },
          {
            term: 'Compliance summary',
            description:
              'Snapshot of which checks pass against the active profile’s thresholds — useful for audits.',
          },
          {
            term: 'Scheduled reports',
            description: 'Pro-tier feature that produces a periodic PDF report on a cadence.',
          },
          {
            term: 'Export',
            description:
              'Download a slice of the underlying data as CSV or JSON for downstream tooling.',
          },
        ],
      },
    ],
  },
  {
    id: 'logs',
    titleKey: 'sections.logs',
    icon: <ScrollText className={ICON} />,
    keywords: ['logs', 'log', 'stream', 'tail', 'daemon', 'level', 'source', 'debug'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Logs streams the seed daemon’s structured log entries live as they are emitted, with filters by level, source, and free-text. Useful for confirming what the backend just did or diagnosing why a test failed.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Level',
            description:
              'Standard severity (debug / info / warn / error). Filter to narrow the view.',
          },
          {
            term: 'Source',
            description:
              'The internal package emitting the entry (for example discovery, canopy, shell).',
          },
          {
            term: 'Live tail',
            description:
              'WebSocket stream of new entries as they are produced by the running seed process.',
          },
          {
            term: 'Daemon health',
            description:
              'Rotating-file usage, error counts, and uptime of the seed process itself.',
          },
        ],
      },
    ],
  },
  {
    id: 'alerts',
    titleKey: 'sections.alerts',
    icon: <AlertTriangle className={ICON} />,
    keywords: ['alerts', 'alert', 'acknowledge', 'resolve', 'severity', 'notification', 'incident'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Alerts lists the conditions the monitoring pipelines have raised — for example a polled device going unreachable or an interface changing state. Filter by severity, acknowledged, or resolved, then act on each row: acknowledge marks it seen, resolve marks it fixed. The person who clicks is recorded against the alert.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Severity',
            description:
              'How urgent the condition is. Filter to focus on the most important alerts first.',
          },
          {
            term: 'Acknowledge',
            description:
              'Marks an alert as seen without closing it — signals someone is looking into it.',
          },
          {
            term: 'Resolve',
            description: 'Marks an alert as fixed and removes it from the default unresolved view.',
          },
          {
            term: 'Acknowledged by',
            description:
              'The operator who acknowledged the alert, taken from the signed-in identity.',
          },
        ],
      },
    ],
  },
  {
    id: 'pollingTargets',
    titleKey: 'sections.pollingTargets',
    icon: <Server className={ICON} />,
    keywords: ['polling', 'targets', 'snmp', 'monitor', 'device', 'collector', 'add', 'edit'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Polling targets is the list of devices Seed polls over SNMP. Add a target to start monitoring it, edit one to change its settings, or remove one you no longer track. A new target picks up the default collector chain and begins polling on the next cycle; the devices and links it discovers appear on the Topology page, and state changes surface as alerts.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Target',
            description: 'A device (by address) that Seed polls on a recurring interval.',
          },
          {
            term: 'Collector chain',
            description:
              'The set of SNMP collectors run against a target (system info, interface table, LLDP, ARP, and forwarding-database neighbors).',
          },
          {
            term: 'SNMP',
            description:
              'Simple Network Management Protocol — the standard used to read device state and tables.',
          },
        ],
      },
    ],
  },
  {
    id: 'topology',
    titleKey: 'sections.topology',
    icon: <Network className={ICON} />,
    keywords: ['topology', 'graph', 'nodes', 'links', 'neighbors', 'map', 'interfaces'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Topology shows the network graph reconciled from discovery and SNMP polling: every node visible to your session, with its interfaces and the links between nodes. Select a node to open a detail panel listing its interfaces and discovered links. The graph is built from the same observations that drive polling targets and alerts.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Node',
            description:
              'A discovered device in the graph — switch, router, host, or access point.',
          },
          {
            term: 'Link',
            description:
              'A neighbor relationship between two nodes, learned from LLDP/CDP or forwarding tables.',
          },
          {
            term: 'Interface',
            description: 'A port on a node, with its status and any links observed on it.',
          },
        ],
      },
    ],
  },
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
