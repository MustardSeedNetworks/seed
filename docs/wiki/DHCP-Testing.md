# Rogue DHCP Server Test Environment

This guide explains how to set up a test environment to validate The Seed's Rogue DHCP detection
feature.

## Overview

The Rogue DHCP detection feature monitors DHCP OFFER packets on the network and alerts when
unauthorized DHCP servers are detected. This test environment allows you to simulate both legitimate
and rogue DHCP servers.

## Prerequisites

- Two systems on the same network:
  - **System A**: Running The Seed (Ubuntu server at 192.168.64.7)
  - **System B**: Test machine to run rogue DHCP server
- Root/sudo access on both systems
- `dnsmasq` or `isc-dhcp-server` package

## Test Scenarios

### Scenario 1: Single Authorized DHCP Server (Baseline)

**Expected Result**: No alerts

1. Configure The Seed to know about your legitimate DHCP server:

   ```yaml
   # seed.yaml
   dhcp:
     rogue_detection:
       enabled: true
       known_servers:
         - 192.168.1.1 # Your router's IP
       alert_on_detection: true
   ```

2. Restart The Seed and verify no alerts appear

### Scenario 2: Rogue DHCP Server Detection

**Expected Result**: Alert generated for rogue server

#### Option A: Using dnsmasq (Recommended for Testing)

1. Install dnsmasq on test machine:

   ```bash
   sudo apt-get update
   sudo apt-get install dnsmasq
   ```

2. Stop system's NetworkManager DHCP to avoid conflicts:

   ```bash
   sudo systemctl stop NetworkManager
   # Or just disable its DHCP:
   sudo nmcli connection modify "YourConnection" ipv4.method manual
   ```

3. Create test dnsmasq config (`/etc/dnsmasq-test.conf`):

   ```conf
   # Interface to listen on
   interface=eth0  # Change to your interface

   # DHCP range
   dhcp-range=192.168.64.100,192.168.64.200,12h

   # Don't read /etc/hosts
   no-hosts

   # Don't read /etc/resolv.conf
   no-resolv

   # Provide DNS
   server=8.8.8.8

   # DHCP server identifier
   dhcp-authoritative
   ```

4. Start rogue DHCP server:

   ```bash
   sudo dnsmasq -C /etc/dnsmasq-test.conf -d
   # -d flag keeps it in foreground for easy stopping
   ```

5. Verify it's sending DHCP OFFERs:

   ```bash
   # On The Seed server
   sudo tcpdump -i enp0s1 -n port 67 or port 68
   ```

6. Trigger DHCP discovery (on any client machine):

   ```bash
   sudo dhclient -r eth0  # Release current lease
   sudo dhclient eth0     # Request new lease
   ```

7. Check The Seed web UI or API for rogue DHCP alert:

   ```bash
   curl -k -H "Authorization: Bearer YOUR_TOKEN" \
     https://192.168.64.7:8443/api/dhcp/rogue
   ```

8. Stop rogue server:
   ```bash
   # Press Ctrl+C in dnsmasq terminal
   sudo systemctl start NetworkManager  # Restore network
   ```

#### Option B: Using isc-dhcp-server

1. Install ISC DHCP server:

   ```bash
   sudo apt-get update
   sudo apt-get install isc-dhcp-server
   ```

2. Configure `/etc/dhcp/dhcpd.conf`:

   ```conf
   # Test DHCP configuration
   default-lease-time 600;
   max-lease-time 7200;
   authoritative;

   subnet 192.168.64.0 netmask 255.255.255.0 {
     range 192.168.64.100 192.168.64.200;
     option routers 192.168.64.1;
     option domain-name-servers 8.8.8.8, 8.8.4.4;
   }
   ```

3. Configure interface in `/etc/default/isc-dhcp-server`:

   ```bash
   INTERFACESv4="eth0"  # Change to your interface
   ```

4. Start server:

   ```bash
   sudo systemctl start isc-dhcp-server
   sudo systemctl status isc-dhcp-server
   ```

5. Verify and test same as Option A steps 5-7

6. Stop server:
   ```bash
   sudo systemctl stop isc-dhcp-server
   ```

### Scenario 3: Multiple DHCP Servers (Mixed)

**Expected Result**: Alert only for unknown server

1. Configure The Seed to know about ONE server:

   ```yaml
   dhcp:
     rogue_detection:
       known_servers:
         - 192.168.64.1 # Legitimate router
   ```

2. Start rogue server on 192.168.64.50 (different IP)

3. Both servers will respond to DHCP requests

4. Verify:
   - No alert for 192.168.64.1
   - Alert for 192.168.64.50

### Scenario 4: Adding Rogue to Known List

**Expected Result**: Alerts stop after adding to known list

1. Start with rogue server running and alerts firing

2. Add rogue server to known list via API:

   ```bash
   curl -k -X POST \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"server": "192.168.64.50", "description": "Test server"}' \
     https://192.168.64.7:8443/api/dhcp/rogue/servers
   ```

3. Verify alerts stop

4. Remove from known list:

   ```bash
   curl -k -X DELETE \
     -H "Authorization: Bearer YOUR_TOKEN" \
     "https://192.168.64.7:8443/api/dhcp/rogue/servers?server=192.168.64.50"
   ```

5. Verify alerts resume

## Network Topology for Testing

```
┌─────────────────────────────────────────────┐
│             Test Network (192.168.64.0/24)   │
├─────────────────────────────────────────────┤
│                                             │
│  ┌──────────────┐      ┌─────────────────┐ │
│  │  Router      │      │  The Seed      │ │
│  │  192.168.64.1│      │  192.168.64.7   │ │
│  │  (Legit DHCP)│      │  (Monitoring)   │ │
│  └──────────────┘      └─────────────────┘ │
│         │                       │           │
│         └───────────┬───────────┘           │
│                     │                       │
│          ┌──────────┴───────────┐           │
│          │   Network Switch     │           │
│          └──────────┬───────────┘           │
│                     │                       │
│          ┌──────────┴───────────┐           │
│          │  Test Machine        │           │
│          │  (Rogue DHCP)        │           │
│          │  192.168.64.50       │           │
│          └──────────────────────┘           │
│                                             │
└─────────────────────────────────────────────┘
```

## Packet Capture for Verification

To see DHCP traffic and verify detection:

```bash
# On The Seed server
sudo tcpdump -i enp0s1 -vvv -n \
  '(port 67 or port 68)' \
  -w dhcp_capture.pcap

# Analysis
sudo tcpdump -r dhcp_capture.pcap -n -vvv | grep -i "DHCP-Message Option 53"
```

## Expected DHCP Packet Flow

1. **DHCP DISCOVER** (Client → Broadcast)
   - Source: 0.0.0.0:68
   - Dest: 255.255.255.255:67

2. **DHCP OFFER** (Server → Client) **← Detected here**
   - Source: DHCP_SERVER_IP:67
   - Contains: Server Identifier Option (option 54)
   - The Seed extracts this IP and checks against known list

3. **DHCP REQUEST** (Client → Broadcast)
   - Client accepts one offer

4. **DHCP ACK** (Server → Client)
   - Confirms lease

## Troubleshooting

### No Packets Detected

```bash
# Verify The Seed has raw socket capability
getcap /home/krisarmstrong/seed/seed
# Should show: cap_net_raw+ep

# Check rogue detector is running
curl -k -H "Authorization: Bearer TOKEN" \
  https://192.168.64.7:8443/api/dhcp/rogue/config
```

### Firewall Blocking DHCP

```bash
# Allow DHCP on test machine
sudo ufw allow 67/udp
sudo ufw allow 68/udp

# Or disable firewall temporarily
sudo ufw disable
```

### Multiple Network Interfaces

If test machine has multiple interfaces, bind dnsmasq to specific interface:

```bash
sudo dnsmasq -C /etc/dnsmasq-test.conf \
  --interface=eth0 \
  --bind-interfaces \
  -d
```

## Cleanup

After testing:

```bash
# Stop test DHCP servers
sudo systemctl stop isc-dhcp-server
sudo killall dnsmasq

# Restore NetworkManager
sudo systemctl start NetworkManager

# Clear rogue DHCP alerts in The Seed
# (via web UI or API)

# Remove test configs
sudo rm /etc/dnsmasq-test.conf
```

## Automated Test Script

See `scripts/test-dhcp-rogue.sh` for automated testing.

## API Reference

### Get Rogue DHCP Status

```bash
GET /api/dhcp/rogue
```

### Get Known Servers

```bash
GET /api/dhcp/rogue/servers
```

### Add Known Server

```bash
POST /api/dhcp/rogue/servers
{
  "server": "192.168.64.1",
  "description": "Main router"
}
```

### Remove Known Server

```bash
DELETE /api/dhcp/rogue/servers?server=192.168.64.1
```

### Update Configuration

```bash
PUT /api/dhcp/rogue/config
{
  "enabled": true,
  "alert_on_detection": true,
  "known_servers": ["192.168.64.1"]
}
```

## Security Considerations

⚠️ **WARNING**: Running rogue DHCP servers can disrupt network connectivity for all devices on the
network. Only perform these tests on:

- Isolated test networks
- Networks you control
- During maintenance windows
- With proper authorization

Running unauthorized DHCP servers on corporate or public networks may violate policies or laws.

## Further Reading

- [RFC 2131 - DHCP](https://tools.ietf.org/html/rfc2131)
- [RFC 2132 - DHCP Options](https://tools.ietf.org/html/rfc2132)
- [DHCP Security Best Practices](https://www.cisco.com/c/en/us/support/docs/ip/dynamic-address-allocation-resolution/13670-18.html)
