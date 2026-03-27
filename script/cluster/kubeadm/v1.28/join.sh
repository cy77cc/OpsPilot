#!/usr/bin/env bash
# Kubernetes Worker 节点加入集群脚本 (kubeadm join)
set -euo pipefail

# 配置参数
JOIN_COMMAND="${JOIN_COMMAND:-}"
CONTROL_PLANE_IP="${CONTROL_PLANE_IP:-}"
TOKEN="${TOKEN:-}"
CA_CERT_HASH="${CA_CERT_HASH:-}"
CRI_SOCKET="${CRI_SOCKET:-/run/containerd/containerd.sock}"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 root 权限
if [[ $EUID -ne 0 ]]; then
    log_error "此脚本需要 root 权限"
    exit 1
fi

# 检查 kubeadm
if ! command -v kubeadm &>/dev/null; then
    log_error "kubeadm 未安装"
    exit 1
fi

# 检查容器运行时
check_container_runtime() {
    log_info "检查容器运行时..."

    if [[ -S "$CRI_SOCKET" ]]; then
        log_info "检测到 containerd socket: $CRI_SOCKET"
    elif [[ -S "/var/run/docker.sock" ]]; then
        log_warn "检测到 Docker，建议使用 containerd"
        CRI_SOCKET="/var/run/dockershim.sock"
    else
        log_error "未检测到容器运行时"
        exit 1
    fi
}

# 执行 join
run_join() {
    log_info "执行 kubeadm join..."

    local join_args=""

    if [[ -n "$JOIN_COMMAND" ]]; then
        # 使用完整的 join 命令
        # 提取参数部分
        join_args=$(echo "$JOIN_COMMAND" | sed 's/^kubeadm join //')
    elif [[ -n "$CONTROL_PLANE_IP" && -n "$TOKEN" && -n "$CA_CERT_HASH" ]]; then
        # 使用分离的参数构建
        join_args="${CONTROL_PLANE_IP}:6443"
        join_args+=" --token ${TOKEN}"
        join_args+=" --discovery-token-ca-cert-hash sha256:${CA_CERT_HASH}"
    else
        log_error "缺少必要的 join 参数"
        echo ""
        echo "请设置以下环境变量之一:"
        echo "  方式 1: JOIN_COMMAND (完整的 kubeadm join 命令)"
        echo "  方式 2: CONTROL_PLANE_IP, TOKEN, CA_CERT_HASH"
        exit 1
    fi

    # 添加 CRI socket
    join_args+=" --cri-socket unix://${CRI_SOCKET}"

    # 忽略 swap 预检查
    join_args+=" --ignore-preflight-errors=Swap"

    log_info "命令: kubeadm join ${join_args}"

    # 执行 join
    kubeadm join ${join_args} 2>&1 | tee /var/log/kubeadm-join.log

    if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
        log_error "kubeadm join 失败"
        cat /var/log/kubeadm-join.log
        exit 1
    fi
}

# 验证节点状态
verify_node() {
    log_info "验证节点状态..."

    # 等待节点就绪
    log_info "等待节点就绪..."

    # 检查 kubelet 服务
    if systemctl is-active kubelet; then
        log_info "kubelet 服务运行中"
    else
        log_error "kubelet 服务未运行"
        journalctl -u kubelet --no-pager -n 20
        exit 1
    fi

    log_info "节点已加入集群"
    log_info "请在控制平面节点执行 'kubectl get nodes' 查看节点状态"
}

# 主函数
main() {
    local action="${1:-join}"

    case "$action" in
        join)
            log_info "开始将节点加入 Kubernetes 集群..."
            check_container_runtime
            run_join
            verify_node
            log_info "节点加入完成!"
            ;;

        *)
            echo "用法: $0 [join]"
            echo ""
            echo "环境变量:"
            echo "  JOIN_COMMAND      完整的 kubeadm join 命令"
            echo "  CONTROL_PLANE_IP  控制平面 IP 地址"
            echo "  TOKEN             join token"
            echo "  CA_CERT_HASH      CA 证书 hash"
            echo "  CRI_SOCKET        容器运行时 socket (默认: /run/containerd/containerd.sock)"
            exit 1
            ;;
    esac
}

main "$@"
