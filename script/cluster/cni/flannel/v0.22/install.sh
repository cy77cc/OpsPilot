#!/usr/bin/env bash
# Flannel CNI 安装脚本
set -euo pipefail

# 配置
FLANNEL_VERSION="${FLANNEL_VERSION:-0.22.3}"
POD_CIDR="${POD_CIDR:-10.244.0.0/16}"
FLANNEL_MANIFEST_URL="${FLANNEL_MANIFEST_URL:-https://raw.githubusercontent.com/flannel-io/flannel/v${FLANNEL_VERSION}/Documentation/kube-flannel.yml}"
BACKEND="${BACKEND:-vxlan}"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 kubectl
if ! command -v kubectl &>/dev/null; then
    log_error "kubectl 未安装"
    exit 1
fi

# 检查集群连接
check_cluster() {
    log_info "检查集群连接..."

    if ! kubectl cluster-info &>/dev/null; then
        log_error "无法连接到 Kubernetes 集群"
        exit 1
    fi

    log_info "集群连接正常"
}

# 安装 Flannel
install_flannel() {
    log_info "安装 Flannel v${FLANNEL_VERSION}..."

    local manifest_file="/tmp/kube-flannel-${FLANNEL_VERSION}.yaml"

    # 下载 manifest
    log_info "下载 Flannel manifest..."
    curl -sSL "$FLANNEL_MANIFEST_URL" -o "$manifest_file"

    if [[ ! -f "$manifest_file" ]]; then
        log_error "下载 manifest 失败"
        exit 1
    fi

    # 修改 Pod CIDR（如果需要）
    if [[ "$POD_CIDR" != "10.244.0.0/16" ]]; then
        log_info "配置 Pod CIDR: $POD_CIDR"

        # 修改 net-conf.json
        local cidr_prefix=$(echo "$POD_CIDR" | cut -d'.' -f1-2)
        sed -i "s#\"Network\": \"10.244.0.0/16\"#\"Network\": \"${POD_CIDR}\"#g" "$manifest_file"
    fi

    # 应用 manifest
    log_info "应用 Flannel manifest..."
    kubectl apply -f "$manifest_file"

    # 等待 Flannel 就绪
    wait_for_flannel

    # 清理临时文件
    rm -f "$manifest_file"
}

# 等待 Flannel 就绪
wait_for_flannel() {
    log_info "等待 Flannel 组件就绪..."

    # 等待 DaemonSet
    kubectl rollout status daemonset/kube-flannel-ds -n kube-flannel --timeout=300s || \
    kubectl rollout status daemonset/kube-flannel-ds -n kube-system --timeout=300s || true

    # 检查 Pod 状态
    log_info "检查 Flannel Pod 状态..."
    kubectl get pods -n kube-flannel -l app=flannel 2>/dev/null || \
    kubectl get pods -n kube-system -l app=flannel 2>/dev/null || true

    echo ""
    log_info "Flannel 安装完成!"
}

# 验证安装
verify_installation() {
    log_info "验证 Flannel 安装..."

    # 检查 flannel Pod
    local flannel_pods=$(kubectl get pods -n kube-flannel -l app=flannel --no-headers 2>/dev/null | wc -l || \
                         kubectl get pods -n kube-system -l app=flannel --no-headers 2>/dev/null | wc -l || echo "0")

    if [[ $flannel_pods -gt 0 ]]; then
        log_info "检测到 $flannel_pods 个 Flannel Pod"
    else
        log_warn "未检测到 Flannel Pod"
    fi

    # 检查 flannel.1 接口
    if ip link show flannel.1 &>/dev/null; then
        log_info "flannel.1 接口存在"
    else
        log_warn "flannel.1 接口不存在"
    fi
}

# 主函数
main() {
    local action="${1:-install}"

    case "$action" in
        install)
            log_info "开始安装 Flannel CNI..."
            check_cluster
            install_flannel
            verify_installation
            ;;

        verify)
            verify_installation
            ;;

        *)
            echo "用法: $0 [install|verify]"
            echo ""
            echo "环境变量:"
            echo "  FLANNEL_VERSION       Flannel 版本 (默认: 0.22.3)"
            echo "  POD_CIDR              Pod 网络 CIDR (默认: 10.244.0.0/16)"
            echo "  BACKEND               后端类型 (默认: vxlan)"
            exit 1
            ;;
    esac
}

main "$@"
