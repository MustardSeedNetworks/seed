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
