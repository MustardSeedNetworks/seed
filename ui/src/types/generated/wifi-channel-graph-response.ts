/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiChannelGraphResponse {
  interface: string;
  available: boolean;
  error?: string;
  remediation?: string;
  data?: ChannelGraphData;
}
export interface ChannelGraphData {
  networks24Ghz: ChannelNetwork[];
  networks5Ghz: ChannelNetwork[];
  networks6Ghz: ChannelNetwork[];
  connectedBssid?: string;
  scanTime: string;
}
export interface ChannelNetwork {
  ssid: string;
  bssid: string;
  channel: number;
  centerFreqMhz: number;
  channelWidthMhz: number;
  signalDbm: number;
  band: '2.4GHz' | '5GHz' | '6GHz';
  isConnected: boolean;
}
