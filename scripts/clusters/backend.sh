#!/usr/bin/env bash
# scripts/clusters/backend.sh — 集群B：后端业务服务
# migrate(job) + rpc×5 + gateway×2
#
# 用法：
#   ./scripts/clusters/backend.sh            # 构建镜像 + 启动
#   ./scripts/clusters/backend.sh build      # 只构建镜像（供并行调度用）
#   ./scripts/clusters/backend.sh up         # 只重启（镜像已存在）
#   ./scripts/clusters/backend.sh build --no-cache

set -euo pipefail
source "$(dirname "$0")/_common.sh"

SERVICES=(
    migrate
    user-rpc product-rpc order-rpc seckill-rpc admin-rpc
    user-gateway admin-gateway
)
# gateway 是最后一个 healthy 的，等它们就够了
HEALTH_CONTAINERS=(flashsale-app-user-gateway-1 flashsale-app-admin-gateway-1)

echo -e "${C_BOLD}=== [Cluster: backend] ===${C_RESET}"

CMD="${1:-all}"
shift || true   # 剩余参数（如 --no-cache）透传给 build

case "$CMD" in
    build)
        step "BUILD" "Building flashsale-backend:local (parallel Go compile inside Docker)"
        # backend-image 是 profile=build 的虚拟服务，专门用于构建+缓存镜像
        dc_build "$@" backend-image
        ok "Backend image built"
        ;;
    up)
        step "UP" "Deploying backend services"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for gateways to become healthy..."
        wait_healthy 120 "${HEALTH_CONTAINERS[@]}" || { fail "Backend health check failed"; exit 1; }
        ok "Backend cluster ready"
        ;;
    all)
        step "BUILD" "Building flashsale-backend:local"
        dc_build "$@" backend-image

        step "UP" "Deploying backend services"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for gateways to become healthy..."
        wait_healthy 120 "${HEALTH_CONTAINERS[@]}" || { fail "Backend health check failed"; exit 1; }
        ok "Backend cluster ready"
        ;;
    *)
        echo "Usage: $0 [all|build|up] [--no-cache]"
        exit 1
        ;;
esac
