#!/usr/bin/env bash
# Calico CNI 安装脚本
set -euo pipefail

# 配置
CALICO_VERSION="${CALICO_VERSION:-3.26.4}"
POD_CIDR="${POD_CIDR:-192.168.0.0/16}"
CALICO_MANIFEST_URL="${CALICO_MANIFEST_URL:-https://raw.githubusercontent.com/projectcalico/calico/v${CALICO_VERSION}/manifests/calico.yaml}"
INSTALLATION_MODE="${INSTALLATION_MODE:-manifest}"  # manifest 或 helm

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
        log_error "请确保 kubeconfig 已正确配置"
        exit 1
    fi

    log_info "集群连接正常"
}

# 使用 manifest 安装
install_with_manifest() {
    log_info "使用 Manifest 安装 Calico v${CALICO_VERSION}..."

    local manifest_file="/tmp/calico-${CALICO_VERSION}.yaml"

    # 下载 manifest
    log_info "下载 Calico manifest..."
    curl -sSL "$CALICO_MANIFEST_URL" -o "$manifest_file"

    if [[ ! -f "$manifest_file" ]]; then
        log_error "下载 manifest 失败"
        exit 1
    fi

    # 修改 Pod CIDR（如果需要）
    if [[ "$POD_CIDR" != "192.168.0.0/16" ]]; then
        log_info "配置 Pod CIDR: $POD_CIDR"
        sed -i "s#192.168.0.0/16#${POD_CIDR}#g" "$manifest_file"
    fi

    # 应用 manifest
    log_info "应用 Calico manifest..."
    kubectl apply -f "$manifest_file"

    # 等待 Calico 组件就绪
    wait_for_calico

    # 清理临时文件
    rm -f "$manifest_file"
}

# 等待 Calico 就绪
wait_for_calico() {
    log_info "等待 Calico 组件就绪..."

    # 等待 DaemonSet
    log_info "等待 calico-node DaemonSet..."
    kubectl rollout status daemonset/calico-node -n kube-system --timeout=300s

    # 等待 Deployment
    log_info "等待 calico-kube-controllers Deployment..."
    kubectl rollout status deployment/calico-kube-controllers -n kube-system --timeout=120s || true

    # 检查 Pod 状态
    log_info "检查 Calico Pod 状态..."
    kubectl get pods -n kube-system -l k8s-app=calico-node

    echo ""
    log_info "Calico 安装完成!"
}

# 验证安装
verify_installation() {
    log_info "验证 Calico 安装..."

    # 检查 calico-node
    local calico_pods=$(kubectl get pods -n kube-system -l k8s-app=calico-node --no-headers 2>/dev/null | wc -l)

    if [[ $calico_pods -gt 0 ]]; then
        log_info "检测到 $calico_pods 个 calico-node Pod"

        # 检查是否全部 Running
        local ready_pods=$(kubectl get pods -n kube-system -l k8s-app=calico-node --no-headers 2>/dev/null | grep -c "Running" || echo "0")

        if [[ $ready_pods -eq $calico_pods ]]; then
            log_info "所有 calico-node Pod 运行正常"
        else
            log_warn "部分 calico-node Pod 未就绪"
            kubectl get pods -n kube-system -l k8s-app=calico-node
        fi
    else
        log_error "未检测到 calico-node Pod"
        exit 1
    fi

    # 检查节点网络状态
    log_info "检查节点网络状态..."
    kubectl get nodes -o wide
}

# 主函数
main() {
    local action="${1:-install}"

    case "$action" in
        install)
            log_info "开始安装 Calico CNI..."
            check_cluster
            install_with_manifest
            verify_installation
            ;;

        verify)
            verify_installation
            ;;

        *)
            echo "用法: $0 [install|verify]"
            echo ""
            echo "环境变量:"
            echo "  CALICO_VERSION        Calico 版本 (默认: 3.26.4)"
            echo "  POD_CIDR              Pod 网络 CIDR (默认: 192.168.0.0/16)"
            echo "  CALICO_MANIFEST_URL   自定义 manifest URL"
            exit 1
            ;;
    esac
}

main "$@"
