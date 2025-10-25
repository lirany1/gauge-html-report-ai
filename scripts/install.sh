#!/bin/bash

# Installation script for Enhanced Gauge HTML Report Plugin
# This script builds and installs the plugin locally

set -e

PLUGIN_NAME="html-report"
VERSION="5.0.0"
BUILD_DIR="build"
PLUGIN_DIR="$BUILD_DIR/plugin"

echo "===================================="
echo "Enhanced Gauge HTML Report Installer"
echo "===================================="
echo ""

# Detect OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    mingw*|msys*|cygwin*)
        OS="windows"
        ;;
esac

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
esac

BINARY_NAME="${PLUGIN_NAME}"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="${PLUGIN_NAME}.exe"
fi

echo "Detected platform: $OS-$ARCH"
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

if ! command -v gauge &> /dev/null; then
    echo "❌ Gauge is not installed. Please install Gauge first."
    echo "   Visit: https://docs.gauge.org/getting_started/installing-gauge.html"
    exit 1
fi

echo "✅ Prerequisites met"
echo ""

# Clean previous builds
echo "Cleaning previous builds..."
rm -rf "$BUILD_DIR"
mkdir -p "$PLUGIN_DIR/bin"

# Download dependencies
echo "Downloading dependencies..."
go mod download
go mod tidy

# Build the binary
echo "Building $PLUGIN_NAME..."
go build -ldflags "-s -w -X main.version=$VERSION" \
    -o "$PLUGIN_DIR/bin/$BINARY_NAME" \
    cmd/html-report-enhanced/main.go

if [ ! -f "$PLUGIN_DIR/bin/$BINARY_NAME" ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

# Copy required files
echo "Copying plugin files..."

# Copy plugin.json (with all our custom settings)
cp plugin.json "$PLUGIN_DIR/"

# Copy themes
echo "Copying themes..."
cp -r web/themes "$PLUGIN_DIR/"

# Copy configuration example
mkdir -p "$PLUGIN_DIR/examples"
cp examples/.gauge-enhanced.yml "$PLUGIN_DIR/examples/"

# Create README for the package
cat > "$PLUGIN_DIR/README.md" <<EOF
# Enhanced Gauge HTML Report Plugin

Version: $VERSION

## Installation

This plugin has been pre-built for your platform ($OS-$ARCH).

To install:
\`\`\`bash
gauge install html-report-enhanced --file html-report-enhanced-$VERSION.zip
\`\`\`

## Usage

After installation, run your Gauge tests normally:
\`\`\`bash
gauge run specs/
\`\`\`

The enhanced HTML reports will be generated in your reports directory.

## Configuration

Copy the example configuration to your project:
\`\`\`bash
cp examples/.gauge-enhanced.yml .
\`\`\`

Then edit it according to your needs.

## Documentation

Full documentation: https://github.com/your-org/gauge-html-report-enhanced
EOF

# Create installation package
echo "Creating installation package..."
PACKAGE_NAME="${PLUGIN_NAME}-${VERSION}"
cd "$BUILD_DIR"
zip -r "${PACKAGE_NAME}.zip" plugin/
cd ..

echo ""
echo "✅ Package created: $BUILD_DIR/${PACKAGE_NAME}.zip"
echo ""

# Install the plugin
echo "Installing plugin to Gauge..."
GAUGE_HOME="${GAUGE_HOME:-$HOME/.gauge}"
PLUGIN_INSTALL_DIR="$GAUGE_HOME/plugins/$PLUGIN_NAME/$VERSION"

# Remove old version if exists
if [ -d "$PLUGIN_INSTALL_DIR" ]; then
    echo "Removing old version..."
    rm -rf "$PLUGIN_INSTALL_DIR"
fi

# Create plugin directory
mkdir -p "$PLUGIN_INSTALL_DIR"

# Copy plugin files
cp -r "$PLUGIN_DIR"/* "$PLUGIN_INSTALL_DIR/"

# Make binary executable
chmod +x "$PLUGIN_INSTALL_DIR/bin/$BINARY_NAME"

echo ""
echo "===================================="
echo "✅ Installation Complete!"
echo "===================================="
echo ""
echo "Plugin installed at: $PLUGIN_INSTALL_DIR"
echo ""
echo "To verify installation:"
echo "  gauge version"
echo ""
echo "To use the plugin:"
echo "  gauge run specs/"
echo ""
echo "The enhanced HTML reports will be generated automatically."
echo ""
