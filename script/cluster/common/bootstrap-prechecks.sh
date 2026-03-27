#!/usr/bin/env bash
set -euo pipefail

REPO_MODE="${REPO_MODE:-online}"
REPO_URL="${REPO_URL:-}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-}"
ETCD_MODE="${ETCD_MODE:-stacked}"
EXTERNAL_ETCD_JSON="${EXTERNAL_ETCD_JSON:-}"

log() { echo "[precheck] $*"; }
fail() { echo "[precheck][ERROR] $*" >&2; exit 1; }

if [[ "$REPO_MODE" == "mirror" ]]; then
  [[ -n "$REPO_URL" ]] || fail "repo_mode=mirror but REPO_URL is empty"
  if ! curl -fsSIL --max-time 4 "$REPO_URL" >/dev/null; then
    fail "mirror repo unreachable: $REPO_URL"
  fi
  log "mirror repo reachable: $REPO_URL"
fi

if [[ -n "$IMAGE_REPOSITORY" ]]; then
  URL="$IMAGE_REPOSITORY"
  if [[ "$URL" != http* ]]; then
    URL="https://${URL%/}/v2/"
  fi
  if ! curl -fsSIL --max-time 4 "$URL" >/dev/null; then
    fail "image repository unreachable: $IMAGE_REPOSITORY"
  fi
  log "image repository reachable: $IMAGE_REPOSITORY"
fi

if [[ "$ETCD_MODE" == "external" ]]; then
  [[ -n "$EXTERNAL_ETCD_JSON" ]] || fail "etcd_mode=external but EXTERNAL_ETCD_JSON empty"
  ENDPOINTS=$(echo "$EXTERNAL_ETCD_JSON" | grep -o 'https\?://[^", ]*' || true)
  [[ -n "$ENDPOINTS" ]] || fail "external etcd endpoints missing"
  for ep in $ENDPOINTS; do
    hostport=${ep#https://}
    hostport=${hostport#http://}
    if ! timeout 4 bash -c "cat < /dev/null > /dev/tcp/${hostport%:*}/${hostport##*:}" 2>/dev/null; then
      fail "external etcd endpoint unreachable: $ep"
    fi
    log "external etcd endpoint reachable: $ep"
  done
fi

log "prechecks passed"
