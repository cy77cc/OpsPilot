#!/usr/bin/env bash
# Kubernetes 系统参数配置脚本
set -euo pipefail

log_info() { echo "[INFO] $1"; }

log_info "配置 Kubernetes 系统参数..."

# 创建 sysctl 配置文件
cat > /etc/sysctl.d/99-kubernetes.conf <<EOF
# 网络设置
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1

# 性能优化
net.core.somaxconn                  = 32768
net.ipv4.tcp_max_syn_backlog        = 32768
net.core.netdev_max_backlog         = 32768
net.ipv4.tcp_fin_timeout            = 30
net.ipv4.tcp_keepalive_time         = 300
net.ipv4.tcp_keepalive_intvl        = 30
net.ipv4.tcp_keepalive_probes       = 3
net.ipv4.tcp_syncookies             = 1
net.ipv4.tcp_max_tw_buckets         = 65535
net.ipv4.tcp_tw_reuse               = 1

# 内存设置
vm.max_map_count                    = 262144
vm.swappiness                       = 0
vm.overcommit_memory                = 1

# 文件描述符
fs.file-max                         = 2097152
fs.inotify.max_user_instances       = 8192
fs.inotify.max_user_watches         = 524288
fs.inotify.max_queued_events        = 16384

# 进程数
kernel.pid_max                      = 4194303
EOF

# 加载 br_netfilter 模块
modprobe br_netfilter 2>/dev/null || true

# 应用配置
sysctl --system

log_info "系统参数配置完成"

# 显示当前配置
echo ""
echo "当前关键配置:"
sysctl net.bridge.bridge-nf-call-iptables
sysctl net.bridge.bridge-nf-call-ip6tables
sysctl net.ipv4.ip_forward
