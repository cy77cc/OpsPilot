#!/usr/bin/env bash
set -euo pipefail

CONTROL_PLANE_ENDPOINT="${CONTROL_PLANE_ENDPOINT:-}"
ENDPOINT_MODE="${ENDPOINT_MODE:-nodeIP}"

if [[ "$ENDPOINT_MODE" == "nodeIP" ]]; then
  exit 0
fi

if [[ -z "$CONTROL_PLANE_ENDPOINT" ]]; then
  echo "[endpoint][ERROR] CONTROL_PLANE_ENDPOINT is empty" >&2
  exit 1
fi

host=${CONTROL_PLANE_ENDPOINT%:*}
port=${CONTROL_PLANE_ENDPOINT##*:}

for i in {1..8}; do
  if timeout 3 bash -c "cat < /dev/null > /dev/tcp/${host}/${port}" 2>/dev/null; then
    echo "[endpoint] endpoint reachable: ${CONTROL_PLANE_ENDPOINT}"
    exit 0
  fi
  sleep 2
done

echo "[endpoint][ERROR] endpoint not reachable: ${CONTROL_PLANE_ENDPOINT}" >&2
exit 1
