#!/usr/bin/env bash
set -euo pipefail

ENDPOINT_MODE="${ENDPOINT_MODE:-nodeIP}"
VIP_PROVIDER="${VIP_PROVIDER:-kube-vip}"
CONTROL_PLANE_ENDPOINT="${CONTROL_PLANE_ENDPOINT:-}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-}"

log() { echo "[vip] $*"; }
fail() { echo "[vip][ERROR] $*" >&2; exit 1; }

if [[ "$ENDPOINT_MODE" != "vip" && "$ENDPOINT_MODE" != "lbDNS" ]]; then
  log "endpoint mode=$ENDPOINT_MODE, skip vip provider"
  exit 0
fi

[[ -n "$CONTROL_PLANE_ENDPOINT" ]] || fail "CONTROL_PLANE_ENDPOINT required when vip/lbDNS"

case "$VIP_PROVIDER" in
  kube-vip)
    if ! command -v kubectl >/dev/null 2>&1; then
      fail "kubectl not found for kube-vip automation"
    fi
    ns="kube-system"
    img="ghcr.io/kube-vip/kube-vip:v0.8.0"
    if [[ -n "$IMAGE_REPOSITORY" ]]; then
      img="${IMAGE_REPOSITORY%/}/kube-vip:v0.8.0"
    fi
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-vip-ds
  namespace: ${ns}
spec:
  selector:
    matchLabels:
      app: kube-vip
  template:
    metadata:
      labels:
        app: kube-vip
    spec:
      hostNetwork: true
      containers:
      - name: kube-vip
        image: ${img}
        args: ["manager"]
        securityContext:
          capabilities:
            add: ["NET_ADMIN","NET_RAW"]
EOF
    log "kube-vip daemonset applied"
    ;;
  keepalived)
    if command -v apt-get >/dev/null 2>&1; then
      apt-get update && apt-get install -y keepalived
    elif command -v yum >/dev/null 2>&1; then
      yum install -y keepalived
    fi
    systemctl enable keepalived || true
    systemctl start keepalived || true
    log "keepalived installed and started"
    ;;
  *)
    fail "unsupported VIP_PROVIDER: $VIP_PROVIDER"
    ;;
esac
