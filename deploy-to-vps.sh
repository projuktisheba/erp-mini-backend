#!/bin/bash

# -----------------------------
# Configuration
# -----------------------------
VPS_HOST="192.250.228.113"
REMOTE_PATH="/home/samiul/apps/bin/main-erp-mini-backend"
SERVICE_NAME="erpminiapi.service"
PING_URL="https://api.erp.pssoft.xyz/ping"

# -----------------------------
# Step 1: Remove old binary locally
# -----------------------------
echo "Removing old binary..."
rm -f app

# -----------------------------
# Step 2: Build the Go app
# -----------------------------
echo "Building app..."
go build -ldflags="-s -w" -o app
if [[ $? -ne 0 ]]; then
    echo "Build failed. Exiting."
    exit 1
fi

# -----------------------------
# Step 3: Stop the service on VPS
# -----------------------------
echo "Stopping remote service..."
ssh root@"$VPS_HOST" "systemctl stop $SERVICE_NAME"
if [[ $? -ne 0 ]]; then
    echo "Failed to stop service. Exiting."
    exit 1
fi

# -----------------------------
# Step 4: Copy the new binary to VPS
# -----------------------------
echo "Uploading new binary..."
scp app root@"$VPS_HOST":"$REMOTE_PATH"
if [[ $? -ne 0 ]]; then
    echo "SCP failed. Exiting."
    exit 1
fi

# -----------------------------
# Step 5: Restart the service
# -----------------------------
echo "Restarting remote service..."
ssh root@"$VPS_HOST" "systemctl restart $SERVICE_NAME && systemctl status $SERVICE_NAME --no-pager"
if [[ $? -ne 0 ]]; then
    echo "Failed to restart service."
    exit 1
fi

# -----------------------------
# Step 6: Ping the endpoint
# -----------------------------
echo "Pinging API..."
curl -s -o /dev/null -w "%{http_code}\n" "$PING_URL"
