#!/bin/bash

# Check if correct number of arguments are provided
if [ $# -ne 2 ]; then
    echo "Usage: $0 <username> <group>"
    exit 1
fi

# Get the username and group from the arguments
USERNAME=$1
GROUP=$2

# Get the current directory
MINIMON_DIR=$(pwd)
MINIMON_BINARY="$MINIMON_DIR/minimon"
SERVICE_FILE="/etc/systemd/system/minimon-${USERNAME}.service"
CONFIG_PATH="$MINIMON_DIR/config.json"

# Build the Go program
echo "Building MiniMon..."
cd "$MINIMON_DIR" || exit
go mod init minimon
go get github.com/fsnotify/fsnotify
go get github.com/gen2brain/beeep
go get github.com/rs/zerolog/log
go build -o "$MINIMON_BINARY" minimon.go

# Ensure the build was successful
if [ $? -ne 0 ]; then
    echo "Build failed! Please check the Go code."
    exit 1
fi

# Create the systemd service file
echo "Creating systemd service file for $USERNAME..."
sudo bash -c "cat > $SERVICE_FILE" <<EOL
[Unit]
Description=MiniMon Log Monitoring Service for $USERNAME
After=network.target

[Service]
ExecStart=$MINIMON_BINARY
WorkingDirectory=$MINIMON_DIR
Environment="MINIMON_CONFIG=$CONFIG_PATH"
Restart=always
User=$USERNAME
Group=$GROUP
KillSignal=SIGINT
TimeoutStopSec=20

[Install]
WantedBy=multi-user.target
EOL

# Reload systemd to apply the changes
echo "Reloading systemd..."
sudo systemctl daemon-reload

# Enable the service to start at boot
echo "Enabling MiniMon service for $USERNAME..."
sudo systemctl enable minimon-${USERNAME}.service

# Start the service immediately
echo "Starting MiniMon service for $USERNAME..."
sudo systemctl start minimon-${USERNAME}.service

# Display the service status
sudo systemctl status minimon-${USERNAME}.service
