#!/usr/bin/env bash
# Kubernetes 集群预检查脚本
# 检查系统是否满足 K8s 安装要求
set -euo pipefail

ERRORS=()
WARNINGS=()

# 颜色定义
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    WARNINGS+=("$1")
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ERRORS+=("$1")
}

# 1. 检查操作系统
check_os() {
    log_info "检查操作系统..."
    if [[ ! -f /etc/os-release ]]; then
        log_error "无法确定操作系统版本"
        return
    fi
    . /etc/os-release
    log_info "操作系统: $PRETTY_NAME"

    # 检查支持的发行版
    case "$ID" in
        ubuntu|debian|centos|rhel|rocky|almalinux|fedora)
            log_info "发行版 $ID 已支持"
            ;;
        *)
            log_warn "发行版 $ID 未经过测试，可能存在兼容性问题"
            ;;
    esac
}

# 2. 检查内存
check_memory() {
    log_info "检查内存..."
    local mem_total=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    local mem_gb=$((mem_total / 1024 / 1024))
    log_info "内存: ${mem_gb}GB"

    if [[ $mem_gb -lt 2 ]]; then
        log_error "内存不足 2GB，当前 ${mem_gb}GB"
    elif [[ $mem_gb -lt 4 ]]; then
        log_warn "内存建议至少 4GB，当前 ${mem_gb}GB"
    fi
}

# 3. 检查 CPU
check_cpu() {
    log_info "检查 CPU..."
    local cpu_cores=$(nproc)
    log_info "CPU 核心数: $cpu_cores"

    if [[ $cpu_cores -lt 2 ]]; then
        log_error "CPU 核心数不足 2，当前 $cpu_cores"
    fi
}

# 4. 检查 swap
check_swap() {
    log_info "检查 swap..."
    local swap_total=$(grep SwapTotal /proc/meminfo | awk '{print $2}')
    local swap_mb=$((swap_total / 1024))

    if [[ $swap_total -gt 0 ]]; then
        log_warn "检测到 swap 已启用 (${swap_mb}MB)，K8s 要求禁用 swap"
        echo "      执行 'swapoff -a' 禁用 swap"
        echo "      并在 /etc/fstab 中注释 swap 行以永久禁用"
    else
        log_info "Swap 已禁用"
    fi
}

# 5. 检查端口
check_ports() {
    log_info "检查端口..."

    # 控制平面端口
    local control_plane_ports=(6443 2379 2380 10250 10251 10252 10257 10259)
    # 工作节点端口
    local worker_ports=(10250 30000-32767)

    local is_control_plane="${IS_CONTROL_PLANE:-true}"

    if [[ "$is_control_plane" == "true" ]]; then
        for port in "${control_plane_ports[@]}"; do
            if ss -tuln | grep -q ":${port} "; then
                log_error "端口 $port 已被占用"
            else
                log_info "端口 $port 可用"
            fi
        done
    fi
}

# 6. 检查必要的内核模块
check_kernel_modules() {
    log_info "检查内核模块..."

    local required_modules=(br_netfilter overlay)
    local optional_modules=(ip_vs ip_vs_rr ip_vs_wrr ip_vs_sh nf_conntrack)

    for mod in "${required_modules[@]}"; do
        if lsmod | grep -q "^${mod}"; then
            log_info "内核模块 $mod 已加载"
        else
            # 尝试加载
            if modprobe "$mod" 2>/dev/null; then
                log_info "内核模块 $mod 已加载 (自动)"
            else
                log_error "内核模块 $mod 未加载，请执行: modprobe $mod"
            fi
        fi
    done

    for mod in "${optional_modules[@]}"; do
        if lsmod | grep -q "^${mod}"; then
            log_info "内核模块 $mod 已加载"
        else
            log_warn "可选内核模块 $mod 未加载 (建议加载以获得更好性能)"
        fi
    done
}

# 7. 检查 sysctl 配置
check_sysctl() {
    log_info "检查 sysctl 配置..."

    local bridge_nf_call_iptables=$(sysctl -n net.bridge.bridge-nf-call-iptables 2>/dev/null || echo "0")
    local bridge_nf_call_ip6tables=$(sysctl -n net.bridge.bridge-nf-call-ip6tables 2>/dev/null || echo "0")
    local ip_forward=$(sysctl -n net.ipv4.ip_forward)

    if [[ "$bridge_nf_call_iptables" != "1" ]]; then
        log_warn "net.bridge.bridge-nf-call-iptables 未设置为 1"
        echo "      执行: sysctl -w net.bridge.bridge-nf-call-iptables=1"
    else
        log_info "net.bridge.bridge-nf-call-iptables = 1"
    fi

    if [[ "$bridge_nf_call_ip6tables" != "1" ]]; then
        log_warn "net.bridge.bridge-nf-call-ip6tables 未设置为 1"
        echo "      执行: sysctl -w net.bridge.bridge-nf-call-ip6tables=1"
    else
        log_info "net.bridge.bridge-nf-call-ip6tables = 1"
    fi

    if [[ "$ip_forward" != "1" ]]; then
        log_warn "net.ipv4.ip_forward 未设置为 1"
        echo "      执行: sysctl -w net.ipv4.ip_forward=1"
    else
        log_info "net.ipv4.ip_forward = 1"
    fi
}

# 8. 检查容器运行时
check_container_runtime() {
    log_info "检查容器运行时..."

    # 检查 containerd
    if command -v containerd &>/dev/null; then
        local containerd_version=$(containerd --version 2>/dev/null | awk '{print $3}')
        log_info "containerd 已安装: $containerd_version"
    elif command -v docker &>/dev/null; then
        local docker_version=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
        log_info "docker 已安装: $docker_version"
        log_warn "建议使用 containerd 作为容器运行时"
    else
        log_info "未检测到容器运行时，将在安装过程中自动安装 containerd"
    fi
}

# 9. 检查时间同步
check_time_sync() {
    log_info "检查时间同步..."

    if command -v timedatectl &>/dev/null; then
        local ntp_status=$(timedatectl show --property=NTP --value 2>/dev/null || echo "unknown")
        if [[ "$ntp_status" == "yes" ]]; then
            log_info "NTP 时间同步已启用"
        else
            log_warn "NTP 时间同步未启用，建议启用以确保集群时间一致"
        fi
    fi

    # 检查 chrony 或 ntpd
    if systemctl is-active chronyd &>/dev/null; then
        log_info "chronyd 服务运行中"
    elif systemctl is-active ntpd &>/dev/null; then
        log_info "ntpd 服务运行中"
    elif systemctl is-active systemd-timesyncd &>/dev/null; then
        log_info "systemd-timesyncd 服务运行中"
    else
        log_warn "未检测到时间同步服务"
    fi
}

# 10. 检查磁盘空间
check_disk_space() {
    log_info "检查磁盘空间..."

    local root_avail=$(df -BG / | awk 'NR==2 {print $4}' | tr -d 'G')
    local var_avail=$(df -BG /var 2>/dev/null | awk 'NR==2 {print $4}' | tr -d 'G' || echo "$root_avail")

    if [[ $root_avail -lt 10 ]]; then
        log_error "根分区可用空间不足 10GB，当前 ${root_avail}GB"
    elif [[ $root_avail -lt 20 ]]; then
        log_warn "根分区可用空间建议至少 20GB，当前 ${root_avail}GB"
    else
        log_info "根分区可用空间: ${root_avail}GB"
    fi
}

# 11. 检查 hostname
check_hostname() {
    log_info "检查主机名..."

    local hostname=$(hostname)
    if [[ "$hostname" =~ [._] ]]; then
        log_warn "主机名包含特殊字符 (. 或 _)，可能影响 K8s 功能"
        echo "      建议使用小写字母和数字，用 - 分隔"
    else
        log_info "主机名: $hostname"
    fi

    # 检查 /etc/hosts
    if ! grep -q "$hostname" /etc/hosts 2>/dev/null; then
        log_warn "主机名未在 /etc/hosts 中配置"
    fi
}

# 12. 检查现有 K8s 组件
check_existing_k8s() {
    log_info "检查现有 Kubernetes 组件..."

    local has_existing=false

    if command -v kubeadm &>/dev/null; then
        log_warn "kubeadm 已安装: $(kubeadm version -o short 2>/dev/null || echo 'unknown')"
        has_existing=true
    fi

    if command -v kubelet &>/dev/null; then
        log_warn "kubelet 已安装: $(kubelet --version 2>/dev/null | awk '{print $2}' || echo 'unknown')"
        has_existing=true
    fi

    if command -v kubectl &>/dev/null; then
        log_warn "kubectl 已安装: $(kubectl version --client -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*"' | cut -d'"' -f4 || echo 'unknown')"
        has_existing=true
    fi

    if [[ "$has_existing" == "true" ]]; then
        log_warn "检测到现有 Kubernetes 组件，安装过程可能会覆盖"
    fi
}

# 输出结果
print_summary() {
    echo ""
    echo "============================================"
    echo "              预检查结果汇总"
    echo "============================================"

    if [[ ${#WARNINGS[@]} -gt 0 ]]; then
        echo ""
        echo -e "${YELLOW}警告 (${#WARNINGS[@]}):${NC}"
        for w in "${WARNINGS[@]}"; do
            echo "  - $w"
        done
    fi

    if [[ ${#ERRORS[@]} -gt 0 ]]; then
        echo ""
        echo -e "${RED}错误 (${#ERRORS[@]}):${NC}"
        for e in "${ERRORS[@]}"; do
            echo "  - $e"
        done
        echo ""
        echo -e "${RED}预检查未通过，请修复上述错误后重试${NC}"
        exit 1
    fi

    echo ""
    echo -e "${GREEN}预检查通过，可以继续安装${NC}"
    exit 0
}

# 执行所有检查
main() {
    echo "============================================"
    echo "       Kubernetes 集群预检查"
    echo "============================================"
    echo ""

    check_os
    check_memory
    check_cpu
    check_swap
    check_ports
    check_kernel_modules
    check_sysctl
    check_container_runtime
    check_time_sync
    check_disk_space
    check_hostname
    check_existing_k8s

    print_summary
}

main "$@"
