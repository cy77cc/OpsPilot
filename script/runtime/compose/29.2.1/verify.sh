#!/usr/bin/env bash
set -euo pipefail
docker compose version >/dev/null 2>&1
echo "compose verify ok"
