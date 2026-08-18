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

import { AlertTriangle, Lightbulb } from '../../ui/icons';
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
    keywords: ['how to', 'guide', 'diagnose', 'health checks', 'walkthrough'],
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
              'For Wi-Fi, check the channel-overlap view for congestion and interference.',
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
];
