/**
 * Standard terms that must NEVER be translated. Acronyms / RFC numbers /
 * protocol names / metric names / units / product+module names. Keep aligned
 * with the cross-repo memory: feedback_no_translate_standard_terms.
 *
 * Lives here rather than in a test file because two suites need the same
 * answer: i18n.parity.test.ts asserts these survive into `es`, and
 * locale-integrity.i18n.test.ts asserts everything else does *not*. Two copies
 * of this list would drift, and each would then be wrong in the other's
 * direction.
 */
export const DNT_TERMS = [
  // Standards
  'RFC 2544',
  'Y.1564',
  'Y.1731',
  'RFC 2889',
  'RFC 6349',
  'MEF',
  'TSN',
  // Protocols & acronyms
  'ARP',
  'DHCP',
  'DNS',
  'BGP',
  'OSPF',
  'SNMP',
  'VLAN',
  'WebSocket',
  // Metrics, abbreviations, units
  'SNR',
  'FLR',
  'FDV',
  'CIR',
  'EIR',
  'Mbps',
  'dBm',
  'jitter',
  'throughput',
  'latency',
  // Product names
  // Addressing and link layer
  'MAC',
  'IP',
  'IPv4',
  'IPv6',
  'CIDR',
  'MTU',
  'LLDP',
  'CDP',
  'NDP',
  'PoE',
  'SFP',
  // Transport and application protocols
  'ICMP',
  'TCP',
  'UDP',
  'HTTP',
  'HTTPS',
  'TLS',
  'SSH',
  'NTP',
  'RTSP',
  'mDNS',
  // Wi-Fi
  'SSID',
  'BSSID',
  'RSSI',
  'WPA',
  'WPA2',
  'WPA3',
  // seed#2296: the tree used to spell it both "Wi-Fi" and "WiFi", so neither
  // could be asserted as canonical. Every locale *value* is now "Wi-Fi" — the
  // Wi-Fi Alliance's own spelling — and the term is guarded like any other.
  // Identifiers (wifiSettings, isWifi, WiFiCard) are code, not copy, and are
  // deliberately untouched.
  'Wi-Fi',
  // Quality of service
  'QoS',
  'DSCP',
  'CoS',
  // Units
  'Gbps',
  'Kbps',
  'GHz',
  'MHz',
  'ms',
  // Products
  'Seed',
  'Stem',
  'NIAC',
];
