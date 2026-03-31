#!/usr/bin/env bash
# Cilium CNI 安装脚本
set -euo pipefail

# 配置
CILIUM_VERSION="${CILIUM_VERSION:-1.14.5}"
POD_CIDR="${POD_CIDR:-10.0.0.0/8}"
INSTALL_METHOD="${INSTALL_METHOD:-helm}"
ENABLE_HUBBLE="${ENABLE_HUBBLE:-true}"
KUBE_PROXY_REPLACEMENT="${KUBE_PROXY_REPLACEMENT:-false}"

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

# 检查内核版本
check_kernel_version() {
    log_info "检查内核版本..."

    local kernel_version=$(uname -r | cut -d'-' -f1)
    local major=$(echo "$kernel_version" | cut -d'.' -f1)
    local minor=$(echo "$kernel_version" | cut -d'.' -f2)

    if [[ $major -lt 4 ]] || [[ $major -eq 4 && $minor -lt 19 ]]; then
        log_error "内核版本过低: $kernel_version, 需要至少 4.19"
        log_error "建议使用 5.10+ 内核以获得最佳性能"
        exit 1
    fi

    if [[ $major -lt 5 ]]; then
        log_warn "内核版本 $kernel_version 可用，但建议升级到 5.10+ 以获得最佳性能"
    else
        log_info "内核版本: $kernel_version"
    fi
}

# 检查集群连接
check_cluster() {
    log_info "检查集群连接..."

    if ! kubectl cluster-info &>/dev/null; then
        log_error "无法连接到 Kubernetes 集群"
        exit 1
    fi

    log_info "集群连接正常"
}

# 使用 Helm 安装
install_with_helm() {
    log_info "使用 Helm 安装 Cilium v${CILIUM_VERSION}..."

    # 检查 Helm
    if ! command -v helm &>/dev/null; then
        log_info "Helm 未安装，尝试安装..."
        curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    fi

    # 添加 Cilium Helm repo
    helm repo add cilium https://helm.cilium.io/
    helm repo update

    # 创建命名空间
    kubectl create namespace cilium --dry-run=client -o yaml | kubectl apply -f -

    # 构建 Helm values
    local helm_values=(
        "--set image.tag=v${CILIUM_VERSION}"
        "--set ipam.mode=kubernetes"
        "--set operator.replicas=1"
    )

    # Pod CIDR
    if [[ "$POD_CIDR" != "10.0.0.0/8" ]]; then
        helm_values+=("--set ipam.operator.clusterPoolIPv4PodCIDR=${POD_CIDR}")
    fi

    # Hubble
    if [[ "$ENABLE_HUBBLE" == "true" ]]; then
        helm_values+=("--set hubble.enabled=true")
        helm_values+=("--set hubble.relay.enabled=true")
        helm_values+=("--set hubble.ui.enabled=true")
        log_info "启用 Hubble 可观测性"
    fi

    # Kube-proxy replacement
    if [[ "$KUBE_PROXY_REPLACEMENT" == "true" ]]; then
        helm_values+=("--set kubeProxyReplacement=true")
        helm_values+=("--set k8sServiceHost=127.0.0.1")
        helm_values+=("--set k8sServicePort=6443")
        log_info "启用 kube-proxy 替换模式"
    else
        helm_values+=("--set kubeProxyReplacement=false")
    fi

    # 安装
    log_info "执行 Helm 安装..."
    helm upgrade --install cilium cilium/cilium --version "${CILIUM_VERSION}" \
        --namespace cilium \
        "${helm_values[@]}"

    # 等待就绪
    wait_for_cilium
}

# 等待 Cilium 就绪
wait_for_cilium() {
    log_info "等待 Cilium 组件就绪..."

    # 等待 DaemonSet
    kubectl rollout status daemonset/cilium -n cilium --timeout=300s

    # 等待 Deployment
    kubectl rollout status deployment/cilium-operator -n cilium --timeout=120s

    if [[ "$ENABLE_HUBBLE" == "true" ]]; then
        kubectl rollout status deployment/hubble-relay -n cilium --timeout=120s || true
    fi

    # 检查 Pod 状态
    log_info "检查 Cilium Pod 状态..."
    kubectl get pods -n cilium

    echo ""
    log_info "Cilium 安装完成!"
}

# 验证安装
verify_installation() {
    log_info "验证 Cilium 安装..."

    # 检查 cilium Pod
    local cilium_pods=$(kubectl get pods -n cilium -l app.kubernetes.io/name=cilium --no-headers 2>/dev/null | wc -l)

    if [[ $cilium_pods -gt 0 ]]; then
        log_info "检测到 $cilium_pods 个 Cilium Pod"

        # 使用 cilium status 命令（如果可用）
        if kubectl exec -n cilium daemonset/cilium -- cilium status &>/dev/null; then
            log_info "Cilium 状态:"
            kubectl exec -n cilium daemonset/cilium -- cilium status
        fi
    else
        log_warn "未检测到 Cilium Pod"
    fi
}

# 主函数
main() {
    local action="${1:-install}"

    case "$action" in
        install)
            log_info "开始安装 Cilium CNI..."
            check_kernel_version
            check_cluster
            install_with_helm
            verify_installation
            ;;

        verify)
            verify_installation
            ;;

        *)
            echo "用法: $0 [install|verify]"
            echo ""
            echo "环境变量:"
            echo "  CILIUM_VERSION           Cilium 版本 (默认: 1.14.5)"
            echo "  POD_CIDR                 Pod 网络 CIDR (默认: 10.0.0.0/8)"
            echo "  ENABLE_HUBBLE            启用 Hubble (默认: true)"
            echo "  KUBE_PROXY_REPLACEMENT   替换 kube-proxy (默认: false)"
            exit 1
            ;;
    esac
}

main "$@"
