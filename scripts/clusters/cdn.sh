#!/usr/bin/env bash
# scripts/clusters/cdn.sh — 集群C：CDN + Media Store
# cdn (nginx 只读) + media-store (文件管理，读写)
# 两者共享 volume：deploy/cdn/assets
#
# 用法：
#   ./scripts/clusters/cdn.sh            # 构建镜像 + 启动
#   ./scripts/clusters/cdn.sh build      # 只构建 media-store 镜像
#   ./scripts/clusters/cdn.sh up         # 只重启
#   ./scripts/clusters/cdn.sh build --no-cache

set -euo pipefail
source "$(dirname "$0")/_common.sh"

SERVICES=(cdn media-store)
CONTAINERS=(flashsale-app-cdn flashsale-app-media-store)

echo -e "${C_BOLD}=== [Cluster: cdn] ===${C_RESET}"

CMD="${1:-all}"
shift || true

case "$CMD" in
    build)
        step "BUILD" "Building media-store image"
        # cdn 使用官方 nginx 镜像，无需构建
        dc_build "$@" media-store
        ok "media-store image built"
        ;;
    up)
        step "UP" "Starting cdn + media-store"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for cdn + media-store to become healthy..."
        wait_healthy 60 "${CONTAINERS[@]}" || { fail "CDN health check failed"; exit 1; }
        ok "CDN cluster ready"
        ;;
    all)
        step "BUILD" "Building media-store image"
        dc_build "$@" media-store

        step "UP" "Starting cdn + media-store"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for cdn + media-store to become healthy..."
        wait_healthy 60 "${CONTAINERS[@]}" || { fail "CDN health check failed"; exit 1; }
        ok "CDN cluster ready"
        ;;
    *)
        echo "Usage: $0 [all|build|up] [--no-cache]"
        exit 1
        ;;
esac
