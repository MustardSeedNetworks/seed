#!/bin/bash
#
# LuminetIQ Hardware Compatibility Test Script
#
# Purpose: Automatically test Wi-Fi and Ethernet hardware capabilities
# Usage: sudo ./scripts/test-hardware-compatibility.sh [interface]
#
# Example:
#   sudo ./scripts/test-hardware-compatibility.sh wlan0
#   sudo ./scripts/test-hardware-compatibility.sh eth0
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (sudo)${NC}"
    echo "Usage: sudo $0 [interface]"
    exit 1
fi

# Get interface from argument or detect
INTERFACE="${1:-}"
if [ -z "$INTERFACE" ]; then
    echo "No interface specified. Detecting available interfaces..."
    ip link show | grep -E "^[0-9]+: " | awk '{print $2}' | sed 's/://g' | grep -v "lo"
    echo ""
    read -p "Enter interface name to test: " INTERFACE
fi

# Verify interface exists
if ! ip link show "$INTERFACE" &>/dev/null; then
    echo -e "${RED}Error: Interface '$INTERFACE' not found${NC}"
    echo "Available interfaces:"
    ip link show | grep -E "^[0-9]+: " | awk '{print $2}' | sed 's/://g'
    exit 1
fi

echo -e "${BLUE}=== LuminetIQ Hardware Compatibility Test ===${NC}"
echo -e "Interface: ${GREEN}$INTERFACE${NC}"
echo -e "Date: $(date)"
echo -e "Kernel: $(uname -r)"
echo ""

# Determine if wireless or ethernet
IS_WIRELESS=0
if [ -d "/sys/class/net/$INTERFACE/wireless" ] || iw dev "$INTERFACE" info &>/dev/null; then
    IS_WIRELESS=1
fi

# System Information
echo -e "${BLUE}## System Information${NC}"
echo "Distribution: $(lsb_release -d 2>/dev/null | cut -f2 || cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo "Kernel: $(uname -r)"
echo "Architecture: $(uname -m)"
echo ""

# Hardware Information
echo -e "${BLUE}## Hardware Information${NC}"
if [ "$IS_WIRELESS" -eq 1 ]; then
    echo "Type: Wi-Fi Adapter"
    if lspci | grep -i "network\|wireless" | grep -q "$INTERFACE\|Wireless"; then
        echo "Bus: PCI/PCIe"
        lspci | grep -i "network\|wireless" | head -1
    elif lsusb | grep -i "wireless\|802.11\|wi-fi"; then
        echo "Bus: USB"
        lsusb | grep -i "wireless\|802.11\|wi-fi" | head -1
    fi
else
    echo "Type: Ethernet NIC"
    if lspci | grep -i "ethernet\|network" | grep -q "$INTERFACE\|Ethernet"; then
        echo "Bus: PCI/PCIe"
        lspci | grep -i "ethernet\|network" | head -1
    elif lsusb | grep -i "ethernet"; then
        echo "Bus: USB"
        lsusb | grep -i "ethernet" | head -1
    fi
fi
echo ""

# Driver Information
echo -e "${BLUE}## Driver Information${NC}"
if ethtool -i "$INTERFACE" &>/dev/null; then
    ethtool -i "$INTERFACE" | grep -E "driver|version|firmware"
else
    echo "ethtool not available or interface doesn't support it"
fi
echo ""

# Wi-Fi Specific Tests
if [ "$IS_WIRELESS" -eq 1 ]; then
    echo -e "${BLUE}## Wi-Fi Capabilities${NC}"

    # Check nl80211 support
    if command -v iw &>/dev/null; then
        echo -e "${GREEN}✓${NC} iw (nl80211) available"

        # Get interface info
        echo ""
        echo "Interface Info:"
        iw dev "$INTERFACE" info 2>/dev/null || echo "Could not get interface info"

        # Check supported modes
        echo ""
        echo "Supported Interface Modes:"
        MODES=$(iw list 2>/dev/null | sed -n '/Supported interface modes:/,/Band/p' | grep -E "\* " | sed 's/\* /  - /g')
        if echo "$MODES" | grep -q "monitor"; then
            echo -e "${GREEN}✓ Monitor mode supported${NC}"
        else
            echo -e "${RED}✗ Monitor mode NOT supported${NC}"
        fi
        echo "$MODES"

        # Test monitor mode switching
        echo ""
        echo "Testing Monitor Mode Switch..."
        ORIG_TYPE=$(iw dev "$INTERFACE" info | grep type | awk '{print $2}')
        ip link set "$INTERFACE" down
        if iw dev "$INTERFACE" set type monitor 2>/dev/null; then
            echo -e "${GREEN}✓ Successfully switched to monitor mode${NC}"
            # Switch back
            iw dev "$INTERFACE" set type "$ORIG_TYPE" 2>/dev/null
            ip link set "$INTERFACE" up
            echo -e "${GREEN}✓ Successfully switched back to $ORIG_TYPE mode${NC}"
        else
            echo -e "${RED}✗ Failed to switch to monitor mode${NC}"
            ip link set "$INTERFACE" up
        fi

        # Channel switching test
        echo ""
        echo "Testing Channel Switching (2.4GHz)..."
        ip link set "$INTERFACE" up
        CHANNEL_TEST_PASS=0
        for ch in 1 6 11; do
            if iw dev "$INTERFACE" set channel "$ch" 2>/dev/null; then
                CURRENT=$(iw dev "$INTERFACE" info | grep channel | awk '{print $2}')
                if [ "$CURRENT" = "$ch" ]; then
                    echo -e "${GREEN}✓${NC} Channel $ch: OK"
                    CHANNEL_TEST_PASS=$((CHANNEL_TEST_PASS + 1))
                else
                    echo -e "${YELLOW}⚠${NC} Channel $ch: Set but verification failed"
                fi
            else
                echo -e "${RED}✗${NC} Channel $ch: Failed to set"
            fi
            sleep 0.5
        done

        if [ "$CHANNEL_TEST_PASS" -eq 3 ]; then
            echo -e "${GREEN}✓ Channel switching: EXCELLENT${NC}"
        elif [ "$CHANNEL_TEST_PASS" -gt 0 ]; then
            echo -e "${YELLOW}⚠ Channel switching: PARTIAL${NC}"
        else
            echo -e "${RED}✗ Channel switching: FAILED${NC}"
        fi

        # Scan capability
        echo ""
        echo "Testing Network Scanning..."
        SCAN_COUNT=$(iw dev "$INTERFACE" scan 2>/dev/null | grep -c "^BSS " || echo "0")
        if [ "$SCAN_COUNT" -gt 0 ]; then
            echo -e "${GREEN}✓ Scan working: Found $SCAN_COUNT networks${NC}"

            # Signal quality test
            SIGNALS=$(iw dev "$INTERFACE" scan 2>/dev/null | grep "signal:" | awk '{print $2}' | head -5)
            if [ -n "$SIGNALS" ]; then
                echo -e "${GREEN}✓ Signal quality reporting: Working${NC}"
                echo "  Sample signals: $(echo $SIGNALS | tr '\n' ' ')"
            fi
        else
            echo -e "${YELLOW}⚠ Scan returned no results (may need to be connected)${NC}"
        fi

    else
        echo -e "${RED}✗ iw command not found (nl80211 not available)${NC}"
    fi
fi

# Ethernet Specific Tests
if [ "$IS_WIRELESS" -eq 0 ]; then
    echo -e "${BLUE}## Ethernet Capabilities${NC}"

    # Link status
    echo "Link Status:"
    if ethtool "$INTERFACE" &>/dev/null; then
        LINK=$(ethtool "$INTERFACE" | grep "Link detected:" | awk '{print $3}')
        SPEED=$(ethtool "$INTERFACE" | grep "Speed:" | awk '{print $2}')
        DUPLEX=$(ethtool "$INTERFACE" | grep "Duplex:" | awk '{print $2}')

        echo "  Link: $LINK"
        echo "  Speed: $SPEED"
        echo "  Duplex: $DUPLEX"
    fi
    echo ""

    # TDR Cable Testing Support
    echo "TDR Cable Testing Support:"
    if ethtool --cable-test "$INTERFACE" &>/dev/null; then
        echo -e "${GREEN}✓ TDR SUPPORTED${NC}"
        echo ""
        echo "Running cable test..."
        timeout 30 ethtool --cable-test "$INTERFACE" 2>&1 || echo "Test completed or timed out"
    else
        TDR_ERROR=$(ethtool --cable-test "$INTERFACE" 2>&1 || true)
        if echo "$TDR_ERROR" | grep -q "Operation not supported"; then
            echo -e "${RED}✗ TDR NOT SUPPORTED${NC}"
            echo "  Error: Operation not supported by this NIC"
        else
            echo -e "${YELLOW}⚠ TDR STATUS UNKNOWN${NC}"
            echo "  Error: $TDR_ERROR"
        fi
    fi
    echo ""

    # Advanced features
    echo "Advanced Features:"
    if ethtool -k "$INTERFACE" &>/dev/null; then
        TSO=$(ethtool -k "$INTERFACE" | grep "tcp-segmentation-offload:" | awk '{print $2}')
        GSO=$(ethtool -k "$INTERFACE" | grep "generic-segmentation-offload:" | awk '{print $2}')
        echo "  TSO (TCP Segmentation Offload): $TSO"
        echo "  GSO (Generic Segmentation Offload): $GSO"
    fi
fi

# Common Tests (both Wi-Fi and Ethernet)
echo ""
echo -e "${BLUE}## Common Capabilities${NC}"

# Packet capture support
echo "Packet Capture (tcpdump/libpcap):"
if command -v tcpdump &>/dev/null; then
    if timeout 2 tcpdump -i "$INTERFACE" -c 1 2>/dev/null >/dev/null; then
        echo -e "${GREEN}✓ Packet capture working${NC}"
    else
        echo -e "${YELLOW}⚠ Packet capture available but may need configuration${NC}"
    fi
else
    echo -e "${RED}✗ tcpdump not installed${NC}"
fi

# MTU
MTU=$(ip link show "$INTERFACE" | grep mtu | awk '{print $5}')
echo "MTU: $MTU"

# MAC address
MAC=$(ip link show "$INTERFACE" | grep ether | awk '{print $2}')
echo "MAC: $MAC"

# Summary
echo ""
echo -e "${BLUE}=== Summary ===${NC}"
echo "Interface: $INTERFACE"
if [ "$IS_WIRELESS" -eq 1 ]; then
    echo "Type: Wi-Fi Adapter"
    if iw list 2>/dev/null | grep -A 10 "Supported interface modes" | grep -q "monitor"; then
        echo -e "Monitor Mode: ${GREEN}✓ Supported${NC}"
    else
        echo -e "Monitor Mode: ${RED}✗ Not Supported${NC}"
    fi
    echo "Recommendation: See output above for detailed Wi-Fi capabilities"
else
    echo "Type: Ethernet NIC"
    if ethtool --cable-test "$INTERFACE" &>/dev/null; then
        echo -e "TDR Cable Testing: ${GREEN}✓ Supported${NC}"
        echo -e "Recommendation: ${GREEN}Excellent for cable diagnostics${NC}"
    else
        echo -e "TDR Cable Testing: ${RED}✗ Not Supported${NC}"
        echo -e "Recommendation: ${YELLOW}Basic diagnostics only. Consider Intel I350/I210 for TDR support.${NC}"
    fi
fi

echo ""
echo -e "${BLUE}=== Next Steps ===${NC}"
echo "1. Review the test results above"
echo "2. Submit a hardware report: https://github.com/krisarmstrong/luminetiq/issues/new?template=hardware-report.yml"
echo "3. See HARDWARE.md for recommended alternatives if hardware is not fully compatible"
echo ""
echo "Test completed at $(date)"
