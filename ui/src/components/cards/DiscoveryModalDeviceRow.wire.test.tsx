/**
 * DeviceRow reads the wire, not a hand-typed mirror of it (seed#2393).
 *
 * networkDiscoveryCardTypes.ts used to declare its own copy of the discovery
 * tree, and two of its fields were spelled in a way the daemon has never sent:
 *
 *   - `device.vulnerabilities` was `{ count, highestSeverity }`; the wire sends
 *     `DeviceVulnerabilities`, whose findings live in `vulnerabilities[]`. The
 *     badge read two undefineds, so the column was always "-" no matter how
 *     many CVEs a device had.
 *   - `SNMPEntity.class` was mirrored as `className`, so the hardware-inventory
 *     filter (`chassis` / `module` / `powerSupply`) never matched and the
 *     Hardware block rendered nothing for a device with a full ENTITY-MIB walk.
 *
 * Both are typed-through-to-generated now. These render a wire-shaped device
 * and assert what the operator sees, so a future hand-typed mirror fails here
 * rather than silently blanking the column again.
 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type {
  DiscoveredDevice,
  Vulnerability,
} from '../../types/generated/engine-discovery-response';
import { DeviceRow, highestSeverity } from './DiscoveryModalDeviceRow';

function finding(cveId: string, severity: string): Vulnerability {
  return {
    cveId,
    description: 'test finding',
    severity,
    score: 7.5,
    published: '2026-01-01T00:00:00Z',
    modified: '2026-01-01T00:00:00Z',
    references: [],
    affectedCpe: '',
  };
}

const device: DiscoveredDevice = {
  ip: '10.44.10.7',
  mac: '00:1b:21:aa:bb:01',
  hostname: 'sw-core-01',
  discoveryMethod: ['arp', 'snmp'],
  lastSeen: new Date().toISOString(),
  isLocal: true,
  vulnerabilities: {
    deviceIp: '10.44.10.7',
    mac: '00:1b:21:aa:bb:01',
    hostname: 'sw-core-01',
    vendor: 'Cisco',
    product: 'IOS',
    version: '15.2',
    scanTime: new Date().toISOString(),
    vulnerabilities: [
      finding('CVE-2026-0001', 'MEDIUM'),
      finding('CVE-2026-0002', 'CRITICAL'),
      finding('CVE-2026-0003', 'LOW'),
    ],
  },
  snmpData: {
    collectedAt: new Date().toISOString(),
    inventory: [
      { index: 1, class: 'chassis', name: 'Chassis', serialNum: 'FOC1234X5YZ' },
      { index: 2, class: 'sensor', name: 'Temp sensor' },
    ],
  },
};

function renderRow(expanded: boolean) {
  return render(
    <table>
      <tbody>
        <DeviceRow device={device} isExpanded={expanded} onToggle={() => {}} isScanning={false} />
      </tbody>
    </table>,
  );
}

describe('DeviceRow against the wire shape', () => {
  it('counts the findings the wire sends', () => {
    renderRow(false);
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('renders an inventory entity the wire spells with `class`', () => {
    renderRow(true);
    expect(screen.getByText('Chassis')).toBeInTheDocument();
    expect(screen.queryByText('Temp sensor')).not.toBeInTheDocument();
  });
});

describe('highestSeverity', () => {
  it('picks the worst severity present, not the first', () => {
    expect(highestSeverity([finding('a', 'LOW'), finding('b', 'CRITICAL')])).toBe('CRITICAL');
    expect(highestSeverity([finding('a', 'MEDIUM'), finding('b', 'HIGH')])).toBe('HIGH');
  });

  it('falls back to LOW for no findings', () => {
    expect(highestSeverity(undefined)).toBe('LOW');
    expect(highestSeverity([])).toBe('LOW');
  });
});
