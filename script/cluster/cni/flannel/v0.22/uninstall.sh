#!/usr/bin/env bash
# Flannel CNI 卸载脚本
set -euo pipefail

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# 删除 Flannel 资源
uninstall_flannel() {
    log_info "删除 Flannel 资源..."

    # 尝试删除不同 namespace 中的资源
    kubectl delete -n kube-flannel daemonset kube-flannel-ds --ignore-not-found=true
    kubectl delete -n kube-flannel configmap kube-flannel-cfg --ignore-not-found=true
    kubectl delete -n kube-flannel serviceaccount flannel --ignore-not-found=true

    kubectl delete -n kube-system daemonset kube-flannel-ds --ignore-not-found=true
    kubectl delete -n kube-system configmap kube-flannel-cfg --ignore-not-found=true
    kubectl delete -n kube-system serviceaccount flannel --ignore-not-found=true

    # 删除 ClusterRole 和 ClusterRoleBinding
    kubectl delete clusterrole flannel --ignore-not-found=true
    kubectl delete clusterrolebinding flannel --ignore-not-found=true

    # 删除 namespace
    kubectl delete namespace kube-flannel --ignore-not-found=true

    log_info "Flannel 资源已删除"
}

# 清理节点网络配置
cleanup_node_network() {
    log_info "清理节点网络配置..."

    # 删除 flannel.1 接口
    ip link delete flannel.1 2>/dev/null || true

    # 删除 cni0 接口
    ip link delete cni0 2>/dev/null || true

    # 清理 iptables 规则
    iptables -t nat -F POSTROUTING -s 10.244.0.0/16 ! -d 10.244.0.0/16 -j MASQUERADE 2>/dev/null || true
    iptables -F FLANNEL-POSTRTG 2>/dev/null || true
    iptables -F FLANNEL-FORWARD 2>/dev/null || true

    log_info "节点网络配置已清理"
}

# 主函数
main() {
    log_info "开始卸载 Flannel CNI..."

    uninstall_flannel
    cleanup_node_network

    log_info "Flannel 卸载完成"
}

main "$@"
