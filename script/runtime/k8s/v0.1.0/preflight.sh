#!/usr/bin/env bash
set -euo pipefail
command -v kubeadm >/dev/null 2>&1
command -v kubectl >/dev/null 2>&1
echo "k8s preflight ok"
