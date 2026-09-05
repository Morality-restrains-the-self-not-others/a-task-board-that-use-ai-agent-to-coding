#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
PROJECT_NAME="docker-mysql"

# Check if MySQL container is running and healthy
if docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps --status running 2>/dev/null | grep -q 'healthy'; then
  exit 0
fi

# Fallback: try to ping MySQL directly
if mysqladmin ping -h 127.0.0.1 -u root -proot123456 --silent 2>/dev/null; then
  exit 0
fi

exit 1
