#!/usr/bin/env bash
# scripts/clusters/proxy.sh — 集群C：反向代理 / CDN / 文件管理
# nginx (反向代理) + cdn (静态资源) + media-store (文件管理)
#
# ⚠️  nginx depends_on user-gateway + admin-gateway，
#     因此 proxy up 必须在 backend up 完成之后。
#
# 用法：
#   ./scripts/clusters/proxy.sh            # 构建镜像 + 启动
#   ./scripts/clusters/proxy.sh build      # 只构建 media-store 镜像
#   ./scripts/clusters/proxy.sh up         # 只重启
#   ./scripts/clusters/proxy.sh build --no-cache

set -euo pipefail
source "$(dirname "$0")/_common.sh"

SERVICES=(cdn media-store nginx)
CONTAINERS=(flashsale-app-cdn flashsale-app-media-store flashsale-app-nginx)

# media-store 容器以 uid=1000(app) 运行，但 bind-mount 的宿主机目录可能是 root 所有。
# Dockerfile 中的 chown 对 bind-mount 无效，因此在启动前修正宿主机目录权限。
CDN_ASSETS_DIR="$(cd "$(dirname "$0")/../.." && pwd)/cdn/assets"
ensure_cdn_writable() {
    if [ -d "$CDN_ASSETS_DIR" ]; then
        local owner
        owner=$(stat -c '%u' "$CDN_ASSETS_DIR" 2>/dev/null || echo "unknown")
        if [ "$owner" != "1000" ]; then
            step "FIX" "Fixing CDN assets ownership (current uid=$owner, need 1000)"
            sudo chown -R 1000:1000 "$CDN_ASSETS_DIR" 2>/dev/null \
                || warn "chown failed — media-store may lack write permission"
        fi
    fi
}

echo -e "${C_BOLD}=== [Cluster: proxy] ===${C_RESET}"

CMD="${1:-all}"
shift || true

case "$CMD" in
    build)
        step "BUILD" "Building media-store image"
        # cdn + nginx 使用官方 nginx 镜像，无需构建
        dc_build "$@" media-store
        ok "media-store image built"
        ;;
    up)
        ensure_cdn_writable
        step "UP" "Starting nginx + cdn + media-store"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for proxy cluster to become healthy..."
        wait_healthy 60 "${CONTAINERS[@]}" || { fail "Proxy health check failed"; exit 1; }
        ok "Proxy cluster ready"
        ;;
    all)
        step "BUILD" "Building media-store image"
        dc_build "$@" media-store

        ensure_cdn_writable
        step "UP" "Starting nginx + cdn + media-store"
        dc_up "${SERVICES[@]}"

        step "HC" "Waiting for proxy cluster to become healthy..."
        wait_healthy 60 "${CONTAINERS[@]}" || { fail "Proxy health check failed"; exit 1; }
        ok "Proxy cluster ready"
        ;;
    *)
        echo "Usage: $0 [all|build|up] [--no-cache]"
        exit 1
        ;;
esac
