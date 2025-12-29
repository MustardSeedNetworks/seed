/**
 * HelpContent.tsx
 *
 * Purpose: Centralized help text constants for all application tooltips and help modals.
 * Provides user-facing documentation for networking concepts, metrics, and features.
 *
 * Key Features:
 * - HTTP timing metrics: DNS, TCP, TLS, Wait, Download explanations
 * - Link status help: Carrier, Speed, Duplex, AutoNeg, MTU documentation
 * - DNS operations: Forward/Reverse/IPv6 lookup and latency help
 * - WiFi metrics: Signal strength, channel, frequency, security explanations
 * - DHCP information: Lease timing, IP configuration help
 * - Performance metrics: Throughput, latency, jitter explanations
 * - Protocol help: TCP/UDP, ICMP, SNMP, iperf3 documentation
 * - Gateway information: Route destination and metric help
 *
 * Usage:
 * ```typescript
 * const httpHelp = HTTP_TIMING_HELP['dns'];
 * const linkHelp = LINK_HELP['duplex'];
 * ```
 *
 * Dependencies: None (pure data constants)
 * Structure: Record<string, string> for tooltip text lookup by metric name
 */

// Help text constants for tooltips and modal
export const HTTP_TIMING_HELP: Record<string, string> = {
  dns: "Time to resolve the hostname to an IP address via DNS lookup. Shows 0 when connection is reused from pool.",
  tcp: "Time for TCP 3-way handshake to establish connection. Shows 0 when using an existing connection.",
  tls: "TLS/SSL handshake time for certificate exchange and key negotiation. Only applies to HTTPS. Shows 0 when connection is reused.",
  wait: "Time to First Byte (TTFB) - time from request sent to first response byte received. Includes server processing time and network latency.",
  download: "Time to download the full response body after receiving the first byte.",
};

export const LINK_HELP: Record<string, string> = {
  carrier:
    "Physical layer signal detection. Shows 'Connected' when NIC detects a link partner (cable connected to active port).",
  speed: "Negotiated link speed between your network interface and the connected device.",
  duplex:
    "Communication mode - Full duplex allows simultaneous bidirectional data, Half duplex is one direction at a time.",
  autoNeg:
    "Whether speed and duplex were auto-negotiated with the link partner or manually configured.",
  advertised:
    "Link speeds this network interface advertises it can support during auto-negotiation.",
  mtu: "Maximum Transmission Unit - largest packet size (in bytes) that can be sent without fragmentation.",
};

export const DNS_HELP: Record<string, string> = {
  forwardLookup: "Resolves hostname to IPv4 address (A record).",
  reverseLookup: "Resolves IP address back to hostname (PTR record).",
  ipv6Lookup: "Resolves hostname to IPv6 address (AAAA record).",
  latency: "Time taken for the DNS query to complete.",
};

export const PERFORMANCE_HELP: Record<string, string> = {
  internetSpeed:
    "Tests download/upload speeds to public speedtest servers. Measures your connection to the internet.",
  lanSpeed: "Tests throughput on your local network using iperf3 to a configured server.",
  download: "Maximum download speed achieved during the test.",
  upload: "Maximum upload speed achieved during the test.",
  latency: "Round-trip time (ping) to the test server.",
  jitter: "Variation in latency over time. Lower is better for real-time applications.",
};

export const DISCOVERY_HELP: Record<string, string> = {
  networkScan: "Discovers active devices on your network subnet using ARP/ICMP.",
  mac: "Hardware address of the network interface - unique identifier for the device.",
  vendor: "Manufacturer identified from the MAC address OUI (first 3 bytes).",
  hostname: "DNS hostname if reverse lookup succeeded.",
};

export const CABLE_HELP: Record<string, string> = {
  tdrTest:
    "Time Domain Reflectometry measures cable length and detects faults by sending electrical pulses.",
  cableStatus:
    "Shows if cable pairs are OK, open (disconnected), short (wires touching), or have impedance mismatch.",
  faultDistance: "Distance to cable fault in meters. Helps locate physical cable problems.",
  pairs:
    "Ethernet cables have 4 twisted pairs. Gigabit uses all 4; Fast Ethernet uses pairs 1-2 and 3-6.",
};

export const WIFI_HELP: Record<string, string> = {
  ssid: "Service Set Identifier - the name of the wireless network you're connected to.",
  bssid: "Basic Service Set Identifier - MAC address of the access point.",
  signal: "Signal strength in dBm. -30 is excellent, -67 is good, -70 is fair, -80 is weak.",
  channel: "WiFi channel number (1-14 for 2.4GHz, 36-165 for 5GHz). Avoid overlapping channels.",
  security: "Encryption protocol protecting the connection (WPA2, WPA3, WEP, or Open).",
  frequency: "Radio band - 2.4GHz has better range, 5GHz has better speed and less interference.",
};

export const SWITCH_HELP: Record<string, string> = {
  protocol: "Discovery protocol used: LLDP (standard), CDP (Cisco), or EDP (Extreme).",
  switchName: "Hostname of the network switch this device is connected to.",
  port: "Physical switch port number where this device is plugged in.",
  portDescription: "Admin-configured label for the switch port.",
  managementIp: "IP address for accessing the switch's management interface.",
  nativeVlan: "Default VLAN for untagged traffic on this port.",
};

export const VLAN_HELP: Record<string, string> = {
  vlanId: "802.1Q VLAN tag number (1-4094). Segments network traffic logically.",
  interface: "Network interface associated with this VLAN.",
  parentInterface: "Physical interface that carries the tagged VLAN traffic.",
};

export const DHCP_HELP: Record<string, string> = {
  lease: "Duration of current IP address assignment before renewal is needed.",
  server: "DHCP server IP that issued the lease (usually your router).",
  gateway: "Default gateway assigned by DHCP for routing traffic off-subnet.",
  dns: "DNS servers assigned by DHCP for name resolution.",
  subnet: "Network mask defining the local subnet size.",
  domain: "DNS search domain for unqualified hostnames.",
};

export const GATEWAY_HELP: Record<string, string> = {
  ipv4Gateway: "Default router for IPv4 traffic leaving your local network.",
  ipv6Gateway: "Default router for IPv6 traffic (may be link-local address).",
  reachability: "Whether the gateway responds to ICMP ping requests.",
  latency: "Round-trip time to gateway. Should be <1ms for local networks.",
  packetLoss: "Percentage of ping packets that didn't receive a response.",
};

export const THRESHOLD_HELP: Record<string, string> = {
  dnsLookup:
    "Time to resolve hostnames. Good: <50ms for local DNS, <100ms for public DNS. Warning indicates potential resolver issues.",
  gatewayPing:
    "Round-trip time to your default gateway (router). Good: <20ms for local LAN. High latency suggests network congestion or equipment issues.",
  wifiSignal:
    "Signal strength in dBm (negative values). Good: >-50 (excellent), Warning: >-70 (good), below -70 is weak. Higher (less negative) is better.",
  healthCheckPing:
    "ICMP ping latency to configured targets. Adjust based on target distance - local servers should be <50ms, internet hosts may be higher.",
  healthCheckTcp:
    "TCP connection time to configured ports. Includes SYN-ACK round trip. Good: <100ms for local services.",
  httpTotal:
    "Total time for complete HTTP request including DNS, TCP, TLS, and response. Good: <500ms for simple pages.",
  httpDns:
    "DNS resolution phase of HTTP request. Should be fast if using local resolver or cached results.",
  httpTcp:
    "TCP handshake time within HTTP request. High values indicate network latency or congestion.",
  httpTls:
    "TLS/SSL handshake time. Includes certificate exchange and key negotiation. Affected by cipher strength.",
  httpTtfb:
    "Time to First Byte - server processing time. High values indicate slow backend or database queries.",
};

export const HARDWARE_HELP: Record<string, string> = {
  cableDiagnostics:
    "TDR (Time Domain Reflectometry) cable testing requires specific network interface hardware. NOT all NICs support this feature.",
  cableSupportedNics:
    "Intel I350/I210/I225-V, Broadcom BCM5719/5720. Most Realtek consumer NICs (RTL8111/8125) do NOT support TDR.",
  cableTesting:
    "Test if your NIC supports TDR: 'sudo ethtool --cable-test eth0'. If you see 'Operation not supported', your NIC cannot perform cable diagnostics.",
  wifiDiagnostics:
    "Advanced Wi-Fi diagnostics (monitor mode, site surveys) require nl80211-compatible chipsets. Built-in laptop Wi-Fi may have limited capabilities.",
  wifiRecommended:
    "Intel AX200/AX210 (Wi-Fi 6/6E), Atheros AR9271 (USB). Broadcom, Realtek have limited support. Apple Silicon built-in Wi-Fi not supported on Linux.",
  wifiCapabilities:
    "Monitor mode allows packet capture and site surveys. Injection enables advanced testing. Check chipset compatibility before purchasing adapters.",
  hardwareGuide:
    "See HARDWARE.md in the repository for complete compatibility matrix, testing procedures, and recommended hardware bundles by use case.",
};

export const HARDWARE_RECOMMENDATIONS = {
  cable: {
    supported: ["Intel I350", "Intel I210", "Intel I225-V", "Broadcom BCM5719", "Broadcom BCM5720"],
    notSupported: ["Realtek RTL8111", "Realtek RTL8125", "Most USB Ethernet adapters"],
    testCommand: "sudo ethtool --cable-test eth0",
  },
  wifi: {
    recommended: ["Intel AX200", "Intel AX210", "Atheros AR9271"],
    limited: ["Broadcom BCM43xx", "Realtek RTL88xx", "MediaTek MT7921"],
    notSupported: ["Apple Silicon built-in Wi-Fi"],
    testCommand: "iw list | grep -A 10 'Supported interface modes'",
  },
};
