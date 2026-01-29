#!/bin/bash

# Mandok Uninstaller
# This script stops and removes the Mandok service and its files.

set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (use sudo)"
  exit 1
fi

INSTALL_DIR="/opt/mandok"
SERVICE_NAME="mandok.service"

echo "Stopping Mandok service..."
systemctl stop "$SERVICE_NAME" || echo "Service not running."

echo "Disabling Mandok service..."
systemctl disable "$SERVICE_NAME" || echo "Service not enabled."

echo "Removing systemd service file..."
rm -f "/etc/systemd/system/$SERVICE_NAME"

echo "Reloading systemd..."
systemctl daemon-reload

echo "Removing Mandok installation directory (excluding configuration)?"
read -p "Do you want to remove the entire $INSTALL_DIR directory? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$INSTALL_DIR"
    echo "Mandok and its configuration have been removed."
else
    # Just remove the binary
    rm -f "$INSTALL_DIR/mandok"
    echo "Mandok binary removed. Configuration and data in $INSTALL_DIR kept."
fi

echo "Uninstallation complete."
