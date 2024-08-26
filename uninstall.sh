#!/bin/bash

SERVICE_FILE="/etc/systemd/system/minimon.service"

echo "Stopping MiniMon service..."
sudo systemctl stop minimon.service

echo "Disabling MiniMon service..."
sudo systemctl disable minimon.service

echo "Removing MiniMon service file..."
sudo rm -f $SERVICE_FILE

echo "Reloading systemd..."
sudo systemctl daemon-reload

rm -f minimon

echo "MiniMon service uninstalled successfully."