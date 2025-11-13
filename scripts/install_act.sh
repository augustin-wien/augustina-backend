#!/usr/bin/env bash
set -euo pipefail

# Download and install 'act' into ./tools/act (no sudo required)
TOOLS_DIR="$(pwd)/tools"
ACT_BIN="$TOOLS_DIR/act"

mkdir -p "$TOOLS_DIR"

echo "Detecting latest act release..."
URL="https://github.com/nektos/act/releases/latest/download/act_Linux_x86_64.tar.gz"

echo "Downloading act from $URL"
curl -fsSL "$URL" -o /tmp/act.tar.gz
tar -xzf /tmp/act.tar.gz -C /tmp
mv /tmp/act "$ACT_BIN"
chmod +x "$ACT_BIN"

echo "act installed to $ACT_BIN"
echo "You can run it via: $ACT_BIN -j Gotest --env-file .env -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:full-22.04"
