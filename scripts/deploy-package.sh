#!/bin/bash
#
# Deploy The Seed via native packages (RPM/DEB)
# Detects target OS and builds/installs appropriate package
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

REMOTE_HOST="${1:-krisarmstrong@10.0.0.210}"
REMOTE_REPO_PATH="/home/krisarmstrong/seed"

echo "================================================"
echo " The Seed Package Deployment"
echo "================================================"
echo
echo "Target: ${REMOTE_HOST}"
echo

# Step 1: Detect target OS
echo -e "${YELLOW}Step 1: Detecting target OS...${NC}"
PKG_MANAGER=$(ssh "${REMOTE_HOST}" "command -v dnf >/dev/null && echo 'dnf' || (command -v apt >/dev/null && echo 'apt' || echo 'unknown')")

if [ "$PKG_MANAGER" = "dnf" ]; then
    PKG_TYPE="rpm"
    INSTALL_CMD="sudo dnf install -y"
    echo -e "${GREEN}✓ Detected RPM-based system (dnf)${NC}"
elif [ "$PKG_MANAGER" = "apt" ]; then
    PKG_TYPE="deb"
    INSTALL_CMD="sudo apt install -y"
    echo -e "${GREEN}✓ Detected DEB-based system (apt)${NC}"
else
    echo -e "${RED}✗ Unknown package manager${NC}"
    exit 1
fi
echo

# Step 2: Pull latest code on server
echo -e "${YELLOW}Step 2: Pulling latest code on server...${NC}"
ssh "${REMOTE_HOST}" "cd ${REMOTE_REPO_PATH} && git pull"
echo -e "${GREEN}✓ Code updated${NC}"
echo

# Step 3: Build package on server
echo -e "${YELLOW}Step 3: Building ${PKG_TYPE} package on server...${NC}"
ssh "${REMOTE_HOST}" "cd ${REMOTE_REPO_PATH} && make ${PKG_TYPE}"
echo -e "${GREEN}✓ Package built${NC}"
echo

# Step 4: Install package
echo -e "${YELLOW}Step 4: Installing package...${NC}"
PKG_FILE=$(ssh "${REMOTE_HOST}" "ls -t ${REMOTE_REPO_PATH}/dist/*.${PKG_TYPE} 2>/dev/null | head -1")
if [ -z "$PKG_FILE" ]; then
    echo -e "${RED}✗ No package found in dist/${NC}"
    exit 1
fi
echo "Installing: ${PKG_FILE}"
ssh "${REMOTE_HOST}" "${INSTALL_CMD} ${PKG_FILE}"
echo -e "${GREEN}✓ Package installed${NC}"
echo

# Step 5: Restart service
echo -e "${YELLOW}Step 5: Restarting service...${NC}"
ssh "${REMOTE_HOST}" "sudo systemctl daemon-reload"
ssh "${REMOTE_HOST}" "sudo systemctl restart seed"
sleep 3
echo -e "${GREEN}✓ Service restarted${NC}"
echo

# Step 6: Check status
echo -e "${YELLOW}Step 6: Checking status...${NC}"
ssh "${REMOTE_HOST}" "sudo systemctl status seed --no-pager -l" || true
echo

echo "================================================"
echo " Deployment Complete!"
echo "================================================"
echo
echo "Service status:"
ssh "${REMOTE_HOST}" "sudo systemctl is-active seed" && \
  echo -e "${GREEN}✓ Service is running${NC}" || \
  echo -e "${RED}✗ Service failed to start${NC}"
echo
echo "View logs: ssh ${REMOTE_HOST} sudo journalctl -u seed -f"
echo
