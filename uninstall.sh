#!/bin/bash

# Check if correct number of arguments are provided
if [ $# -ne 2 ]; then
    echo "Usage: $0 <username> <group>"
    exit 1
fi
USERNAME=$1
GROUP=$2

SERVICE_FILE="/etc/systemd/system/minimon-${USERNAME}.service"

echo "Stopping MiniMon service..."
sudo systemctl stop minimon-${USERNAME}.service

echo "Disabling MiniMon service..."
sudo systemctl disable minimon-${USERNAME}.service

echo "Removing MiniMon service file..."
sudo rm -f $SERVICE_FILE

echo "Reloading systemd..."
sudo systemctl daemon-reload

rm -f minimon

echo "MiniMon service uninstalled successfully."