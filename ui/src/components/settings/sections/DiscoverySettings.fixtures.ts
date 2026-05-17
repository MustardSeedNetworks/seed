/**
 * DiscoverySettings.fixtures.ts — Storybook fixtures and a baseArgs() helper
 * extracted from DiscoverySettings.stories.tsx. The fixtures keep the per-
 * story override blocks short by providing a complete args object with
 * sensible defaults; each story spreads `baseArgs()` and overrides only the
 * fields it cares about.
 */

import type { ComponentProps } from 'react';
import type { NetworkDiscoverySettings, SNMPSettings, SubnetConfig } from '../../../types/settings';
import type { DiscoverySettings } from './DiscoverySettings';

const noop = (): void => {
  // intentionally empty
};

export const defaultSettings: NetworkDiscoverySettings = {
  enabled: true,
  arpScanWorkers: 50,
  pingTimeoutMs: 500,
  scanTimeoutMs: 30000,
  autoScan: false,
  scanIntervalMs: 0,
  ouiFilePath: 'data/oui.txt',
  options: {
    passiveProtocols: {
      lldp: true,
      cdp: true,
      edp: false,
      ndp: false,
    },
    arpScan: true,
    icmpScan: true,
    portScan: {
      enabled: false,
      preset: 'common',
      tcpPorts: '',
      udpPorts: '',
      bannerTimeoutMs: 3000,
    },
    tcpProbe: {
      timeoutMs: 3000,
      workers: 10,
    },
    traceroute: false,
    snmpQuery: false,
  },
  timing: {
    probeIntervalMs: 100,
    rescanIntervalMs: 300000,
    workers: 50,
  },
  profiler: {
    enabled: true,
    timeoutMs: 5000,
    maxConcurrent: 10,
    quickPorts: [22, 80, 443],
  },
  fingerprinting: {
    enabled: true,
    osDetection: true,
    serviceProbes: true,
  },
  ipv6Enabled: false,
};

export const defaultSnmpSettings: SNMPSettings = {
  communities: ['public'],
  v3Credentials: [],
  timeout: 5000,
  retries: 2,
  port: 161,
};

type DiscoverySettingsArgs = ComponentProps<typeof DiscoverySettings>;

/**
 * Returns a complete DiscoverySettings args object using fixture defaults.
 * Each story then overrides only the fields under test.
 */
export const baseArgs = (subnets: SubnetConfig[] = []): DiscoverySettingsArgs => ({
  networkDiscoverySettings: defaultSettings,
  setNetworkDiscoverySettings: noop,
  networkDiscoveryStatus: 'idle',
  subnets,
  subnetsStatus: 'idle',
  newSubnetCidr: '',
  setNewSubnetCidr: noop,
  newSubnetName: '',
  setNewSubnetName: noop,
  subnetError: null,
  setSubnetError: noop,
  addSubnet: noop,
  toggleSubnet: noop,
  deleteSubnet: noop,
  snmpSettings: defaultSnmpSettings,
  setSnmpSettings: noop,
  snmpStatus: 'idle',
});
