#!/usr/bin/env bash
# Calico CNI 卸载脚本
set -euo pipefail

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 删除 Calico 资源
uninstall_calico() {
    log_info "删除 Calico 资源..."

    # 删除 DaemonSet 和 Deployment
    kubectl delete daemonset calico-node -n kube-system --ignore-not-found=true
    kubectl delete deployment calico-kube-controllers -n kube-system --ignore-not-found=true

    # 删除相关资源
    kubectl delete configmap calico-config -n kube-system --ignore-not-found=true
    kubectl delete secret calico-etcd-secrets -n kube-system --ignore-not-found=true
    kubectl delete clusterrole calico-kube-controllers --ignore-not-found=true
    kubectl delete clusterrolebinding calico-kube-controllers --ignore-not-found=true
    kubectl delete clusterrole calico-node --ignore-not-found=true
    kubectl delete clusterrolebinding calico-node --ignore-not-found=true
    kubectl delete serviceaccount calico-node -n kube-system --ignore-not-found=true
    kubectl delete serviceaccount calico-kube-controllers -n kube-system --ignore-not-found=true

    # 删除 CRD
    kubectl delete crd bgpconfigurations.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd bgppeers.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd blockaffinities.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd caliconodestatuses.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd clusterinformations.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd felixconfigurations.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd globalnetworkpolicies.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd globalnetworksets.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd hostendpoints.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd ipamblocks.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd ipamconfigs.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd ipamhandles.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd ippools.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd ipreservations.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd kubecontrollersconfigurations.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd networkpolicies.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd networksets.crd.projectcalico.org --ignore-not-found=true
    kubectl delete crd profiles.crd.projectcalico.org --ignore-not-found=true

    log_info "Calico 资源已删除"
}

# 清理节点网络配置
cleanup_node_network() {
    log_info "清理节点网络配置..."

    # 删除 tunl0 接口
    ip link delete tunl0 2>/dev/null || true

    # 删除 cali 接口
    for iface in $(ip link show | grep -o 'cali[0-9a-f]*' | sort -u); do
        ip link delete "$iface" 2>/dev/null || true
    done

    # 清理 iptables 规则
    iptables -t nat -F cali-nat-outgoing 2>/dev/null || true
    iptables -t nat -F cali-fip-snat 2>/dev/null || true
    iptables -t nat -F cali-fip-dnat 2>/dev/null || true
    iptables -F cali-fw 2>/dev/null || true
    iptables -F cali-pri 2>/dev/null || true
    iptables -F cali-pro 2>/dev/null || true
    iptables -F cali-anti 2>/dev/null || true
    iptables -F cali-wl-pro 2>/dev/null || true
    iptables -F cali-pro-ks 2>/dev/null || true

    log_info "节点网络配置已清理"
}

# 主函数
main() {
    log_info "开始卸载 Calico CNI..."

    uninstall_calico
    cleanup_node_network

    log_info "Calico 卸载完成"
}

main "$@"
