import type { PortPreset } from '../../../../types/settings';

/**
 * Port presets used by the DiscoveryCustomOptions panel.
 *
 * Each preset describes the TCP / UDP port list that the discovery scanner
 * should use and a human-readable description shown beneath the preset
 * picker. The lists mirror the same presets defined in the Go backend
 * (internal/config/config_types_network.go: PortsCommon/Secure/InsecureTCP).
 *
 * - common:   OS / application / service identification
 * - secure:   Encrypted, authenticated services
 * - insecure: Ports that should probably be disabled
 * - custom:   User-edited port lists (empty by default)
 */
export const PORT_PRESETS: Record<PortPreset, { tcp: string; udp: string; description: string }> = {
  common: {
    tcp: '21,22,23,25,53,80,110,111,135,139,143,443,445,993,995,1433,1521,3306,3389,5432,5900,5985,8080,8443',
    udp: '53,67,68,69,123,137,138,161,162,500,514,1900',
    description: 'Common service ports for OS/application identification',
  },
  secure: {
    tcp: '22,443,465,587,636,853,993,995,8443,9443',
    udp: '443,500,4500,853',
    description: 'Encrypted and authenticated services (recommended)',
  },
  insecure: {
    tcp: '21,23,25,69,80,110,111,135,139,143,445,512,513,514,1099,2049,3389,5800,5900,6000-6009',
    udp: '67,68,69,111,137,138,161,162,514,1900,2049',
    description: 'Insecure ports that should be disabled',
  },
  custom: {
    tcp: '',
    udp: '',
    description: 'Manually configure port lists',
  },
};
