#!/usr/bin/env bash
# Containerd 安装脚本
# 支持 Ubuntu/Debian 和 CentOS/RHEL 系列
set -euo pipefail

# 配置
CONTAINERD_VERSION="${CONTAINERD_VERSION:-1.7.13}"
RUNC_VERSION="${RUNC_VERSION:-1.1.12}"
CNI_PLUGINS_VERSION="${CNI_PLUGINS_VERSION:-1.4.0}"

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
                apt-transport-https \
                software-properties-common
            ;;
        centos|rhel|rocky|almalinux|fedora)
            yum install -y \
                ca-certificates \
                curl \
                gnupg2 \
                lsb-release \
                yum-utils
            ;;
        *)
            log_error "不支持的操作系统: $OS"
            exit 1
            ;;
    esac
}

# 配置内核模块
configure_kernel_modules() {
    log_info "配置内核模块..."

    cat > /etc/modules-load.d/containerd.conf <<EOF
overlay
br_netfilter
EOF

    modprobe overlay
    modprobe br_netfilter

    log_info "内核模块已配置"
}

# 配置 sysctl
configure_sysctl() {
    log_info "配置 sysctl 参数..."

    cat > /etc/sysctl.d/99-kubernetes-cri.conf <<EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

    sysctl --system
    log_info "sysctl 参数已配置"
}

# 安装 containerd (使用包管理器)
install_containerd_package() {
    log_info "通过包管理器安装 containerd..."

    case "$OS" in
        ubuntu|debian)
            # 添加 Docker 官方仓库
            install -m 0755 -d /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/$ID/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            chmod a+r /etc/apt/keyrings/docker.gpg

            echo \
                "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$ID \
                $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
                tee /etc/apt/sources.list.d/docker.list > /dev/null

            apt-get update
            apt-get install -y containerd.io
            ;;

        centos|rhel|rocky|almalinux)
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y containerd.io
            ;;

        fedora)
            dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
            dnf install -y containerd.io
            ;;
    esac
}

# 配置 containerd
configure_containerd() {
    log_info "配置 containerd..."

    # 创建配置目录
    mkdir -p /etc/containerd

    # 生成默认配置
    containerd config default > /etc/containerd/config.toml

    # 配置 SystemdCgroup = true
    sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

    # 配置镜像加速 (可选)
    if [[ -n "${REGISTRY_MIRROR:-}" ]]; then
        log_info "配置镜像加速: $REGISTRY_MIRROR"
        # 在配置文件中添加 mirror 配置
        # 这里简化处理，实际需要更复杂的 TOML 操作
    fi

    log_info "containerd 配置完成"
}

# 安装 runc (如果需要)
install_runc() {
    if ! command -v runc &>/dev/null; then
        log_info "安装 runc..."

        case "$OS" in
            ubuntu|debian)
                apt-get install -y runc
                ;;
            centos|rhel|rocky|almalinux|fedora)
                yum install -y runc
                ;;
        esac
    else
        log_info "runc 已安装: $(runc --version | head -1)"
    fi
}

# 安装 CNI 插件
install_cni_plugins() {
    log_info "检查 CNI 插件..."

    local cni_bin_dir="/opt/cni/bin"
    local cni_version="${CNI_VERSION:-$CNI_PLUGINS_VERSION}"

    if [[ -f "$cni_bin_dir/bridge" ]]; then
        log_info "CNI 插件已存在"
        return
    fi

    log_info "安装 CNI 插件 v${cni_version}..."

    mkdir -p "$cni_bin_dir"

    local arch=$(uname -m)
    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64) arch="arm64" ;;
    esac

    local download_url="https://github.com/containernetworking/plugins/releases/download/v${cni_version}/cni-plugins-linux-${arch}-v${cni_version}.tgz"

    curl -sSL "$download_url" | tar -xzf - -C "$cni_bin_dir"

    log_info "CNI 插件安装完成"
}

# 启动服务
start_service() {
    log_info "启动 containerd 服务..."

    systemctl daemon-reload
    systemctl enable containerd
    systemctl start containerd

    # 等待服务启动
    sleep 2

    if systemctl is-active containerd; then
        log_info "containerd 服务已启动"
    else
        log_error "containerd 服务启动失败"
        journalctl -u containerd --no-pager -n 20
        exit 1
    fi
}

# 验证安装
verify_installation() {
    log_info "验证安装..."

    # 检查 containerd
    if command -v containerd &>/dev/null; then
        local version=$(containerd --version | awk '{print $3}')
        log_info "containerd 版本: $version"
    else
        log_error "containerd 未正确安装"
        exit 1
    fi

    # 检查 runc
    if command -v runc &>/dev/null; then
        local version=$(runc --version | head -1 | awk '{print $2}')
        log_info "runc 版本: $version"
    else
        log_error "runc 未正确安装"
        exit 1
    fi

    # 测试运行容器
    log_info "测试运行容器..."
    if ctr images pull docker.io/library/alpine:latest >/dev/null 2>&1; then
        log_info "镜像拉取测试成功"
        ctr images rm docker.io/library/alpine:latest >/dev/null 2>&1 || true
    else
        log_warn "镜像拉取测试失败，可能是网络问题"
    fi

    log_info "安装验证完成"
}

# 卸载函数 (供回滚使用)
uninstall() {
    log_info "卸载 containerd..."

    systemctl stop containerd || true
    systemctl disable containerd || true

    case "$OS" in
        ubuntu|debian)
            apt-get remove -y containerd.io || true
            ;;
        centos|rhel|rocky|almalinux|fedora)
            yum remove -y containerd.io || true
            ;;
    esac

    rm -rf /etc/containerd
    rm -f /etc/modules-load.d/containerd.conf
    rm -f /etc/sysctl.d/99-kubernetes-cri.conf

    log_info "containerd 已卸载"
}

# 主函数
main() {
    local action="${1:-install}"

    case "$action" in
        install)
            log_info "开始安装 containerd..."
            install_dependencies
            configure_kernel_modules
            configure_sysctl
            install_containerd_package
            install_runc
            configure_containerd
            install_cni_plugins
            start_service
            verify_installation
            log_info "containerd 安装完成!"
            ;;
        uninstall)
            uninstall
            ;;
        *)
            echo "用法: $0 [install|uninstall]"
            exit 1
            ;;
    esac
}

main "$@"
