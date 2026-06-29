#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KOMMENTS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

MCP_DIR="$KOMMENTS_ROOT/mcp-server"
DIST_ENTRY="$MCP_DIR/dist/index.js"
GO_BIN="$KOMMENTS_ROOT/komments"

# Build Go binary if needed
if [ ! -f "$GO_BIN" ]; then
  echo "Building komments binary..."
  (cd "$KOMMENTS_ROOT" && go build -o komments ./cmd/komments/)
fi

if [ ! -f "$GO_BIN" ]; then
  echo "Error: Go build failed — $GO_BIN not found" >&2
  exit 1
fi

# Build MCP server if needed
if [ ! -f "$DIST_ENTRY" ]; then
  echo "Building MCP server..."
  (cd "$MCP_DIR" && npm install && npm run build)
fi

if [ ! -f "$DIST_ENTRY" ]; then
  echo "Error: MCP build failed — $DIST_ENTRY not found" >&2
  exit 1
fi

SETTINGS_DIR="$HOME/.claude"
SETTINGS_FILE="$SETTINGS_DIR/settings.json"

mkdir -p "$SETTINGS_DIR"

# Ensure komments binary is on PATH or use absolute path
KOMMENTS_BIN="$GO_BIN"

node -e "
const fs = require('fs');

const settingsPath = process.argv[1];
const entryPoint = process.argv[2];
const kommentsBin = process.argv[3];

let settings = {};
try {
  settings = JSON.parse(fs.readFileSync(settingsPath, 'utf-8'));
} catch {}

if (!settings.mcpServers) {
  settings.mcpServers = {};
}

settings.mcpServers.komments = {
  command: 'node',
  args: [entryPoint],
  env: { KOMMENTS_BIN: kommentsBin },
};

fs.writeFileSync(settingsPath, JSON.stringify(settings, null, 2) + '\n');
console.log('Updated ' + settingsPath);
" "$SETTINGS_FILE" "$DIST_ENTRY" "$KOMMENTS_BIN"

echo "Komments installed for Claude Code."
echo "  Binary:     $KOMMENTS_BIN"
echo "  MCP server: $DIST_ENTRY"
