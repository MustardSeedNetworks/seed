/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface EngineDiscoveryResponse {
  devices: DiscoveredDevice[];
  stats: EngineStats;
  scanResult?: ScanResult;
  capabilities: {
    [k: string]: boolean;
  };
}
export interface DiscoveredDevice {
  ip: string;
  ipv6?: string;
  ipv6Addresses?: string[];
  mac: string;
  hostname?: string;
  netbiosName?: string;
  mdnsName?: string;
  displayName?: string;
  vendor?: string;
  osGuess?: string;
  ttl?: number;
  discoveryMethod: string[];
  lastSeen: string;
  isLocal: boolean;
  isRouter?: boolean;
  hasDuplicateIP?: boolean;
  duplicateMACs?: string[];
  lldpInfo?: LLDPDeviceInfo;
  cdpInfo?: CDPDeviceInfo;
  edpInfo?: EDPDeviceInfo;
  ndpInfo?: NDPDeviceInfo;
  profile?: DeviceProfile;
  snmpData?: SNMPFullData;
  vulnerabilities?: DeviceVulnerabilities;
  wolCapable?: boolean;
  wolStatus?: string;
  connectionTypes?: string[];
  wifiPresence?: WiFiPresence;
  bluetoothPresence?: BluetoothPresence;
}
export interface LLDPDeviceInfo {
  chassisId: string;
  portId: string;
  portDescription?: string;
  systemName?: string;
  systemDescription?: string;
  capabilities?: string[];
  managementAddress?: string;
}
export interface CDPDeviceInfo {
  deviceId: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  capabilities?: string[];
  managementAddress?: string;
  nativeVlan?: number;
  voiceVlan?: number;
}
export interface EDPDeviceInfo {
  deviceId: string;
  displayName?: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  vlan?: number;
}
export interface NDPDeviceInfo {
  linkLayerAddress: string;
  isRouter: boolean;
  reachableTime?: number;
  retransTimer?: number;
  flags?: number;
  lastAdvertisement?: string;
}
export interface DeviceProfile {
  profiledAt: string;
  openPorts?: OpenPort[];
  httpInfo?: HTTPInfo;
  snmpInfo?: SNMPInfo;
  mdnsServices?: MDNSService[];
  deviceType?: string;
  deviceIcons?: string[];
}
export interface OpenPort {
  port: number;
  protocol: string;
  service?: string;
  banner?: string;
  isOpen: boolean;
}
export interface HTTPInfo {
  port: number;
  statusCode: number;
  title?: string;
  server?: string;
  isHttps: boolean;
}
export interface SNMPInfo {
  sysDescr?: string;
  sysName?: string;
  sysContact?: string;
  sysLocation?: string;
}
export interface MDNSService {
  name: string;
  type: string;
  port?: number;
  txt?: {
    [k: string]: string;
  };
}
export interface SNMPFullData {
  collectedAt: string;
  system?: SystemInfo;
  interfaces?: SNMPInterface[];
  ipAddresses?: SNMPIPAddress[];
  macTable?: SNMPMACEntry[];
  vlans?: SNMPVLAN[];
  inventory?: SNMPEntity[];
  lldpNeighbors?: SNMPLLDPNeighbor[];
  routing?: SNMPRoute[];
  errors?: string[];
}
export interface SystemInfo {
  sysDescr: string;
  sysObjectId: string;
  sysName: string;
  sysContact: string;
  sysLocation: string;
  sysUpTime: number;
}
export interface SNMPInterface {
  index: number;
  name?: string;
  description?: string;
  alias?: string;
  type?: number;
  mtu?: number;
  speedMbps?: number;
  mac?: string;
  adminStatus?: string;
  operStatus?: string;
  inOctets?: number;
  outOctets?: number;
  inErrors?: number;
  outErrors?: number;
  inDiscards?: number;
  outDiscards?: number;
  lastUpdated?: number;
}
export interface SNMPIPAddress {
  address: string;
  prefix?: number;
  ifIndex: number;
  type?: string;
  addressIP?: string;
}
export interface SNMPMACEntry {
  mac: string;
  vlan?: number;
  ifIndex: number;
  type?: string;
  port?: string;
}
export interface SNMPVLAN {
  id: number;
  name?: string;
  status?: string;
  egressPorts?: number[];
  type?: string;
}
export interface SNMPEntity {
  index: number;
  description?: string;
  vendorType?: string;
  containedIn?: number;
  class?: string;
  parentRelPos?: number;
  name?: string;
  hardwareRev?: string;
  firmwareRev?: string;
  softwareRev?: string;
  serialNum?: string;
  mfgName?: string;
  modelName?: string;
  isFRU?: boolean;
}
export interface SNMPLLDPNeighbor {
  localIfIndex: number;
  localPortId?: string;
  remoteChassisId?: string;
  remotePortId?: string;
  remoteSysName?: string;
  remoteSysDescr?: string;
  remoteMgmtAddr?: string;
}
export interface SNMPRoute {
  destination: string;
  prefix?: number;
  nextHop?: string;
  ifIndex?: number;
  type?: string;
  protocol?: string;
  metric?: number;
}
export interface DeviceVulnerabilities {
  deviceIp: string;
  mac: string;
  hostname: string;
  vendor: string;
  product: string;
  version: string;
  vulnerabilities: Vulnerability[];
  scanTime: string;
  error?: string;
}
export interface Vulnerability {
  cveId: string;
  description: string;
  severity: string;
  score: number;
  published: string;
  modified: string;
  references: string[];
  affectedCpe: string;
  activelyExploited?: boolean;
  ransomwareRelated?: boolean;
  requiredAction?: string;
  dueDate?: string;
  originalSeverity?: string;
}
export interface WiFiPresence {
  ssid?: string;
  channel?: number;
  channelWidth?: number;
  frequencyMHz?: number;
  signalDbm?: number;
  isAccessPoint: boolean;
  isAuthorized: boolean;
  securityType?: string;
  band?: string;
  lastSeen: string;
}
export interface BluetoothPresence {
  name?: string;
  type: string;
  deviceClass?: string;
  rssi?: number;
  txPower?: number;
  isPaired: boolean;
  isConnected: boolean;
  isAuthorized: boolean;
  services?: string[];
  lastSeen: string;
}
export interface EngineStats {
  registry: RegistryStats;
  events: EventBusStats;
  scanCount: number;
  running: boolean;
  scanning: boolean;
}
export interface RegistryStats {
  totalDevices: number;
  wiredDevices: number;
  wifiDevices: number;
  btDevices: number;
  multiConnected: number;
  addCount: number;
  updateCount: number;
  removeCount: number;
  lastUpdate: string;
}
export interface EventBusStats {
  eventCount: number;
  subscriberCount: number;
  bufferedEvents: number;
  bufferSize: number;
}
export interface ScanResult {
  devices: DiscoveredDevice[];
  stats: ScanStats;
  phases: string[];
  scanType: string;
  startTime: string;
  endTime: string;
  duration: number;
  error?: string;
}
export interface ScanStats {
  totalDevices: number;
  wiredDevices: number;
  wifiDevices: number;
  bluetoothDevices: number;
  multiConnected: number;
  newDevices: number;
  updatedDevices: number;
  enrichedDevices: number;
  vulnerableDevices: number;
}
