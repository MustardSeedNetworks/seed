#!/bin/bash
#
# Deploy The Seed as systemd service
# Kills any manual runs and installs/starts systemd service
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

REMOTE_HOST="${1:-krisarmstrong@192.168.64.7}"
APP_DIR="/home/krisarmstrong/seed"
SERVICE_NAME="seed"

echo "================================================"
echo " The Seed Systemd Deployment"
echo "================================================"
echo

echo -e "${YELLOW}Step 1: Building frontend...${NC}"
cd web && npm run build && cd ..
echo -e "${GREEN}✓ Frontend built${NC}"
echo

echo -e "${YELLOW}Step 2: Syncing files to server...${NC}"
rsync -avz --exclude 'web/node_modules' --exclude '.git' \
  ./ "${REMOTE_HOST}:${APP_DIR}/"
echo -e "${GREEN}✓ Files synced${NC}"
echo

echo -e "${YELLOW}Step 3: Building on server...${NC}"
ssh "${REMOTE_HOST}" "cd ${APP_DIR} && go build -o seed ./cmd/seed"
echo -e "${GREEN}✓ Built on server${NC}"
echo

echo -e "${YELLOW}Step 4: Killing any manual runs...${NC}"
ssh "${REMOTE_HOST}" "pkill -9 seed || true"
sleep 2
echo -e "${GREEN}✓ Manual processes stopped${NC}"
echo

echo -e "${YELLOW}Step 5: Installing systemd service...${NC}"
ssh "${REMOTE_HOST}" "sudo cp ${APP_DIR}/deploy/seed-dev.service /etc/systemd/system/${SERVICE_NAME}.service"
ssh "${REMOTE_HOST}" "sudo systemctl daemon-reload"
ssh "${REMOTE_HOST}" "sudo systemctl enable ${SERVICE_NAME}"
echo -e "${GREEN}✓ Service installed and enabled${NC}"
echo

echo -e "${YELLOW}Step 6: Starting service...${NC}"
ssh "${REMOTE_HOST}" "sudo systemctl restart ${SERVICE_NAME}"
sleep 3
echo -e "${GREEN}✓ Service started${NC}"
echo

echo -e "${YELLOW}Step 7: Checking status...${NC}"
ssh "${REMOTE_HOST}" "sudo systemctl status ${SERVICE_NAME} --no-pager -l" || true
echo

echo "================================================"
echo " Deployment Complete!"
echo "================================================"
echo
echo "Service status:"
ssh "${REMOTE_HOST}" "sudo systemctl is-active ${SERVICE_NAME}" && \
  echo -e "${GREEN}✓ Service is running${NC}" || \
  echo -e "${RED}✗ Service failed to start${NC}"
echo
echo "View logs: ssh ${REMOTE_HOST} sudo journalctl -u ${SERVICE_NAME} -f"
echo "Stop service: ssh ${REMOTE_HOST} sudo systemctl stop ${SERVICE_NAME}"
echo "Restart service: ssh ${REMOTE_HOST} sudo systemctl restart ${SERVICE_NAME}"
echo
