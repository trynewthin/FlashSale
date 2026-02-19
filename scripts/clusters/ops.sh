#!/usr/bin/env bash
# scripts/clusters/ops.sh — 集群D：Ops 控制面
# ops-control（embed 前端，挂载 docker.sock）
#
# 注意：ops-control 的 Dockerfile 会自行构建前端并编译 fs 二进制，
# 同时依赖 flashsale-backend:local 提供其他业务二进制和配置。
#
# 用法：
#   ./scripts/clusters/ops.sh            # 构建镜像 + 启动
#   ./scripts/clusters/ops.sh build      # 只构建镜像
#   ./scripts/clusters/ops.sh up         # 只重启（镜像已存在）
#   ./scripts/clusters/ops.sh hot        # 热更新：本地编译 + docker cp（最快，~15s）
#   ./scripts/clusters/ops.sh build --no-cache

set -euo pipefail
source "$(dirname "$0")/_common.sh"

CONTAINER="flashsale-app-ops-control"
FRONTEND_DIR="$REPO_ROOT/frontend/ops"
WEB_DIR="$REPO_ROOT/cmd/fs/internal/ops/web"

echo -e "${C_BOLD}=== [Cluster: ops] ===${C_RESET}"

# _build_frontend_local: 仅供 hot 模式使用（本地开发环境）
_build_frontend_local() {
    step "FRONTEND" "Building ops frontend (local)"
    if command -v bun &>/dev/null; then
        (cd "$FRONTEND_DIR" && bun run build)
    elif command -v npm &>/dev/null; then
        (cd "$FRONTEND_DIR" && npm install && npm run build)
    else
        fail "hot mode requires bun or npm on host"; exit 1
    fi
    info "Syncing dist -> cmd/fs/internal/ops/web"
    rm -rf "$WEB_DIR"
    cp -r "$FRONTEND_DIR/dist" "$WEB_DIR"
    ok "Frontend built"
}

CMD="${1:-all}"
shift || true

case "$CMD" in
    build)
        step "BUILD" "Building ops-control image (frontend + fs built inside Docker)"
        dc_build "$@" ops-control
        ok "ops-control image built"
        ;;
    up)
        step "UP" "Starting ops-control"
        dc_up --force-recreate ops-control

        step "HC" "Waiting for ops-control to become healthy..."
        wait_healthy 60 "$CONTAINER" || { fail "ops-control health check failed"; exit 1; }
        ok "Ops cluster ready"
        ;;
    all)
        step "BUILD" "Building ops-control image (frontend + fs built inside Docker)"
        dc_build "$@" ops-control

        step "UP" "Starting ops-control"
        dc_up --force-recreate ops-control

        step "HC" "Waiting for ops-control to become healthy..."
        wait_healthy 60 "$CONTAINER" || { fail "ops-control health check failed"; exit 1; }
        ok "Ops cluster ready"
        ;;
    hot)
        # 热更新：本地交叉编译 + docker cp，跳过 Docker 镜像重建
        # 适合只改了 Go 代码或前端，不想等完整 docker build 的场景
        _build_frontend_local

        step "COMPILE" "Cross-compiling fs binary (linux/amd64)"
        TMP_BIN="$REPO_ROOT/.tmp-fs-bin"
        GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
            go build -buildvcs=false -trimpath -o "$TMP_BIN" "$REPO_ROOT/cmd/fs"
        ok "Binary compiled"

        step "COPY" "Replacing binary in container"
        docker cp "$TMP_BIN" "${CONTAINER}:/app/bin/fs"
        rm -f "$TMP_BIN"

        step "RESTART" "Restarting container"
        docker restart "$CONTAINER" >/dev/null

        step "HC" "Waiting for ops-control to become healthy..."
        wait_healthy 30 "$CONTAINER" || warn "Health check timeout (may still be starting)"
        ok "Ops hot-update complete"
        ;;
    *)
        echo "Usage: $0 [all|build|up|hot] [--no-cache]"
        exit 1
        ;;
esac
