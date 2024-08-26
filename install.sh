#!/bin/bash

# Get the current directory
MINIMON_DIR=$(pwd)
MINIMON_BINARY="$MINIMON_DIR/minimon"
SERVICE_FILE="/etc/systemd/system/minimon.service"
CONFIG_PATH="$MINIMON_DIR/config.json"

# Replace placeholders
USERNAME=$(whoami)
GROUP=$(id -gn)

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
echo "Creating systemd service file..."
sudo bash -c "cat > $SERVICE_FILE" <<EOL
[Unit]
Description=MiniMon Log Monitoring Service
After=network.target

[Service]
ExecStart=$MINIMON_BINARY
WorkingDirectory=$MINIMON_DIR
Environment="MINIMON_CONFIG=$CONFIG_PATH"
Restart=always
User=$USERNAME
Group=$GROUP

[Install]
WantedBy=multi-user.target
EOL

# Reload systemd to apply the changes
echo "Reloading systemd..."
sudo systemctl daemon-reload

# Enable the service to start at boot
echo "Enabling MiniMon service..."
sudo systemctl enable minimon.service

# Start the service immediately
echo "Starting MiniMon service..."
sudo systemctl start minimon.service

# Display the service status
sudo systemctl status minimon.service
