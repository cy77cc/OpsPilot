#!/usr/bin/env bash
# Cilium CNI 卸载脚本
set -euo pipefail

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# 使用 Helm 卸载
uninstall_with_helm() {
    log_info "使用 Helm 卸载 Cilium..."

    # 检查 Helm
    if command -v helm &>/dev/null; then
        helm uninstall cilium -n cilium --ignore-not-found=true
    else
        log_warn "Helm 未安装，尝试手动删除资源"
    fi

    # 删除命名空间
    kubectl delete namespace cilium --ignore-not-found=true

    log_info "Cilium 资源已删除"
}

# 清理节点网络配置
cleanup_node_network() {
    log_info "清理节点网络配置..."

    # 删除 cilium 接口
    ip link delete cilium_host 2>/dev/null || true
    ip link delete cilium_net1 2>/dev/null || true
    ip link delete cilium_vxlan 2>/dev/null || true

    # 清理 BPF 程序（如果工具可用）
    if command -v bpftool &>/dev/null; then
        bpftool prog show name cilium 2>/dev/null | grep -o 'id [0-9]*' | awk '{print $2}' | xargs -r -I {} bpftool prog detach id {} 2>/dev/null || true
    fi

    # 清理 iptables 规则
    iptables -t nat -F CILIUM_POST_nat 2>/dev/null || true
    iptables -t nat -F CILIUM_PRE_nat 2>/dev/null || true
    iptables -F CILIUM_FORWARD 2>/dev/null || true
    iptables -F CILIUM_INPUT 2>/dev/null || true
    iptables -F CILIUM_OUTPUT 2>/dev/null || true

    log_info "节点网络配置已清理"
}

# 主函数
main() {
    log_info "开始卸载 Cilium CNI..."

    uninstall_with_helm
    cleanup_node_network

    log_info "Cilium 卸载完成"
}

main "$@"
