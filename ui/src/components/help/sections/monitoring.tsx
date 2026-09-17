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
        text: 'content.healthChecks.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.healthChecks.terms.pingTest.term',
            description: 'content.healthChecks.terms.pingTest.description',
          },
          {
            term: 'content.healthChecks.terms.tcpTest.term',
            description: 'content.healthChecks.terms.tcpTest.description',
          },
          {
            term: 'content.healthChecks.terms.httpTest.term',
            description: 'content.healthChecks.terms.httpTest.description',
          },
          {
            term: 'content.healthChecks.terms.customTargets.term',
            description: 'content.healthChecks.terms.customTargets.description',
          },
          {
            term: 'content.healthChecks.terms.thresholds.term',
            description: 'content.healthChecks.terms.thresholds.description',
          },
        ],
      },
      {
        kind: 'tips',
        heading: 'content.healthChecks.commonIssues.title',
        items: [
          'content.healthChecks.commonIssues.timeout',
          'content.healthChecks.commonIssues.highLatency',
          'content.healthChecks.commonIssues.packetLoss',
          'content.healthChecks.commonIssues.connectionRefused',
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
        text: 'content.rtspChecks.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.rtspChecks.terms.rtspUrl.term',
            description: 'content.rtspChecks.terms.rtspUrl.description',
          },
          {
            term: 'content.rtspChecks.terms.options.term',
            description: 'content.rtspChecks.terms.options.description',
          },
          {
            term: 'content.rtspChecks.terms.describe.term',
            description: 'content.rtspChecks.terms.describe.description',
          },
          {
            term: 'content.rtspChecks.terms.authentication.term',
            description: 'content.rtspChecks.terms.authentication.description',
          },
        ],
      },
      {
        kind: 'steps',
        heading: 'content.rtspChecks.configuration.title',
        ordered: true,
        items: [
          {
            description: 'content.rtspChecks.configuration.steps.0',
          },
          {
            description: 'content.rtspChecks.configuration.steps.1',
          },
          {
            description: 'content.rtspChecks.configuration.steps.2',
          },
          {
            description: 'content.rtspChecks.configuration.steps.3',
          },
          {
            description: 'content.rtspChecks.configuration.steps.4',
          },
          {
            description: 'content.rtspChecks.configuration.steps.5',
          },
          {
            description: 'content.rtspChecks.configuration.steps.6',
          },
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
        text: 'content.dicomChecks.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.dicomChecks.terms.aeTitle.term',
            description: 'content.dicomChecks.terms.aeTitle.description',
          },
          {
            term: 'content.dicomChecks.terms.cEcho.term',
            description: 'content.dicomChecks.terms.cEcho.description',
          },
          {
            term: 'content.dicomChecks.terms.association.term',
            description: 'content.dicomChecks.terms.association.description',
          },
          {
            term: 'content.dicomChecks.terms.scp.term',
            description: 'content.dicomChecks.terms.scp.description',
          },
          {
            term: 'content.dicomChecks.terms.scu.term',
            description: 'content.dicomChecks.terms.scu.description',
          },
          {
            term: 'content.dicomChecks.terms.port.term',
            description: 'content.dicomChecks.terms.port.description',
          },
        ],
      },
      {
        kind: 'note',
        text: 'content.dicomChecks.compliance.points.0',
      },
      {
        kind: 'note',
        text: 'content.dicomChecks.compliance.points.3',
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
        text: 'content.security.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.security.terms.portScan.term',
            description: 'content.security.terms.portScan.description',
          },
          {
            term: 'content.security.terms.vulnScan.term',
            description: 'content.security.terms.vulnScan.description',
          },
          {
            term: 'content.security.terms.devicePosture.term',
            description: 'content.security.terms.devicePosture.description',
          },
          {
            term: 'content.security.terms.rogueDhcp.term',
            description: 'content.security.terms.rogueDhcp.description',
          },
        ],
      },
      {
        kind: 'steps',
        heading: 'content.security.passwordRecovery.title',
        ordered: true,
        items: [
          {
            description: 'content.security.passwordRecovery.steps.0',
          },
          {
            description: 'content.security.passwordRecovery.steps.1',
          },
          {
            description: 'content.security.passwordRecovery.steps.2',
          },
          {
            description: 'content.security.passwordRecovery.steps.3',
          },
          {
            description: 'content.security.passwordRecovery.steps.4',
          },
          {
            description: 'content.security.passwordRecovery.steps.5',
          },
          {
            description: 'content.security.passwordRecovery.steps.6',
          },
          {
            description: 'content.security.passwordRecovery.steps.7',
          },
        ],
      },
      {
        kind: 'note',
        text: 'content.security.passwordRecovery.note',
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:mfa.title',
            description: 'content.cardHelp.MfaCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:guestAudit.title',
            description: 'content.cardHelp.GuestNetworkAuditCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:insecurePorts.title',
            description: 'content.cardHelp.InsecurePortScanCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:bluetooth.title',
            description: 'content.cardHelp.BluetoothCard.description',
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
];
