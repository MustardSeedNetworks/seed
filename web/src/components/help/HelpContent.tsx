// Help text constants for tooltips and modal

export const HTTP_TIMING_HELP: Record<string, string> = {
  DNS: "Time to resolve the hostname to an IP address via DNS lookup. Shows 0 when connection is reused from pool.",
  TCP: "Time for TCP 3-way handshake to establish connection. Shows 0 when using an existing connection.",
  TLS: "TLS/SSL handshake time for certificate exchange and key negotiation. Only applies to HTTPS. Shows 0 when connection is reused.",
  Wait: "Time to First Byte (TTFB) - time from request sent to first response byte received. Includes server processing time and network latency.",
  Download:
    "Time to download the full response body after receiving the first byte.",
};

export const LINK_HELP: Record<string, string> = {
  Carrier:
    "Physical layer signal detection. Shows 'Connected' when NIC detects a link partner (cable connected to active port).",
  Speed:
    "Negotiated link speed between your network interface and the connected device.",
  Duplex:
    "Communication mode - Full duplex allows simultaneous bidirectional data, Half duplex is one direction at a time.",
  AutoNeg:
    "Whether speed and duplex were auto-negotiated with the link partner or manually configured.",
  Advertised:
    "Link speeds this network interface advertises it can support during auto-negotiation.",
  MTU: "Maximum Transmission Unit - largest packet size (in bytes) that can be sent without fragmentation.",
};

export const DNS_HELP: Record<string, string> = {
  "Forward Lookup": "Resolves hostname to IPv4 address (A record).",
  "Reverse Lookup": "Resolves IP address back to hostname (PTR record).",
  "IPv6 Lookup": "Resolves hostname to IPv6 address (AAAA record).",
  Latency: "Time taken for the DNS query to complete.",
};

export const PERFORMANCE_HELP: Record<string, string> = {
  "Internet Speed":
    "Tests download/upload speeds to public speedtest servers. Measures your connection to the internet.",
  "LAN Speed":
    "Tests throughput on your local network using iperf3 to a configured server.",
  Download: "Maximum download speed achieved during the test.",
  Upload: "Maximum upload speed achieved during the test.",
  Latency: "Round-trip time (ping) to the test server.",
  Jitter:
    "Variation in latency over time. Lower is better for real-time applications.",
};

export const DISCOVERY_HELP: Record<string, string> = {
  "Network Scan":
    "Discovers active devices on your network subnet using ARP/ICMP.",
  MAC: "Hardware address of the network interface - unique identifier for the device.",
  Vendor: "Manufacturer identified from the MAC address OUI (first 3 bytes).",
  Hostname: "DNS hostname if reverse lookup succeeded.",
};

export const CABLE_HELP: Record<string, string> = {
  "TDR Test":
    "Time Domain Reflectometry measures cable length and detects faults by sending electrical pulses.",
  "Cable Status":
    "Shows if cable pairs are OK, open (disconnected), short (wires touching), or have impedance mismatch.",
  "Fault Distance":
    "Distance to cable fault in meters. Helps locate physical cable problems.",
  Pairs:
    "Ethernet cables have 4 twisted pairs. Gigabit uses all 4; Fast Ethernet uses pairs 1-2 and 3-6.",
};

export const WIFI_HELP: Record<string, string> = {
  SSID: "Service Set Identifier - the name of the wireless network you're connected to.",
  BSSID: "Basic Service Set Identifier - MAC address of the access point.",
  Signal:
    "Signal strength in dBm. -30 is excellent, -67 is good, -70 is fair, -80 is weak.",
  Channel:
    "WiFi channel number (1-14 for 2.4GHz, 36-165 for 5GHz). Avoid overlapping channels.",
  Security:
    "Encryption protocol protecting the connection (WPA2, WPA3, WEP, or Open).",
  Frequency:
    "Radio band - 2.4GHz has better range, 5GHz has better speed and less interference.",
};

export const SWITCH_HELP: Record<string, string> = {
  Protocol:
    "Discovery protocol used: LLDP (standard), CDP (Cisco), or EDP (Extreme).",
  "Switch Name": "Hostname of the network switch this device is connected to.",
  Port: "Physical switch port number where this device is plugged in.",
  "Port Description": "Admin-configured label for the switch port.",
  "Management IP":
    "IP address for accessing the switch's management interface.",
  "Native VLAN": "Default VLAN for untagged traffic on this port.",
};

export const VLAN_HELP: Record<string, string> = {
  "VLAN ID":
    "802.1Q VLAN tag number (1-4094). Segments network traffic logically.",
  Interface: "Network interface associated with this VLAN.",
  "Parent Interface":
    "Physical interface that carries the tagged VLAN traffic.",
};

export const DHCP_HELP: Record<string, string> = {
  Lease: "Duration of current IP address assignment before renewal is needed.",
  Server: "DHCP server IP that issued the lease (usually your router).",
  Gateway: "Default gateway assigned by DHCP for routing traffic off-subnet.",
  DNS: "DNS servers assigned by DHCP for name resolution.",
  Subnet: "Network mask defining the local subnet size.",
  Domain: "DNS search domain for unqualified hostnames.",
};

export const GATEWAY_HELP: Record<string, string> = {
  "IPv4 Gateway": "Default router for IPv4 traffic leaving your local network.",
  "IPv6 Gateway":
    "Default router for IPv6 traffic (may be link-local address).",
  Reachability: "Whether the gateway responds to ICMP ping requests.",
  Latency: "Round-trip time to gateway. Should be <1ms for local networks.",
  "Packet Loss": "Percentage of ping packets that didn't receive a response.",
};

export const THRESHOLD_HELP: Record<string, string> = {
  "DNS Lookup":
    "Time to resolve hostnames. Good: <50ms for local DNS, <100ms for public DNS. Warning indicates potential resolver issues.",
  "Gateway Ping":
    "Round-trip time to your default gateway (router). Good: <20ms for local LAN. High latency suggests network congestion or equipment issues.",
  "Wi-Fi Signal":
    "Signal strength in dBm (negative values). Good: >-50 (excellent), Warning: >-70 (good), below -70 is weak. Higher (less negative) is better.",
  "Health Check: Ping":
    "ICMP ping latency to configured targets. Adjust based on target distance - local servers should be <50ms, internet hosts may be higher.",
  "Health Check: TCP":
    "TCP connection time to configured ports. Includes SYN-ACK round trip. Good: <100ms for local services.",
  "HTTP Total":
    "Total time for complete HTTP request including DNS, TCP, TLS, and response. Good: <500ms for simple pages.",
  "HTTP DNS":
    "DNS resolution phase of HTTP request. Should be fast if using local resolver or cached results.",
  "HTTP TCP":
    "TCP handshake time within HTTP request. High values indicate network latency or congestion.",
  "HTTP TLS":
    "TLS/SSL handshake time. Includes certificate exchange and key negotiation. Affected by cipher strength.",
  "HTTP TTFB":
    "Time to First Byte - server processing time. High values indicate slow backend or database queries.",
};

export const HARDWARE_HELP: Record<string, string> = {
  "Cable Diagnostics":
    "TDR (Time Domain Reflectometry) cable testing requires specific network interface hardware. NOT all NICs support this feature.",
  "Cable - Supported NICs":
    "✅ Intel I350/I210/I225-V, Broadcom BCM5719/5720. ❌ Most Realtek consumer NICs (RTL8111/8125) do NOT support TDR.",
  "Cable - Testing":
    "Test if your NIC supports TDR: 'sudo ethtool --cable-test eth0'. If you see 'Operation not supported', your NIC cannot perform cable diagnostics.",
  "WiFi Diagnostics":
    "Advanced Wi-Fi diagnostics (monitor mode, site surveys) require nl80211-compatible chipsets. Built-in laptop Wi-Fi may have limited capabilities.",
  "WiFi - Recommended":
    "✅ Intel AX200/AX210 (Wi-Fi 6/6E), Atheros AR9271 (USB). ⚠️ Broadcom, Realtek have limited support. ❌ Apple Silicon built-in Wi-Fi not supported on Linux.",
  "WiFi - Capabilities":
    "Monitor mode allows packet capture and site surveys. Injection enables advanced testing. Check chipset compatibility before purchasing adapters.",
  "Hardware Guide":
    "See HARDWARE.md in the repository for complete compatibility matrix, testing procedures, and recommended hardware bundles by use case.",
};

export const HARDWARE_RECOMMENDATIONS = {
  cable: {
    supported: [
      "Intel I350",
      "Intel I210",
      "Intel I225-V",
      "Broadcom BCM5719",
      "Broadcom BCM5720",
    ],
    notSupported: [
      "Realtek RTL8111",
      "Realtek RTL8125",
      "Most USB Ethernet adapters",
    ],
    testCommand: "sudo ethtool --cable-test eth0",
  },
  wifi: {
    recommended: ["Intel AX200", "Intel AX210", "Atheros AR9271"],
    limited: ["Broadcom BCM43xx", "Realtek RTL88xx", "MediaTek MT7921"],
    notSupported: ["Apple Silicon built-in Wi-Fi"],
    testCommand: "iw list | grep -A 10 'Supported interface modes'",
  },
};
