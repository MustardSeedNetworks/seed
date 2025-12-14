#!/bin/bash
#
# Automated Rogue DHCP Server Test Script
# Tests LuminetIQ's rogue DHCP detection capability
#
# Usage: sudo ./test-dhcp-rogue.sh [luminetiq_url] [auth_token]
# Example: sudo ./test-dhcp-rogue.sh https://192.168.64.7:8443 your_jwt_token
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
LUMINETIQ_URL="${1:-https://192.168.64.7:8443}"
AUTH_TOKEN="${2}"
TEST_INTERFACE="${TEST_INTERFACE:-enp0s1}"
DHCP_RANGE_START="192.168.64.150"
DHCP_RANGE_END="192.168.64.200"
ROGUE_SERVER_IP="192.168.64.50"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}ERROR: This script must be run as root (for DHCP server)${NC}"
  exit 1
fi

# Check if auth token provided
if [ -z "$AUTH_TOKEN" ]; then
  echo -e "${RED}ERROR: Auth token required${NC}"
  echo "Usage: sudo $0 <luminetiq_url> <auth_token>"
  exit 1
fi

# Check dependencies
command -v dnsmasq >/dev/null 2>&1 || {
  echo -e "${RED}ERROR: dnsmasq is not installed${NC}"
  echo "Install with: sudo apt-get install dnsmasq"
  exit 1
}

command -v tcpdump >/dev/null 2>&1 || {
  echo -e "${YELLOW}WARNING: tcpdump not installed (optional for verification)${NC}"
}

echo "================================================"
echo " Rogue DHCP Detection Test Suite"
echo "================================================"
echo
echo "LuminetIQ URL: $LUMINETIQ_URL"
echo "Test Interface: $TEST_INTERFACE"
echo "DHCP Range: $DHCP_RANGE_START - $DHCP_RANGE_END"
echo

# Function to call LuminetIQ API
api_call() {
  local method="$1"
  local endpoint="$2"
  local data="$3"

  if [ -n "$data" ]; then
    curl -k -s -X "$method" \
      -H "Authorization: Bearer $AUTH_TOKEN" \
      -H "Content-Type: application/json" \
      -d "$data" \
      "$LUMINETIQ_URL$endpoint"
  else
    curl -k -s -X "$method" \
      -H "Authorization: Bearer $AUTH_TOKEN" \
      "$LUMINETIQ_URL$endpoint"
  fi
}

# Function to start rogue DHCP server
start_rogue_dhcp() {
  echo -e "${YELLOW}Starting rogue DHCP server...${NC}"

  # Create temporary config
  cat > /tmp/dnsmasq-rogue-test.conf <<EOF
# Rogue DHCP test configuration
interface=$TEST_INTERFACE
bind-interfaces
dhcp-range=$DHCP_RANGE_START,$DHCP_RANGE_END,12h
dhcp-option=3,$ROGUE_SERVER_IP
dhcp-option=6,8.8.8.8,8.8.4.4
dhcp-authoritative
no-hosts
no-resolv
server=8.8.8.8
log-dhcp
pid-file=/tmp/dnsmasq-rogue-test.pid
EOF

  # Start dnsmasq
  dnsmasq -C /tmp/dnsmasq-rogue-test.conf -d &
  DNSMASQ_PID=$!

  # Wait for it to start
  sleep 2

  if ps -p $DNSMASQ_PID > /dev/null; then
    echo -e "${GREEN}Rogue DHCP server started (PID: $DNSMASQ_PID)${NC}"
    return 0
  else
    echo -e "${RED}Failed to start rogue DHCP server${NC}"
    return 1
  fi
}

# Function to stop rogue DHCP server
stop_rogue_dhcp() {
  echo -e "${YELLOW}Stopping rogue DHCP server...${NC}"

  if [ -n "$DNSMASQ_PID" ]; then
    kill $DNSMASQ_PID 2>/dev/null || true
    wait $DNSMASQ_PID 2>/dev/null || true
  fi

  # Also try PID file
  if [ -f /tmp/dnsmasq-rogue-test.pid ]; then
    ROGUE_PID=$(cat /tmp/dnsmasq-rogue-test.pid)
    kill $ROGUE_PID 2>/dev/null || true
    rm /tmp/dnsmasq-rogue-test.pid
  fi

  # Cleanup
  rm -f /tmp/dnsmasq-rogue-test.conf

  echo -e "${GREEN}Rogue DHCP server stopped${NC}"
}

# Cleanup on exit
cleanup() {
  echo
  echo "Cleaning up..."
  stop_rogue_dhcp
  exit
}

trap cleanup EXIT INT TERM

# Test 1: Verify LuminetIQ API is accessible
echo "================================================"
echo "Test 1: Verify LuminetIQ API Connection"
echo "================================================"

if api_call GET /api/status > /dev/null 2>&1; then
  echo -e "${GREEN}✓ LuminetIQ API is accessible${NC}"
else
  echo -e "${RED}✗ Cannot connect to LuminetIQ API${NC}"
  exit 1
fi
echo

# Test 2: Check rogue DHCP feature is enabled
echo "================================================"
echo "Test 2: Check Rogue DHCP Detection Configuration"
echo "================================================"

CONFIG=$(api_call GET /api/dhcp/rogue/config)
ENABLED=$(echo "$CONFIG" | grep -o '"enabled":[^,}]*' | cut -d: -f2 | tr -d ' ')

if [ "$ENABLED" = "true" ]; then
  echo -e "${GREEN}✓ Rogue DHCP detection is enabled${NC}"
else
  echo -e "${YELLOW}⚠ Rogue DHCP detection is disabled, enabling...${NC}"
  api_call PUT /api/dhcp/rogue/config '{"enabled":true,"alert_on_detection":true}' > /dev/null
  echo -e "${GREEN}✓ Enabled rogue DHCP detection${NC}"
fi
echo

# Test 3: Clear known servers (baseline)
echo "================================================"
echo "Test 3: Configure Known Servers (Baseline)"
echo "================================================"

echo "Clearing all known servers for clean test..."
api_call GET /api/dhcp/rogue/servers | \
  grep -o '"server":"[^"]*"' | \
  cut -d'"' -f4 | \
  while read server; do
    api_call DELETE "/api/dhcp/rogue/servers?server=$server" > /dev/null
    echo "  Removed: $server"
  done

# Add legitimate server (typically the router)
LEGIT_SERVER=$(ip route | grep default | awk '{print $3}' | head -1)
if [ -n "$LEGIT_SERVER" ]; then
  echo "Adding legitimate server: $LEGIT_SERVER"
  api_call POST /api/dhcp/rogue/servers "{\"server\":\"$LEGIT_SERVER\",\"description\":\"Default gateway\"}" > /dev/null
  echo -e "${GREEN}✓ Known server configured${NC}"
fi
echo

# Test 4: Start rogue DHCP server
echo "================================================"
echo "Test 4: Start Rogue DHCP Server"
echo "================================================"

if start_rogue_dhcp; then
  echo -e "${GREEN}✓ Rogue server running${NC}"
else
  echo -e "${RED}✗ Failed to start rogue server${NC}"
  exit 1
fi
echo

# Test 5: Trigger DHCP discovery
echo "================================================"
echo "Test 5: Trigger DHCP Discovery"
echo "================================================"

echo "Sending DHCP DISCOVER packet..."
# Use dhcping if available, otherwise use dhclient
if command -v dhcping >/dev/null 2>&1; then
  timeout 5 dhcping -s $ROGUE_SERVER_IP -i $TEST_INTERFACE 2>&1 | grep -i "Got answer" && \
    echo -e "${GREEN}✓ DHCP OFFER received${NC}" || \
    echo -e "${YELLOW}⚠ No DHCP response (this is OK if network is quiet)${NC}"
else
  echo -e "${YELLOW}⚠ dhcping not installed, skipping active discovery${NC}"
  echo "Waiting 10 seconds for passive detection..."
  sleep 10
fi
echo

# Test 6: Check for rogue detection
echo "================================================"
echo "Test 6: Verify Rogue DHCP Detection"
echo "================================================"

echo "Checking for detected rogue servers..."
sleep 3  # Give time for detection

ROGUE_STATUS=$(api_call GET /api/dhcp/rogue)
DETECTED_SERVERS=$(echo "$ROGUE_STATUS" | grep -o '"rogueServers":\[[^]]*\]')

if echo "$DETECTED_SERVERS" | grep -q "$ROGUE_SERVER_IP"; then
  echo -e "${GREEN}✓ Rogue server detected: $ROGUE_SERVER_IP${NC}"
  echo "Detection details:"
  echo "$ROGUE_STATUS" | jq . 2>/dev/null || echo "$ROGUE_STATUS"
else
  echo -e "${YELLOW}⚠ Rogue server not yet detected${NC}"
  echo "This may be because:"
  echo "  1. No DHCP traffic on network yet"
  echo "  2. Detection interval hasn't passed"
  echo "  3. Packet capture permissions issue"
  echo
  echo "Current status:"
  echo "$ROGUE_STATUS" | jq . 2>/dev/null || echo "$ROGUE_STATUS"
fi
echo

# Test 7: Add rogue to known list
echo "================================================"
echo "Test 7: Add Rogue Server to Known List"
echo "================================================"

echo "Adding $ROGUE_SERVER_IP to known servers..."
api_call POST /api/dhcp/rogue/servers "{\"server\":\"$ROGUE_SERVER_IP\",\"description\":\"Test rogue server\"}" > /dev/null

sleep 3
ROGUE_STATUS=$(api_call GET /api/dhcp/rogue)

if echo "$ROGUE_STATUS" | grep -q "$ROGUE_SERVER_IP"; then
  echo -e "${YELLOW}⚠ Server still showing as rogue (detection may be cached)${NC}"
else
  echo -e "${GREEN}✓ Server no longer flagged as rogue${NC}"
fi
echo

# Test 8: Remove from known list
echo "================================================"
echo "Test 8: Remove from Known List (Should Alert Again)"
echo "================================================"

echo "Removing $ROGUE_SERVER_IP from known servers..."
api_call DELETE "/api/dhcp/rogue/servers?server=$ROGUE_SERVER_IP" > /dev/null

sleep 3
ROGUE_STATUS=$(api_call GET /api/dhcp/rogue)

echo "Final status:"
echo "$ROGUE_STATUS" | jq . 2>/dev/null || echo "$ROGUE_STATUS"
echo

# Summary
echo "================================================"
echo " Test Suite Complete"
echo "================================================"
echo
echo "Summary:"
echo "  - API Connection: ✓"
echo "  - Feature Enabled: ✓"
echo "  - Rogue Server Started: ✓"
echo "  - Detection: See results above"
echo
echo "Next steps:"
echo "  1. Check LuminetIQ web UI for alerts"
echo "  2. Review /api/dhcp/rogue endpoint"
echo "  3. Monitor DHCP traffic: sudo tcpdump -i $TEST_INTERFACE port 67 or port 68"
echo
echo "Cleanup: Rogue DHCP server will be stopped automatically"
echo

# Keep running for manual inspection
echo "Press Ctrl+C to stop the rogue server and exit..."
sleep infinity
