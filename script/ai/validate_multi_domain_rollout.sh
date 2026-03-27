#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "[1/4] Running Go test suite"
(cd "$ROOT_DIR" && GOCACHE=/tmp/go-build go test ./...)

echo "[2/4] Building frontend"
(cd "$ROOT_DIR/web" && npm run build)

echo "[3/4] Validating OpenSpec change"
(cd "$ROOT_DIR" && openspec validate multi-domain-agent-architecture --json)

echo "[4/4] Manual checklist"
echo "Run docs/ai-multi-domain-manual-test.md to complete manual verification."
