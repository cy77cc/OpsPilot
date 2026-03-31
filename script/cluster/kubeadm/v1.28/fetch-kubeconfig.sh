#!/usr/bin/env bash
# 获取 Kubernetes 集群 kubeconfig 脚本
set -euo pipefail

# 配置
OUTPUT_FILE="${OUTPUT_FILE:-/tmp/kubeconfig}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-}"  # base64 或空

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 kubeconfig 文件
check_kubeconfig() {
    local kubeconfig="/etc/kubernetes/admin.conf"

    if [[ ! -f "$kubeconfig" ]]; then
        log_error "kubeconfig 文件不存在: $kubeconfig"
        log_error "请确保此节点是控制平面节点且集群已初始化"
        exit 1
    fi

    echo "$kubeconfig"
}

# 获取 kubeconfig
get_kubeconfig() {
    local kubeconfig=$(check_kubeconfig)

    log_info "读取 kubeconfig: $kubeconfig"

    # 读取内容
    local content
    content=$(cat "$kubeconfig")

    # 修改 server 地址（如果指定）
    if [[ -n "${API_SERVER_HOST:-}" ]]; then
        log_info "修改 API Server 地址为: $API_SERVER_HOST"
        content=$(echo "$content" | sed "s/server: https:\/\/[^:]*:/server: https:\/\/${API_SERVER_HOST}:/")
    fi

    echo "$content"
}

# 输出 kubeconfig
output_kubeconfig() {
    local content="$1"

    case "${OUTPUT_FORMAT:-}" in
        base64)
            log_info "输出 Base64 编码的 kubeconfig..."
            echo "$content" | base64 -w 0
            echo ""
            ;;
        json)
            log_info "输出 JSON 格式的 kubeconfig..."
            local encoded=$(echo "$content" | base64 -w 0)
            echo "{\"kubeconfig\": \"${encoded}\"}"
            ;;
        *)
            # 直接输出到文件
            echo "$content" > "$OUTPUT_FILE"
            chmod 600 "$OUTPUT_FILE"
            log_info "kubeconfig 已保存到: $OUTPUT_FILE"

            # 同时输出到标准输出（供脚本捕获）
            echo "---KUBECONFIG_START---"
            echo "$content"
            echo "---KUBECONFIG_END---"
            ;;
    esac
}

# 验证 kubeconfig
verify_kubeconfig() {
    local kubeconfig="$1"

    log_info "验证 kubeconfig..."

    # 检查格式
    if ! echo "$kubeconfig" | grep -q "kind: Config"; then
        log_error "kubeconfig 格式无效"
        exit 1
    fi

    # 检查必要的字段
    if ! echo "$kubeconfig" | grep -q "server:"; then
        log_error "kubeconfig 缺少 server 配置"
        exit 1
    fi

    log_info "kubeconfig 格式验证通过"
}

# 主函数
main() {
    log_info "获取 Kubernetes kubeconfig..."

    local kubeconfig
    kubeconfig=$(get_kubeconfig)

    verify_kubeconfig "$kubeconfig"
    output_kubeconfig "$kubeconfig"

    log_info "完成!"
}

main "$@"
