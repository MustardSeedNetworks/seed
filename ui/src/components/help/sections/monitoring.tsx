/**
 * monitoring.tsx — Continuous checks and the security posture view.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { Heart, HeartPulse, Monitor, Shield } from '../../ui/icons';
import type { HelpSection } from '../helpModel';

const ICON = 'w-4 h-4';

export const monitoringSections: HelpSection[] = [
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
        text: 'Security & Administration covers device scanning, posture assessment, and account administration — security posture for the network.',
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
];
