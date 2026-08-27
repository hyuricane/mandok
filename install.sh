#!/bin/bash

# Mandok Installer
# This script builds and installs Mandok as a systemd service.

set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (use sudo)"
  exit 1
fi

INSTALL_DIR="/opt/mandok"
BIN_NAME="mandok"
SERVICE_NAME="mandok.service"

if [ ! -f "dist/mandok" ]; then
    echo "Error: dist/mandok not found. Please run ./build.sh first."
    exit 1
fi

echo "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR/projects"

# if busy offer to kill or stop current service
if [ -f "$INSTALL_DIR/mandok" ]; then
    echo "Mandok is already installed. Do you want to update it?"
    read -p "[Y/n] " -n 1 -r
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        exit 0
    fi
    echo "stopping current service..."
    sudo systemctl stop $SERVICE_NAME
fi

echo "Installing binary..."
cp dist/mandok "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/mandok"

echo "Copying env.example..."
cp env.example "$INSTALL_DIR/"

if [ ! -f "$INSTALL_DIR/.env" ]; then
    echo "Initializing .env..."
    if [ -f ".env" ]; then
        echo "Copying existing .env..."
        cp .env "$INSTALL_DIR/.env"
    elif [ -f "env.example" ]; then
        echo "Creating .env from env.example..."
        cp env.example "$INSTALL_DIR/.env"
    elif [ -f "sample.env" ]; then
        echo "Creating .env from sample.env..."
        cp sample.env "$INSTALL_DIR/.env"
    fi
    echo "IMPORTANT: Please edit $INSTALL_DIR/.env with your configuration."
fi

echo "Configuring systemd service..."
# Prepare the service file with correct paths
sed -e "s|WorkingDirectory=.*|WorkingDirectory=$INSTALL_DIR|" \
    -e "s|ExecStart=.*|ExecStart=$INSTALL_DIR/$BIN_NAME|" \
    mandok.service > "/etc/systemd/system/$SERVICE_NAME"

echo "Reloading systemd..."
systemctl daemon-reload

echo "Enabling $SERVICE_NAME..."
systemctl enable "$SERVICE_NAME"

echo "Starting $SERVICE_NAME..."
systemctl start "$SERVICE_NAME"

echo "Mandok has been installed and started."
echo "You can check the status with: systemctl status $SERVICE_NAME"
echo "Installation directory: $INSTALL_DIR"
