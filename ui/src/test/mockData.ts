/**
 * Mock Data for Storybook Stories
 *
 * Centralized mock data for use in Storybook stories. Only the fixtures
 * actually imported elsewhere live here; older inline fixtures for
 * cards whose stories have been simplified or removed were pruned in
 * the seed#1052 cleanup (alpha-era, no backwards-compat needs).
 */

import type { NetworkDiscoveryData } from '../components/cards/NetworkDiscoveryCard';

// ============================================================================
// Network Discovery Card Mock Data — consumed by DiscoveryModal.stories.tsx
// ============================================================================

const profileFor = (deviceType: string, ports: number[]) => ({
  deviceType,
  openPorts: ports.map((port) => ({ port, protocol: 'tcp', isOpen: true })),
  profiledAt: new Date().toISOString(),
});

export const mockNetworkDiscoveryData: Record<string, NetworkDiscoveryData> = {
  withDevices: {
    devices: [
      {
        ip: '192.168.1.1',
        mac: 'aa:bb:cc:dd:ee:ff',
        hostname: 'router.local',
        vendor: 'Cisco Systems',
        lastSeen: new Date(Date.now() - 60000).toISOString(),
        // deviceType lives on DeviceProfile, not DiscoveredDevice
        isLocal: true,
        discoveryMethod: ['arp'],
        profile: profileFor('router', [22, 80, 443]),
      },
      {
        ip: '192.168.1.100',
        mac: '11:22:33:44:55:66',
        hostname: 'workstation-01',
        vendor: 'Dell Inc.',
        lastSeen: new Date(Date.now() - 30000).toISOString(),
        // deviceType lives on DeviceProfile, not DiscoveredDevice
        isLocal: true,
        discoveryMethod: ['arp'],
        profile: profileFor('computer', [22]),
      },
      {
        ip: '192.168.1.150',
        mac: '77:88:99:aa:bb:cc',
        hostname: 'printer-office',
        vendor: 'HP Inc.',
        lastSeen: new Date(Date.now() - 120000).toISOString(),
        // deviceType lives on DeviceProfile, not DiscoveredDevice
        isLocal: true,
        discoveryMethod: ['arp'],
        profile: profileFor('printer', [9100]),
      },
    ],
    status: {
      scanning: false,
      deviceCount: 3,
      lastScan: new Date(Date.now() - 60000).toISOString(),
      subnet: '192.168.1.0/24',
      localIP: '192.168.1.100',
      interface: 'eth0',
    },
  } satisfies NetworkDiscoveryData,

  scanning: {
    devices: [],
    status: {
      scanning: true,
      deviceCount: 0,
      lastScan: '',
      subnet: '192.168.1.0/24',
      localIP: '192.168.1.100',
      interface: 'eth0',
    },
  } satisfies NetworkDiscoveryData,

  empty: {
    devices: [],
    status: {
      scanning: false,
      deviceCount: 0,
      lastScan: new Date(Date.now() - 300000).toISOString(),
      subnet: '192.168.1.0/24',
      localIP: '192.168.1.100',
      interface: 'eth0',
    },
  } satisfies NetworkDiscoveryData,
};
