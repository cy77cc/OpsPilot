#!/usr/bin/env bash
# Kubernetes 控制平面初始化脚本 (kubeadm init)
set -euo pipefail

# 配置参数 (可通过环境变量覆盖)
K8S_VERSION="${K8S_VERSION:-1.28.0}"
POD_CIDR="${POD_CIDR:-10.244.0.0/16}"
SERVICE_CIDR="${SERVICE_CIDR:-10.96.0.0/12}"
CONTROL_PLANE_ENDPOINT="${CONTROL_PLANE_ENDPOINT:-}"
ADVERTISE_ADDRESS="${ADVERTISE_ADDRESS:-}"
CRI_SOCKET="${CRI_SOCKET:-/run/containerd/containerd.sock}"
KUBECONFIG_OUTPUT="${KUBECONFIG_OUTPUT:-/etc/kubernetes/admin.conf}"
BOOTSTRAP_INIT_MODE="${BOOTSTRAP_INIT_MODE:-legacy}"
KUBEADM_CONFIG_B64="${KUBEADM_CONFIG_B64:-}"

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
    log_error "kubeadm 未安装，请先运行 install.sh"
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
        log_error "未检测到容器运行时，请先安装 containerd"
        exit 1
    fi
}

# 获取本机 IP
get_local_ip() {
    local ip=""
    # 尝试获取默认路由的 IP
    ip=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
    if [[ -z "$ip" ]]; then
        # 回退方案
        ip=$(hostname -I | awk '{print $1}')
    fi
    echo "$ip"
}

# 构建 kubeadm init 参数
build_init_args() {
    local args=(
        "--kubernetes-version=v${K8S_VERSION}"
        "--pod-network-cidr=${POD_CIDR}"
        "--service-cidr=${SERVICE_CIDR}"
        "--cri-socket=unix://${CRI_SOCKET}"
        "--ignore-preflight-errors=Swap"
        "--upload-certs"
    )

    # 控制平面端点
    if [[ -n "$CONTROL_PLANE_ENDPOINT" ]]; then
        args+=("--control-plane-endpoint=${CONTROL_PLANE_ENDPOINT}")
    fi

    # 广告地址
    if [[ -n "$ADVERTISE_ADDRESS" ]]; then
        args+=("--apiserver-advertise-address=${ADVERTISE_ADDRESS}")
    else
        # 自动获取本机 IP
        local local_ip=$(get_local_ip)
        if [[ -n "$local_ip" ]]; then
            args+=("--apiserver-advertise-address=${local_ip}")
            log_info "使用自动检测的 IP: $local_ip"
        fi
    fi

    echo "${args[@]}"
}

# 执行 kubeadm init
run_kubeadm_init() {
    log_info "执行 kubeadm init..."
    log_info "参数:"
    log_info "  Kubernetes 版本: v${K8S_VERSION}"
    log_info "  Pod CIDR: ${POD_CIDR}"
    log_info "  Service CIDR: ${SERVICE_CIDR}"

    if [[ "$BOOTSTRAP_INIT_MODE" == "config" && -n "$KUBEADM_CONFIG_B64" ]]; then
        local cfg_file="/tmp/kubeadm-init-config.yaml"
        echo "$KUBEADM_CONFIG_B64" | base64 -d > "$cfg_file"
        prepare_external_etcd_materials
        log_info "命令: kubeadm init --config ${cfg_file}"
        kubeadm init --config "${cfg_file}" --upload-certs 2>&1 | tee /var/log/kubeadm-init.log
    else
        local args=$(build_init_args)
        log_info "命令: kubeadm init ${args}"
        kubeadm init ${args} 2>&1 | tee /var/log/kubeadm-init.log
    fi

    if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
        log_error "kubeadm init 失败"
        cat /var/log/kubeadm-init.log
        exit 1
    fi
}

prepare_external_etcd_materials() {
    if [[ "${ETCD_MODE:-stacked}" != "external" ]]; then
        return
    fi
    if [[ -z "${EXTERNAL_ETCD_JSON:-}" ]]; then
        return
    fi
    mkdir -p /etc/kubernetes/pki/external-etcd

    local ca cert key
    ca=$(echo "$EXTERNAL_ETCD_JSON" | sed -n 's/.*"ca_cert":"\([^"]*\)".*/\1/p')
    cert=$(echo "$EXTERNAL_ETCD_JSON" | sed -n 's/.*"cert":"\([^"]*\)".*/\1/p')
    key=$(echo "$EXTERNAL_ETCD_JSON" | sed -n 's/.*"key":"\([^"]*\)".*/\1/p')

    if [[ -n "$ca" && -n "$cert" && -n "$key" ]]; then
        printf '%s' "$ca" | sed 's/\\n/\n/g' > /etc/kubernetes/pki/external-etcd/ca.crt
        printf '%s' "$cert" | sed 's/\\n/\n/g' > /etc/kubernetes/pki/external-etcd/client.crt
        printf '%s' "$key" | sed 's/\\n/\n/g' > /etc/kubernetes/pki/external-etcd/client.key
        chmod 600 /etc/kubernetes/pki/external-etcd/client.key
    fi
}

# 配置 kubectl
configure_kubectl() {
    log_info "配置 kubectl..."

    local kubeconfig="/etc/kubernetes/admin.conf"

    if [[ ! -f "$kubeconfig" ]]; then
        log_error "kubeconfig 文件不存在: $kubeconfig"
        exit 1
    fi

    # 为 root 用户配置
    mkdir -p /root/.kube
    cp -f "$kubeconfig" /root/.kube/config
    chmod 600 /root/.kube/config

    # 为指定用户配置 (如果设置了环境变量)
    if [[ -n "${SUDO_USER:-}" ]]; then
        local user_home=$(eval echo ~$SUDO_USER)
        mkdir -p "$user_home/.kube"
        cp -f "$kubeconfig" "$user_home/.kube/config"
        chown -R "$SUDO_USER:$SUDO_USER" "$user_home/.kube"
        log_info "kubectl 已为用户 $SUDO_USER 配置"
    fi

    # 复制到指定输出位置
    if [[ "$KUBECONFIG_OUTPUT" != "/etc/kubernetes/admin.conf" ]]; then
        cp -f "$kubeconfig" "$KUBECONFIG_OUTPUT"
        log_info "kubeconfig 已复制到: $KUBECONFIG_OUTPUT"
    fi
}

# 生成并保存 join 命令
save_join_command() {
    log_info "生成 join 命令..."

    # 获取 join 命令
    local join_cmd=$(kubeadm token create --print-join-command 2>/dev/null)

    if [[ -n "$join_cmd" ]]; then
        echo "$join_cmd" > /tmp/kubeadm-join.sh
        chmod +x /tmp/kubeadm-join.sh
        log_info "Join 命令已保存到 /tmp/kubeadm-join.sh"

        echo ""
        echo "============================================"
        echo "Worker 节点可以使用以下命令加入集群:"
        echo "============================================"
        echo "$join_cmd"
        echo "============================================"
    else
        log_warn "无法生成 join 命令"
    fi

    # 生成 control-plane join 命令 (用于多控制平面)
    local cp_cert_key=$(kubeadm init phase upload-certs --upload-certs 2>/dev/null | grep -oP 'Using certificate key:\s*\K.*' || true)
    if [[ -n "$cp_cert_key" ]]; then
        echo ""
        echo "Control Plane 加入命令:"
        echo "$join_cmd --control-plane --certificate-key $cp_cert_key"
        echo ""
        echo "Certificate Key: $cp_cert_key"
        echo "$cp_cert_key" > /tmp/kubeadm-cert-key.txt
    fi
}

# 移除 master 污点 (单节点集群可选)
remove_master_taint() {
    if [[ "${REMOVE_MASTER_TAINT:-false}" == "true" ]]; then
        log_info "移除 master 污点，允许调度 Pod 到控制平面节点..."
        kubectl taint nodes --all node-role.kubernetes.io/control-plane- 2>/dev/null || true
        kubectl taint nodes --all node-role.kubernetes.io/master- 2>/dev/null || true
    fi
}

# 验证集群状态
verify_cluster() {
    log_info "验证集群状态..."

    # 等待 API Server 就绪
    log_info "等待 API Server 就绪..."
    local max_retries=30
    local retry=0

    while [[ $retry -lt $max_retries ]]; do
        if kubectl cluster-info &>/dev/null; then
            break
        fi
        sleep 2
        ((retry++))
    done

    if [[ $retry -eq $max_retries ]]; then
        log_error "API Server 未能在指定时间内就绪"
        exit 1
    fi

    # 显示集群信息
    echo ""
    echo "============================================"
    echo "           集群信息"
    echo "============================================"
    kubectl cluster-info

    echo ""
    echo "节点状态:"
    kubectl get nodes -o wide

    echo ""
    echo "组件状态:"
    kubectl get cs 2>/dev/null || kubectl get componentstatuses 2>/dev/null || true
}

# 主函数
main() {
    local action="${1:-init}"

    case "$action" in
        init)
            log_info "开始初始化 Kubernetes 控制平面..."
            check_container_runtime
            run_kubeadm_init
            configure_kubectl
            save_join_command
            remove_master_taint
            verify_cluster
            log_info "控制平面初始化完成!"
            ;;

        reset)
            log_warn "执行 kubeadm reset..."
            kubeadm reset -f
            rm -rf /root/.kube /etc/cni/net.d
            log_info "集群已重置"
            ;;

        *)
            echo "用法: $0 [init|reset]"
            echo ""
            echo "环境变量:"
            echo "  K8S_VERSION              Kubernetes 版本 (默认: 1.28.0)"
            echo "  POD_CIDR                 Pod 网络 CIDR (默认: 10.244.0.0/16)"
            echo "  SERVICE_CIDR             Service 网络 CIDR (默认: 10.96.0.0/12)"
            echo "  CONTROL_PLANE_ENDPOINT   控制平面端点 (负载均衡器地址)"
            echo "  ADVERTISE_ADDRESS        API Server 广告地址"
            echo "  REMOVE_MASTER_TAINT      移除 master 污点 (true/false)"
            exit 1
            ;;
    esac
}

main "$@"
