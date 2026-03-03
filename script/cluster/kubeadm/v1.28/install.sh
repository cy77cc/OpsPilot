#!/usr/bin/env bash
# Kubernetes 组件安装脚本 (kubeadm, kubelet, kubectl)
# 支持 Ubuntu/Debian 和 CentOS/RHEL 系列
set -euo pipefail

# 版本配置
KUBERNETES_VERSION="${KUBERNETES_VERSION:-1.28.0}"
KUBERNETES_PACKAGE_VERSION="${KUBERNETES_PACKAGE_VERSION:-}"
REPO_MODE="${REPO_MODE:-online}"
REPO_URL="${REPO_URL:-}"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检测操作系统
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        echo "$ID"
    else
        log_error "无法检测操作系统"
        exit 1
    fi
}

OS=$(detect_os)
log_info "检测到操作系统: $OS"

# 确定包版本
if [[ -z "$KUBERNETES_PACKAGE_VERSION" ]]; then
    case "$OS" in
        ubuntu|debian)
            KUBERNETES_PACKAGE_VERSION="${KUBERNETES_VERSION}-1.1"
            ;;
        centos|rhel|rocky|almalinux|fedora)
            KUBERNETES_PACKAGE_VERSION="${KUBERNETES_VERSION}-0"
            ;;
    esac
fi

log_info "将安装 Kubernetes 版本: ${KUBERNETES_VERSION}"

# 禁用 swap
disable_swap() {
    log_info "禁用 swap..."
    swapoff -a

    # 永久禁用
    if [[ -f /etc/fstab ]]; then
        sed -i '/swap/d' /etc/fstab
    fi
}

# 安装依赖
install_dependencies() {
    log_info "安装依赖包..."

    case "$OS" in
        ubuntu|debian)
            apt-get update
            apt-get install -y \
                ca-certificates \
                curl \
                gnupg \
                lsb-release \
                apt-transport-https
            ;;
        centos|rhel|rocky|almalinux)
            yum install -y \
                ca-certificates \
                curl \
                gnupg2 \
                yum-utils
            ;;
        fedora)
            dnf install -y \
                ca-certificates \
                curl \
                gnupg2 \
                dnf-utils
            ;;
    esac
}

# 添加 Kubernetes 仓库
add_kubernetes_repo() {
    log_info "添加 Kubernetes 仓库..."

    case "$OS" in
        ubuntu|debian)
            if [[ "$REPO_MODE" == "mirror" && -n "$REPO_URL" ]]; then
                log_info "使用 mirror 模式包仓: $REPO_URL"
                cat > /etc/apt/sources.list.d/kubernetes.list <<EOF
deb ${REPO_URL} /
EOF
                apt-get update
                return
            fi
            # 创建 keyrings 目录
            install -m 0755 -d /etc/apt/keyrings

            # 添加 Kubernetes GPG key
            curl -fsSL https://pkgs.k8s.io/core:/stable:/v${KUBERNETES_VERSION%.*}/deb/Release.key | \
                gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
            chmod 644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg

            # 添加仓库
            echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v${KUBERNETES_VERSION%.*}/deb/ /" | \
                tee /etc/apt/sources.list.d/kubernetes.list
            chmod 644 /etc/apt/sources.list.d/kubernetes.list

            apt-get update
            ;;

        centos|rhel|rocky|almalinux)
            if [[ "$REPO_MODE" == "mirror" && -n "$REPO_URL" ]]; then
                log_info "使用 mirror 模式包仓: $REPO_URL"
                cat > /etc/yum.repos.d/kubernetes.repo <<EOF
[kubernetes]
name=Kubernetes
baseurl=${REPO_URL}
enabled=1
gpgcheck=0
EOF
                return
            fi
            cat > /etc/yum.repos.d/kubernetes.repo <<EOF
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v${KUBERNETES_VERSION%.*}/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v${KUBERNETES_VERSION%.*}/rpm/repodata/repomd.xml.key
EOF
            ;;

        fedora)
            if [[ "$REPO_MODE" == "mirror" && -n "$REPO_URL" ]]; then
                log_info "使用 mirror 模式包仓: $REPO_URL"
                cat > /etc/yum.repos.d/kubernetes.repo <<EOF
[kubernetes]
name=Kubernetes
baseurl=${REPO_URL}
enabled=1
gpgcheck=0
EOF
                return
            fi
            cat > /etc/yum.repos.d/kubernetes.repo <<EOF
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v${KUBERNETES_VERSION%.*}/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v${KUBERNETES_VERSION%.*}/rpm/repodata/repomd.xml.key
EOF
            ;;
    esac
}

# 安装 Kubernetes 组件
install_kubernetes() {
    log_info "安装 kubeadm, kubelet, kubectl..."

    case "$OS" in
        ubuntu|debian)
            apt-get install -y \
                kubeadm="${KUBERNETES_PACKAGE_VERSION}" \
                kubelet="${KUBERNETES_PACKAGE_VERSION}" \
                kubectl="${KUBERNETES_PACKAGE_VERSION}"
            ;;
        centos|rhel|rocky|almalinux)
            yum install -y \
                kubeadm-"${KUBERNETES_PACKAGE_VERSION}" \
                kubelet-"${KUBERNETES_PACKAGE_VERSION}" \
                kubectl-"${KUBERNETES_PACKAGE_VERSION}"
            ;;
        fedora)
            dnf install -y \
                kubeadm-"${KUBERNETES_PACKAGE_VERSION}" \
                kubelet-"${KUBERNETES_PACKAGE_VERSION}" \
                kubectl-"${KUBERNETES_PACKAGE_VERSION}"
            ;;
    esac

    # 锁定版本 (Ubuntu/Debian)
    if [[ "$OS" == "ubuntu" || "$OS" == "debian" ]]; then
        apt-mark hold kubeadm kubelet kubectl
    fi
}

# 配置 kubelet
configure_kubelet() {
    log_info "配置 kubelet..."

    # 创建 kubelet 配置目录
    mkdir -p /etc/default

    # 创建 kubelet 默认配置
    cat > /etc/default/kubelet <<EOF
KUBELET_EXTRA_ARGS="--cgroup-driver=systemd"
EOF

    # 启用 kubelet 服务
    systemctl enable kubelet
}

# 验证安装
verify_installation() {
    log_info "验证安装..."

    local success=true

    if command -v kubeadm &>/dev/null; then
        log_info "kubeadm: $(kubeadm version -o short 2>/dev/null || echo 'installed')"
    else
        log_error "kubeadm 未安装"
        success=false
    fi

    if command -v kubelet &>/dev/null; then
        log_info "kubelet: $(kubelet --version 2>/dev/null || echo 'installed')"
    else
        log_error "kubelet 未安装"
        success=false
    fi

    if command -v kubectl &>/dev/null; then
        log_info "kubectl: $(kubectl version --client -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*"' | cut -d'"' -f4 || echo 'installed')"
    else
        log_error "kubectl 未安装"
        success=false
    fi

    if [[ "$success" == "true" ]]; then
        log_info "Kubernetes 组件安装完成!"
    else
        log_error "安装验证失败"
        exit 1
    fi
}

# 卸载函数
uninstall() {
    log_info "卸载 Kubernetes 组件..."

    systemctl stop kubelet || true

    case "$OS" in
        ubuntu|debian)
            apt-mark unhold kubeadm kubelet kubectl || true
            apt-get remove -y kubeadm kubelet kubectl || true
            apt-get autoremove -y
            ;;
        centos|rhel|rocky|almalinux|fedora)
            yum remove -y kubeadm kubelet kubectl || true
            ;;
    esac

    rm -f /etc/apt/sources.list.d/kubernetes.list
    rm -f /etc/yum.repos.d/kubernetes.repo
    rm -f /etc/apt/keyrings/kubernetes-apt-keyring.gpg

    log_info "Kubernetes 组件已卸载"
}

# 主函数
main() {
    local action="${1:-install}"

    case "$action" in
        install)
            log_info "开始安装 Kubernetes ${KUBERNETES_VERSION}..."
            disable_swap
            install_dependencies
            add_kubernetes_repo
            install_kubernetes
            configure_kubelet
            verify_installation
            ;;
        uninstall)
            uninstall
            ;;
        *)
            echo "用法: $0 [install|uninstall]"
            echo ""
            echo "环境变量:"
            echo "  KUBERNETES_VERSION    Kubernetes 版本 (默认: 1.28.0)"
            exit 1
            ;;
    esac
}

main "$@"
