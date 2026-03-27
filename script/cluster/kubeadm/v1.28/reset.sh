#!/usr/bin/env bash
# Kubernetes 节点重置脚本 (kubeadm reset)
set -euo pipefail

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 确认操作
confirm_reset() {
    if [[ "${FORCE:-false}" != "true" ]]; then
        echo "警告: 此操作将:"
        echo "  - 重置 kubeadm 配置"
        echo "  - 删除本节点的 Kubernetes 配置"
        echo "  - 清理 iptables/IPVS 规则"
        echo "  - 删除 CNI 配置"
        echo ""
        read -p "确认继续? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "操作已取消"
            exit 0
        fi
    fi
}

# 执行 kubeadm reset
run_reset() {
    log_info "执行 kubeadm reset..."

    if command -v kubeadm &>/dev/null; then
        kubeadm reset -f --cri-socket unix:///run/containerd/containerd.sock 2>/dev/null || true
    fi
}

# 清理配置文件
cleanup_configs() {
    log_info "清理配置文件..."

    # 删除 kubeconfig
    rm -rf /root/.kube
    rm -rf /home/*/.kube 2>/dev/null || true

    # 删除 CNI 配置
    rm -rf /etc/cni/net.d/*

    # 删除 kubeadm 配置
    rm -rf /etc/kubernetes
    rm -f /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

    # 删除 pki
    rm -rf /var/lib/kubelet/pki 2>/dev/null || true
    rm -rf /var/lib/etcd 2>/dev/null || true

    log_info "配置文件已清理"
}

# 清理 iptables
cleanup_iptables() {
    log_info "清理 iptables 规则..."

    # 清理 iptables
    if command -v iptables &>/dev/null; then
        iptables -F
        iptables -t nat -F
        iptables -t mangle -F
        iptables -X
    fi

    # 清理 ip6tables
    if command -v ip6tables &>/dev/null; then
        ip6tables -F
        ip6tables -t nat -F
        ip6tables -t mangle -F
        ip6tables -X
    fi

    # 清理 IPVS
    if command -v ipvsadm &>/dev/null; then
        ipvsadm -C 2>/dev/null || true
    fi

    log_info "iptables 规则已清理"
}

# 重启 kubelet
restart_kubelet() {
    log_info "重启 kubelet 服务..."

    systemctl daemon-reload
    systemctl stop kubelet 2>/dev/null || true
    systemctl disable kubelet 2>/dev/null || true

    log_info "kubelet 服务已停止"
}

# 清理网络接口
cleanup_network() {
    log_info "清理网络接口..."

    # 删除 cni0 接口
    ip link delete cni0 2>/dev/null || true

    # 删除 tunl0 接口 (Calico)
    ip link delete tunl0 2>/dev/null || true

    # 删除 flannel.1 接口
    ip link delete flannel.1 2>/dev/null || true

    # 删除 cilium_host 接口
    ip link delete cilium_host 2>/dev/null || true
    ip link delete cilium_vxlan 2>/dev/null || true

    log_info "网络接口已清理"
}

# 主函数
main() {
    log_info "开始重置 Kubernetes 节点..."

    confirm_reset
    run_reset
    cleanup_configs
    cleanup_iptables
    cleanup_network
    restart_kubelet

    echo ""
    log_info "节点重置完成!"
    log_info "如需重新加入集群，请使用相应的 kubeadm join 或 kubeadm init 命令"
}

main "$@"
