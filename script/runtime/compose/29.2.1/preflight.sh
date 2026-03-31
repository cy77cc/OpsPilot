#!/usr/bin/env bash
set -euo pipefail
command -v docker >/dev/null 2>&1
docker compose version >/dev/null 2>&1
echo "compose preflight ok"
